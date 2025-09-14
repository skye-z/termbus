package sftp

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/termbus/termbus/internal/logger"
	"go.uber.org/zap"
)

type ChunkConfig struct {
	ChunkSize   int64
	MaxParallel int
}

func DefaultChunkConfig() *ChunkConfig {
	return &ChunkConfig{
		ChunkSize:   10 * 1024 * 1024,
		MaxParallel: 4,
	}
}

func (m *SFTPManager) UploadWithChunk(sessionID, localPath, remotePath string, config *ChunkConfig, progress chan float64) error {
	if config == nil {
		config = DefaultChunkConfig()
	}

	info, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	totalSize := info.Size()
	if totalSize <= config.ChunkSize {
		return m.Upload(sessionID, localPath, remotePath, progress)
	}

	chunkCount := (totalSize + config.ChunkSize - 1) / config.ChunkSize

	logger.GetLogger().Info("starting chunked upload",
		zap.String("path", localPath),
		zap.Int64("total_size", totalSize),
		zap.Int64("chunk_size", config.ChunkSize),
		zap.Int("chunk_count", int(chunkCount)),
	)

	sem := make(chan struct{}, config.MaxParallel)
	var wg sync.WaitGroup
	var completed int64

	for i := int64(0); i < chunkCount; i++ {
		wg.Add(1)
		go func(chunkIndex int64) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			offset := chunkIndex * config.ChunkSize
			chunkSize := config.ChunkSize
			if offset+chunkSize > totalSize {
				chunkSize = totalSize - offset
			}

			chunkPath := fmt.Sprintf("%s.chunk.%d", remotePath, chunkIndex)

			chunkFile, err := os.Open(localPath)
			if err != nil {
				logger.GetLogger().Error("failed to open chunk file",
					zap.Int("index", int(chunkIndex)),
					zap.String("error", err.Error()),
				)
				return
			}
			defer chunkFile.Close()

			_, err = chunkFile.Seek(offset, 0)
			if err != nil {
				logger.GetLogger().Error("failed to seek chunk",
					zap.Int("index", int(chunkIndex)),
					zap.String("error", err.Error()),
				)
				return
			}

			chunkData := make([]byte, chunkSize)
			n, readErr := chunkFile.Read(chunkData)
			if readErr != nil || int64(n) != chunkSize {
				logger.GetLogger().Error("failed to read chunk",
					zap.Int("index", int(chunkIndex)),
				)
				return
			}

			client, err := m.getOrCreateClient(sessionID)
			if err != nil {
				logger.GetLogger().Error("failed to get sftp client",
					zap.String("error", err.Error()),
				)
				return
			}

			f, err := client.Create(chunkPath)
			if err != nil {
				logger.GetLogger().Error("failed to create chunk",
					zap.Int("index", int(chunkIndex)),
					zap.String("error", err.Error()),
				)
				return
			}
			defer f.Close()

			_, err = f.Write(chunkData)
			if err != nil {
				logger.GetLogger().Error("failed to write chunk",
					zap.Int("index", int(chunkIndex)),
					zap.String("error", err.Error()),
				)
				return
			}

			atomic.AddInt64(&completed, 1)
			if progress != nil {
				progress <- float64(atomic.LoadInt64(&completed)) / float64(chunkCount) * 100
			}

			logger.GetLogger().Debug("chunk uploaded",
				zap.Int("index", int(chunkIndex)),
			)
		}(i)
	}

	wg.Wait()

	err = m.mergeChunks(sessionID, remotePath, chunkCount)
	if err != nil {
		return fmt.Errorf("failed to merge chunks: %w", err)
	}

	logger.GetLogger().Info("chunked upload completed",
		zap.String("path", remotePath),
	)

	return nil
}

