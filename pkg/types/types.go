package types

import (
	"context"
	"io"
	"time"
)

// SessionState 会话状态枚举
type SessionState string

const (
	SessionStateDisconnected SessionState = "disconnected"
	SessionStateConnecting   SessionState = "connecting"
	SessionStateConnected    SessionState = "connected"
	SessionStateReconnecting SessionState = "reconnecting"
	SessionStateError        SessionState = "error"
)

// PaneType 窗格类型
type PaneType string

const (
	PaneTypeShell  PaneType = "shell"
	PaneTypeSFTP   PaneType = "sftp"
	PaneTypeLog    PaneType = "log"
	PaneTypePlugin PaneType = "plugin"
)

// ForwardType 隧道类型
type ForwardType string

const (
	ForwardTypeLocal   ForwardType = "local"
	ForwardTypeRemote  ForwardType = "remote"
	ForwardTypeDynamic ForwardType = "dynamic"
	ForwardTypeX11     ForwardType = "x11"
)

// TunnelStatus 隧道状态
type TunnelStatus string

const (
	TunnelStatusStopped TunnelStatus = "stopped"
	TunnelStatusRunning TunnelStatus = "running"
	TunnelStatusError   TunnelStatus = "error"
)

// Pane 窗格对象
type Pane struct {
	ID        string    `json:"id"`
	Type      PaneType  `json:"type"`
	Title     string    `json:"title"`
	SessionID string    `json:"session_id"`
	Content   io.Reader `json:"-"`
	Active    bool      `json:"active"`
}

// Window 窗口对象
type Window struct {
	ID           string           `json:"id"`
	SessionID    string           `json:"session_id"`
	HostID       string           `json:"host_id"`
	HostAlias    string           `json:"host_alias"`
	Panes        map[string]*Pane `json:"panes"`
	ActivePaneID string           `json:"active_pane_id"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

// SSHHostConfig 主机配置
type SSHHostConfig struct {
	Host                  string   `mapstructure:"Host"`
	HostName              string   `mapstructure:"HostName"`
	User                  string   `mapstructure:"User"`
	Port                  int      `mapstructure:"Port" default:"22"`
	IdentityFile          []string `mapstructure:"IdentityFile"`
	ProxyJump             string   `mapstructure:"ProxyJump"`
	ProxyCommand          string   `mapstructure:"ProxyCommand"`
	ForwardAgent          bool     `mapstructure:"ForwardAgent"`
	ForwardX11            bool     `mapstructure:"ForwardX11"`
	ConnectTimeout        int      `mapstructure:"ConnectTimeout" default:"30"`
	ServerAliveInterval   int      `mapstructure:"ServerAliveInterval" default:"60"`
	ServerAliveCountMax   int      `mapstructure:"ServerAliveCountMax" default:"3"`
	StrictHostKeyChecking string   `mapstructure:"StrictHostKeyChecking" default:"ask"`
	UserKnownHostsFile    string   `mapstructure:"UserKnownHostsFile"`
	Alias                 string   `mapstructure:"Alias"`
	Group                 string   `mapstructure:"Group"`
	Description           string   `mapstructure:"Description"`
}

// Session SSH会话对象
type Session struct {
	ID              string             `json:"id"`
	HostConfig      *SSHHostConfig     `json:"host_config"`
	State           SessionState       `json:"state"`
	Windows         map[string]*Window `json:"windows"`
	ActiveWindowID  string             `json:"active_window_id"`
	CreatedAt       time.Time          `json:"created_at"`
	ConnectedAt     *time.Time         `json:"connected_at"`
	ErrorMsg        string             `json:"error_msg"`
	KeepaliveConfig *KeepaliveConfig   `json:"keepalive_config"`
	ReconnectConfig *ReconnectConfig   `json:"reconnect_config"`
}

// KeepaliveConfig 保活配置
type KeepaliveConfig struct {
	Enabled  bool `json:"enabled"`
	Interval int  `json:"interval"`
	CountMax int  `json:"count_max"`
}

// ReconnectConfig 重连配置
type ReconnectConfig struct {
	Enabled     bool `json:"enabled"`
	MaxAttempts int  `json:"max_attempts"`
	Interval    int  `json:"interval"`
}

// ForwardTunnel 隧道对象
type ForwardTunnel struct {
	ID         string       `json:"id"`
	SessionID  string       `json:"session_id"`
	Type       ForwardType  `json:"type"`
	LocalAddr  string       `json:"local_addr"`
	RemoteAddr string       `json:"remote_addr"`
	Status     TunnelStatus `json:"status"`
	AutoStart  bool         `json:"auto_start"`
	CreatedAt  time.Time    `json:"created_at"`
}

// FileInfo 文件信息
type FileInfo struct {
	Name    string      `json:"name"`
	Size    int64       `json:"size"`
	Mode    interface{} `json:"mode"`
	ModTime time.Time   `json:"mod_time"`
	IsDir   bool        `json:"is_dir"`
	Path    string      `json:"path"`
	Symlink string      `json:"symlink"`
}

// AgentPlan AI执行计划
type AgentPlan struct {
	Task        string      `json:"task"`
	Description string      `json:"description"`
	Steps       []AgentStep `json:"steps"`
	RiskLevel   RiskLevel   `json:"risk_level"`
	RiskTips    []string    `json:"risk_tips"`
}

// AgentStep 执行步骤
type AgentStep struct {
	ID          int        `json:"id"`
	Description string     `json:"description"`
	Status      StepStatus `json:"status"`
	Result      string     `json:"result"`
	Error       string     `json:"error"`
}

// RiskLevel 风险等级
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// StepStatus 步骤状态
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
)

// Tool 工具接口
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, input map[string]interface{}) (string, error)
}

// EventBus 事件总线接口
type EventBus interface {
	Subscribe(topic string, handler interface{})
	Publish(topic string, args ...interface{})
}
