package views

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/termbus/termbus/internal/ssh"
	"github.com/termbus/termbus/pkg/interfaces"
	"github.com/termbus/termbus/pkg/types"
	"github.com/termbus/termbus/tui/components/commandbar"
	"github.com/termbus/termbus/tui/components/hostlist"
	"github.com/termbus/termbus/tui/components/modal"
	"github.com/termbus/termbus/tui/components/shell"
	"github.com/termbus/termbus/tui/components/statusbar"
	"github.com/termbus/termbus/tui/components/tabs"
)

type AppModel struct {
	width  int
	height int

	eventBus interfaces.EventBus
	sessions interfaces.SessionManager
	sshMgr   *ssh.SSHManager

	hostList   hostlist.Model
	sessionTab tabs.Model
	shellView  shell.Model
	commandBar commandbar.Model
	statusBar  statusbar.Model
	modalState modalState

	activeSessionID string
	activeHostAlias string
}

func NewApp(eventBus interfaces.EventBus, sessions interfaces.SessionManager, sshMgr *ssh.SSHManager, width, height int) AppModel {
	hosts := loadHosts(sshMgr)

	hostList := hostlist.New(24, height-4, hosts)
	commandBar := commandbar.New()
	commandBar.SetSize(width)
	statusBar := statusbar.New()
	statusBar.SetSize(width)
	shellView := shell.New("Shell", eventBus)
	shellView.SetSize(width-26, height-4)

	model := AppModel{
		width:      width,
		height:     height,
		eventBus:   eventBus,
		sessions:   sessions,
		sshMgr:     sshMgr,
		hostList:   hostList,
		shellView:  shellView,
		commandBar: commandBar,
		statusBar:  statusBar,
		sessionTab: tabs.New(nil),
		modalState: modalState{active: false},
	}

	model.subscribeEvents()
	model.refreshTabs()

	return model
}

func (m AppModel) Init() tea.Cmd {
	// Subscribe to event bus events
	if m.eventBus != nil {
		m.eventBus.Subscribe("command.output", func(args ...interface{}) {
			if len(args) > 1 {
				if text, ok := args[1].(string); ok {
					m.shellView.AppendOutput(text + "\n")
				}
			}
		})
		m.eventBus.Subscribe("plugin.permission.requested", func(args ...interface{}) {
			pluginID := ""
			perm := ""
			if len(args) > 0 {
				if v, ok := args[0].(string); ok {
					pluginID = v
				}
			}
			if len(args) > 1 {
				if v, ok := args[1].(string); ok {
					perm = v
				}
			}
			m.modalState.confirm = modal.ConfirmModal{
				Title:   "Permission Request",
				Message: fmt.Sprintf("Plugin %s is requesting permission: %s", pluginID, perm),
				OnYes: func() {
					m.modalState.active = false
					if m.eventBus != nil {
						m.eventBus.Publish("plugin.permission.granted", pluginID)
					}
				},
				OnNo: func() {
					m.modalState.active = false
					if m.eventBus != nil {
						m.eventBus.Publish("plugin.permission.revoked", pluginID)
					}
				},
				Active: true,
			}
			m.modalState.active = true
		})
		m.eventBus.Subscribe("plugin.permission.granted", func(args ...interface{}) {
			if len(args) == 0 {
				return
			}
			if id, ok := args[0].(string); ok {
				m.statusBar.Left = "Permission granted: " + id
			}
		})
		m.eventBus.Subscribe("plugin.permission.revoked", func(args ...interface{}) {
			if len(args) == 0 {
				return
			}
			if id, ok := args[0].(string); ok {
				m.statusBar.Left = "Permission denied: " + id
			}
		})
	}
	return tea.WindowSize()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case string:
		// command output messages rendered in shell view for now
		m.shellView.AppendOutput(msg + "\n")
	case []interface{}:
		if len(msg) > 1 {
			if text, ok := msg[1].(string); ok {
				m.shellView.AppendOutput(text + "\n")
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
	case tea.KeyMsg:
		if m.modalState.active {
			var cmd tea.Cmd
			m.modalState.confirm, cmd = m.modalState.confirm.Update(msg)
			return m, cmd
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if selected := m.hostList.Selected(); selected != nil {
				session, err := m.sessions.CreateSession(selected)
				if err == nil {
					_ = m.sessions.ConnectSession(session.ID)
					_ = m.sessions.SetActiveSession(session.ID)
					m.activeSessionID = session.ID
					m.activeHostAlias = selected.Alias

					ptyModel := &shell.PTYModel{}
					if err := ptyModel.Connect(session.ID, m.sessions); err == nil {
						m.shellView.Bind(ptyModel, session.ID)
						if m.eventBus != nil {
							m.eventBus.Publish("shell.connected", session.ID)
						}
						go func() {
							_ = ptyModel.Stream(func(chunk string) {
								m.shellView.AppendOutput(chunk)
							})
						}()
					}

					m.refreshTabs()
				}
			}
		}
	}

	var cmd tea.Cmd
	var cmdHost tea.Cmd
	var cmdBar tea.Cmd
	m.hostList, cmdHost = m.hostList.Update(msg)
	m.commandBar, cmdBar = m.commandBar.Update(msg)
	cmd = tea.Batch(cmdHost, cmdBar)
	return m, cmd
}

func (m AppModel) View() string {
	status := m.statusBar.View()
	tabsView := m.sessionTab.View(m.width)

	left := m.hostList.View()
	right := m.shellView.View()

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	view := lipgloss.JoinVertical(lipgloss.Left, status, tabsView, body, m.commandBar.View())
	if m.modalState.active {
		modalView := m.modalState.confirm.View(m.width / 2)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modalView)
	}
	return view
}

func (m *AppModel) resize() {
	leftWidth := 24
	mainHeight := m.height - 4

	m.hostList.SetSize(leftWidth, mainHeight)
	m.shellView.SetSize(m.width-leftWidth-2, mainHeight)
	m.commandBar.SetSize(m.width)
	m.statusBar.SetSize(m.width)
}

func (m *AppModel) refreshTabs() {
	sessions := m.sessions.ListSessions()
	tabsList := make([]tabs.Tab, 0, len(sessions))
	for _, session := range sessions {
		label := session.HostConfig.Host
		if session.HostConfig.Alias != "" {
			label = session.HostConfig.Alias
		}
		tabsList = append(tabsList, tabs.Tab{
			ID:      session.ID,
			Title:   fmt.Sprintf("[%s]", label),
			Active:  session.ID == m.activeSessionID,
			HasWarn: session.State == types.SessionStateError,
		})
	}
	m.sessionTab.SetTabs(tabsList)
}

func (m *AppModel) subscribeEvents() {
	if m.eventBus == nil {
		return
	}
	m.eventBus.Subscribe("session.created", func(session *types.Session) {
		m.activeSessionID = session.ID
		m.refreshTabs()
	})
	m.eventBus.Subscribe("session.state.changed", func(session *types.Session) {
		m.refreshTabs()
	})
}

func loadHosts(sshMgr *ssh.SSHManager) []types.SSHHostConfig {
	if sshMgr == nil {
		return nil
	}
	configs, err := sshMgr.ScanSSHConfigs()
	if err != nil {
		return nil
	}

	hosts := make([]types.SSHHostConfig, 0, len(configs))
	for _, cfg := range configs {
		hosts = append(hosts, types.SSHHostConfig{
			Host:     cfg.Host,
			HostName: cfg.HostName,
			User:     cfg.User,
			Port:     cfg.Port,
			Alias:    cfg.Host,
		})
	}
	return hosts
}
