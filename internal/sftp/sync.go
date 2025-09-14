package sftp

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/termbus/termbus/internal/logger"
	"github.com/termbus/termbus/pkg/types"
	"go.uber.org/zap"
)

type SyncMode int

const (
	SyncModeSize SyncMode = iota
	SyncModeModTime
	SyncModeChecksum
)

type SyncOptions struct {
	Mode     SyncMode
	Delete   bool
	Ignore   []string
	Parallel int
}

type SyncResult struct {
	Uploaded   int
	Downloaded int
	Deleted    int
	Skipped    int
	Errors     []string
}

type SyncManager struct {
	sftpManager *SFTPManager
	resumeMgr   *ResumeManager
}

func NewSyncManager(sftpManager *SFTPManager, resumeMgr *ResumeManager) *SyncManager {
	return &SyncManager{
		sftpManager: sftpManager,
		resumeMgr:   resumeMgr,
	}
}

func (m *SyncManager) SyncUpload(sessionID, localDir, remoteDir string, opts *SyncOptions, progress chan float64) error {
	if opts == nil {
		opts = &SyncOptions{
			Mode:     SyncModeModTime,
			Parallel: 4,
		}
	}

	localFiles, err := m.scanLocalDir(localDir)
	if err != nil {
		return fmt.Errorf("failed to scan local dir: %w", err)
	}

	remoteFiles, err := m.sftpManager.List(sessionID, remoteDir)
	if err != nil && !isEmptyDirError(err) {
		return fmt.Errorf("failed to list remote dir: %w", err)
	}

	remoteMap := make(map[string]types.FileInfo)
	for _, f := range remoteFiles {
		remoteMap[f.Name] = f
	}

	var toUpload []syncFileTask
	for _, local := range localFiles {
		if m.shouldIgnore(local.name, opts.Ignore) {
			continue
		}

		remote, exists := remoteMap[local.name]
		if !exists {
			toUpload = append(toUpload, syncFileTask{
				localPath:  local.path,
				remotePath: filepath.Join(remoteDir, local.name),
				size:       local.size,
			})
			continue
		}

		needSync, err := m.needSync(local, remote, opts.Mode, sessionID)
		if err != nil {
			logger.GetLogger().Warn("failed to check sync status",
				zap.String("file", local.name),
				zap.String("error", err.Error()),
			)
			continue
		}

		if needSync {
			toUpload = append(toUpload, syncFileTask{
				localPath:  local.path,
				remotePath: filepath.Join(remoteDir, local.name),
				size:       local.size,
			})
		}
	}

	if opts.Delete {
		toDelete := m.findOrphans(localFiles, remoteFiles)
		for _, f := range toDelete {
			m.sftpManager.Delete(sessionID, filepath.Join(remoteDir, f.Name))
		}
	}

	return m.executeParallelUpload(sessionID, toUpload, opts.Parallel, progress)
}

func (m *SyncManager) SyncDownload(sessionID, remoteDir, localDir string, opts *SyncOptions, progress chan float64) error {
	if opts == nil {
		opts = &SyncOptions{
			Mode:     SyncModeModTime,
			Parallel: 4,
		}
	}

	remoteFiles, err := m.sftpManager.List(sessionID, remoteDir)
	if err != nil {
		return fmt.Errorf("failed to list remote dir: %w", err)
	}

	localFiles, err := m.scanLocalDir(localDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to scan local dir: %w", err)
	}

	localMap := make(map[string]localFileInfo)
	for _, f := range localFiles {
		localMap[f.name] = f
	}

	var toDownload []syncFileTask
	for _, remote := range remoteFiles {
		if m.shouldIgnore(remote.Name, opts.Ignore) {
			continue
		}

		local, exists := localMap[remote.Name]
		if !exists {
			toDownload = append(toDownload, syncFileTask{
				localPath:  filepath.Join(localDir, remote.Name),
				remotePath: remote.Path,
				size:       remote.Size,
			})
			continue
		}

		needSync, err := m.needSyncLocal(local, remote, opts.Mode)
		if err != nil {
			logger.GetLogger().Warn("failed to check sync status",
				zap.String("file", remote.Name),
				zap.String("error", err.Error()),
			)
			continue
		}

		if needSync {
			toDownload = append(toDownload, syncFileTask{
				localPath:  filepath.Join(localDir, remote.Name),
				remotePath: remote.Path,
				size:       remote.Size,
			})
		}
	}

	if opts.Delete {
		toDelete := m.findLocalOrphans(remoteFiles, localFiles)
		for _, f := range toDelete {
			os.Remove(filepath.Join(localDir, f.name))
		}
	}

	return m.executeParallelDownload(sessionID, toDownload, opts.Parallel, progress)
}

type syncFileTask struct {
	localPath  string
	remotePath string
	size       int64
}

type localFileInfo struct {
	name     string
	path     string
	size     int64
	modTime  int64
	checksum string
}

