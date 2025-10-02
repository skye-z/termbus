package hostlist

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/termbus/termbus/pkg/types"
)

type item struct {
	host types.SSHHostConfig
}

func (i item) Title() string { return i.host.Alias }

func (i item) Description() string {
	if i.host.HostName == "" {
		return i.host.Host
	}
	if i.host.User == "" {
		return i.host.HostName
	}
	return fmt.Sprintf("%s@%s", i.host.User, i.host.HostName)
}

func (i item) FilterValue() string {
	if i.host.Alias != "" {
		return i.host.Alias
	}
	return i.host.Host
}

type Model struct {
	list  list.Model
	hosts []types.SSHHostConfig
}

func New(width, height int, hosts []types.SSHHostConfig) Model {
	items := make([]list.Item, 0, len(hosts))
	for _, host := range hosts {
		items = append(items, item{host: host})
	}

	l := list.New(items, list.NewDefaultDelegate(), width, height)
	l.Title = "Hosts"
	return Model{list: l, hosts: hosts}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return m.list.View()
}

func (m Model) Selected() *types.SSHHostConfig {
	selected := m.list.SelectedItem()
	if selected == nil {
		return nil
	}
	it, ok := selected.(item)
	if !ok {
		return nil
	}
	return &it.host
}

func (m Model) SetSize(width, height int) {
	m.list.SetSize(width, height)
}