func (m *SFTPManager) DownloadWithChunk(sessionID, remotePath, localPath string, config *ChunkConfig, progress chan float64) error {
	if config == nil {
		config = DefaultChunkConfig()
	}

	size, err := m.GetFileSize(sessionID, remotePath)
	if err != nil {
		return fmt.Errorf("failed to get remote file size: %w", err)
	}

	if size <= config.ChunkSize {
		return m.Download(sessionID, remotePath, localPath, progress)
	}

	chunkCount := (size + config.ChunkSize - 1) / config.ChunkSize

	logger.GetLogger().Info("starting chunked download",
		zap.String("path", remotePath),
		zap.Int64("total_size", size),
		zap.Int64("chunk_size", config.ChunkSize),
		zap.Int("chunk_count", int(chunkCount)),
	)

	localDir := fmt.Sprintf("%s.chunks", localPath)
	os.MkdirAll(localDir, 0755)

	sem := make(chan struct{}, config.MaxParallel)
	var wg sync.WaitGroup
	var completed int64

	for i := int64(0); i < chunkCount; i++ {
		wg.Add(1)
		go func(chunkIndex int64) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			offset := chunkIndex * config.ChunkSize
			chunkSize := config.ChunkSize
			if offset+chunkSize > size {
				chunkSize = size - offset
			}

			chunkPath := fmt.Sprintf("%s/chunk.%d", localDir, chunkIndex)

			client, err := m.getOrCreateClient(sessionID)
			if err != nil {
				logger.GetLogger().Error("failed to get sftp client",
					zap.String("error", err.Error()),
				)
				return
			}

			remoteFile, err := client.Open(remotePath)
			if err != nil {
				logger.GetLogger().Error("failed to open remote file",
					zap.String("error", err.Error()),
				)
				return
			}
			defer remoteFile.Close()

			_, err = remoteFile.Seek(offset, 0)
			if err != nil {
				logger.GetLogger().Error("failed to seek",
					zap.Int("index", int(chunkIndex)),
					zap.String("error", err.Error()),
				)
				return
			}

			localFile, err := os.Create(chunkPath)
			if err != nil {
				logger.GetLogger().Error("failed to create local chunk",
					zap.Int("index", int(chunkIndex)),
					zap.String("error", err.Error()),
				)
				return
			}
			defer localFile.Close()

			buffer := make([]byte, 32*1024)
			var written int64
			for written < chunkSize {
				n, readErr := remoteFile.Read(buffer)
				if n > 0 {
					localFile.Write(buffer[:n])
					written += int64(n)
				}
				if readErr != nil {
					break
				}
			}

			atomic.AddInt64(&completed, 1)
			if progress != nil {
				progress <- float64(atomic.LoadInt64(&completed)) / float64(chunkCount) * 100
			}

			logger.GetLogger().Debug("chunk downloaded",
				zap.Int("index", int(chunkIndex)),
			)
		}(i)
	}

	wg.Wait()

	err = m.mergeDownloadChunks(localDir, localPath, chunkCount)
	if err != nil {
		return fmt.Errorf("failed to merge chunks: %w", err)
	}

	os.RemoveAll(localDir)

	logger.GetLogger().Info("chunked download completed",
		zap.String("path", localPath),
	)

	return nil
}

func (m *SFTPManager) mergeChunks(sessionID, remotePath string, chunkCount int64) error {
	client, err := m.getOrCreateClient(sessionID)
	if err != nil {
		return err
	}

	mergedFile, err := client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("failed to create merged file: %w", err)
	}
	defer mergedFile.Close()

	for i := int64(0); i < chunkCount; i++ {
		chunkPath := fmt.Sprintf("%s.chunk.%d", remotePath, i)

		chunkFile, err := client.Open(chunkPath)
		if err != nil {
			return fmt.Errorf("failed to open chunk %d: %w", i, err)
		}

		buffer := make([]byte, 32*1024)
		for {
			n, readErr := chunkFile.Read(buffer)
			if n > 0 {
				mergedFile.Write(buffer[:n])
			}
			if readErr != nil {
				break
			}
		}
		chunkFile.Close()
		client.Remove(chunkPath)
	}

	return nil
}

func (m *SFTPManager) mergeDownloadChunks(chunkDir, outputPath string, chunkCount int64) error {
	mergedFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create merged file: %w", err)
	}
	defer mergedFile.Close()

	for i := int64(0); i < chunkCount; i++ {
		chunkPath := fmt.Sprintf("%s/chunk.%d", chunkDir, i)

		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return fmt.Errorf("failed to open chunk %d: %w", i, err)
		}

		buffer := make([]byte, 32*1024)
		for {
			n, readErr := chunkFile.Read(buffer)
			if n > 0 {
				mergedFile.Write(buffer[:n])
			}
			if readErr != nil {
				break
			}
		}
		chunkFile.Close()
		os.Remove(chunkPath)
	}

	return nil
}
