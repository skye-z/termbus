package tunnel

import (
	"sync"
	"sync/atomic"
	"time"
)

type TrafficStats struct {
	TunnelID    string
	BytesIn     int64
	BytesOut    int64
	Connections int64
	UpdatedAt   time.Time
	StartTime   time.Time
}

type TrafficMonitor struct {
	tunnelManager *TunnelManager
	stats         map[string]*TrafficStats
	statsMu       sync.RWMutex
}

func NewTrafficMonitor(tm *TunnelManager) *TrafficMonitor {
	return &TrafficMonitor{
		tunnelManager: tm,
		stats:         make(map[string]*TrafficStats),
	}
}

func (m *TrafficMonitor) StartMonitor(tunnelID string) chan TrafficStats {
	statsCh := make(chan TrafficStats, 100)

	m.statsMu.Lock()
	m.stats[tunnelID] = &TrafficStats{
		TunnelID:  tunnelID,
		UpdatedAt: time.Now(),
		StartTime: time.Now(),
	}
	m.statsMu.Unlock()

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				m.statsMu.RLock()
				stats, exists := m.stats[tunnelID]
				m.statsMu.RUnlock()

				if !exists {
					close(statsCh)
					return
				}

				stats.UpdatedAt = time.Now()

				select {
				case statsCh <- *stats:
				default:
				}

			case <-m.tunnelManager.ctx.Done():
				close(statsCh)
				return
			}
		}
	}()

	return statsCh
}

func (m *TrafficMonitor) GetStats(tunnelID string) *TrafficStats {
	m.statsMu.RLock()
	defer m.statsMu.RUnlock()

	if stats, exists := m.stats[tunnelID]; exists {
		return stats
	}

	return nil
}

func (m *TrafficMonitor) StopMonitor(tunnelID string) {
	m.statsMu.Lock()
	defer m.statsMu.Unlock()

	delete(m.stats, tunnelID)
}

func (m *TrafficMonitor) RecordBytesIn(tunnelID string, bytes int64) {
	m.statsMu.RLock()
	defer m.statsMu.RUnlock()

	if stats, exists := m.stats[tunnelID]; exists {
		atomic.AddInt64(&stats.BytesIn, bytes)
	}
}

func (m *TrafficMonitor) RecordBytesOut(tunnelID string, bytes int64) {
	m.statsMu.RLock()
	defer m.statsMu.RUnlock()

	if stats, exists := m.stats[tunnelID]; exists {
		atomic.AddInt64(&stats.BytesOut, bytes)
	}
}

func (m *TrafficMonitor) IncrementConnections(tunnelID string) {
	m.statsMu.RLock()
	defer m.statsMu.RUnlock()

	if stats, exists := m.stats[tunnelID]; exists {
		atomic.AddInt64(&stats.Connections, 1)
	}
}

func (m *TrafficMonitor) DecrementConnections(tunnelID string) {
	m.statsMu.RLock()
	defer m.statsMu.RUnlock()

	if stats, exists := m.stats[tunnelID]; exists {
		atomic.AddInt64(&stats.Connections, -1)
	}
}

func (m *TrafficMonitor) GetAllStats() map[string]*TrafficStats {
	m.statsMu.RLock()
	defer m.statsMu.RUnlock()

	result := make(map[string]*TrafficStats)
	for k, v := range m.stats {
		result[k] = v
	}

	return result
}

func (m *TrafficMonitor) ClearStats(tunnelID string) {
	m.statsMu.Lock()
	defer m.statsMu.Unlock()

	if stats, exists := m.stats[tunnelID]; exists {
		stats.BytesIn = 0
		stats.BytesOut = 0
		stats.Connections = 0
	}
}