func (m *SyncManager) scanLocalDir(dir string) ([]localFileInfo, error) {
	var files []localFileInfo

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(dir, path)
		files = append(files, localFileInfo{
			name:    relPath,
			path:    path,
			size:    info.Size(),
			modTime: info.ModTime().Unix(),
		})

		return nil
	})

	return files, err
}

func (m *SyncManager) shouldIgnore(name string, ignore []string) bool {
	for _, pattern := range ignore {
		matched, _ := filepath.Match(pattern, name)
		if matched {
			return true
		}
	}
	return false
}

func (m *SyncManager) needSync(local localFileInfo, remote types.FileInfo, mode SyncMode, sessionID string) (bool, error) {
	switch mode {
	case SyncModeSize:
		return local.size != remote.Size, nil
	case SyncModeModTime:
		localModTime := local.modTime
		remoteModTime := remote.ModTime.Unix()
		return localModTime > remoteModTime, nil
	case SyncModeChecksum:
		localChecksum, err := m.resumeMgr.CalculateChecksum(local.path)
		if err != nil {
			return false, err
		}
		remoteChecksum, err := m.sftpManager.ReadFile(sessionID, remote.Path+".sha256")
		if err != nil {
			return true, nil
		}
		return localChecksum != remoteChecksum, nil
	default:
		return false, fmt.Errorf("unknown sync mode: %d", mode)
	}
}

func (m *SyncManager) needSyncLocal(local localFileInfo, remote types.FileInfo, mode SyncMode) (bool, error) {
	switch mode {
	case SyncModeSize:
		return local.size != remote.Size, nil
	case SyncModeModTime:
		localModTime := local.modTime
		remoteModTime := remote.ModTime.Unix()
		return localModTime < remoteModTime, nil
	default:
		return false, nil
	}
}

func (m *SyncManager) findOrphans(localFiles []localFileInfo, remoteFiles []types.FileInfo) []types.FileInfo {
	localNames := make(map[string]bool)
	for _, f := range localFiles {
		localNames[f.name] = true
	}

	var orphans []types.FileInfo
	for _, f := range remoteFiles {
		if !localNames[f.Name] {
			orphans = append(orphans, f)
		}
	}
	return orphans
}

func (m *SyncManager) findLocalOrphans(remoteFiles []types.FileInfo, localFiles []localFileInfo) []localFileInfo {
	remoteNames := make(map[string]bool)
	for _, f := range remoteFiles {
		remoteNames[f.Name] = true
	}

	var orphans []localFileInfo
	for _, f := range localFiles {
		if !remoteNames[f.name] {
			orphans = append(orphans, f)
		}
	}
	return orphans
}

func (m *SyncManager) executeParallelUpload(sessionID string, tasks []syncFileTask, parallel int, progress chan float64) error {
	if len(tasks) == 0 {
		return nil
	}

	if parallel <= 0 {
		parallel = 4
	}

	var wg sync.WaitGroup
	var completed int64
	total := len(tasks)

	sem := make(chan struct{}, parallel)

	for _, task := range tasks {
		wg.Add(1)
		go func(t syncFileTask) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			err := m.sftpManager.Upload(sessionID, t.localPath, t.remotePath, nil)
			if err != nil {
				logger.GetLogger().Error("failed to upload",
					zap.String("path", t.localPath),
					zap.String("error", err.Error()),
				)
			}

			atomic.AddInt64(&completed, 1)
			if progress != nil {
				progress <- float64(completed) / float64(total) * 100
			}
		}(task)
	}

	wg.Wait()

	logger.GetLogger().Info("sync upload completed",
		zap.Int("total", total),
	)
	return nil
}

func (m *SyncManager) executeParallelDownload(sessionID string, tasks []syncFileTask, parallel int, progress chan float64) error {
	if len(tasks) == 0 {
		return nil
	}

	if parallel <= 0 {
		parallel = 4
	}

	var wg sync.WaitGroup
	var completed int64
	total := len(tasks)

	sem := make(chan struct{}, parallel)

	for _, task := range tasks {
		wg.Add(1)
		go func(t syncFileTask) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			localDir := filepath.Dir(t.localPath)
			if err := os.MkdirAll(localDir, 0755); err != nil {
				logger.GetLogger().Error("failed to create local dir",
					zap.String("dir", localDir),
					zap.String("error", err.Error()),
				)
				return
			}

			err := m.sftpManager.Download(sessionID, t.remotePath, t.localPath, nil)
			if err != nil {
				logger.GetLogger().Error("failed to download",
					zap.String("path", t.remotePath),
					zap.String("error", err.Error()),
				)
			}

			atomic.AddInt64(&completed, 1)
			if progress != nil {
				progress <- float64(completed) / float64(total) * 100
			}
		}(task)
	}

	wg.Wait()

	logger.GetLogger().Info("sync download completed",
		zap.Int("total", total),
	)
	return nil
}

func isEmptyDirError(err error) bool {
	return err != nil && err.Error() == "empty directory"
}
