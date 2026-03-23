package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"virt-tui/internal/models"
	"virt-tui/internal/tui/styles"
	"virt-tui/internal/virt"
)

type NetworkModel struct {
	List   list.Model
	Styles styles.Styles
}

func NewNetworkModel(s styles.Styles) NetworkModel {
	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	return NetworkModel{List: l, Styles: s}
}

func (m NetworkModel) Update(msg tea.Msg) (NetworkModel, tea.Cmd) {
	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)
	return m, cmd
}

func (m NetworkModel) View(w, h int) string {
	sel, ok := m.List.SelectedItem().(models.NetItem)
	if !ok {
		return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, "No Network selected")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		m.Styles.SectionHeader.Render(" [ NETWORK SUMMARY ] "),
		m.renderRow("Name", sel.Net.Name),
		m.renderRow("Active", fmt.Sprintf("%v", sel.Net.Active)),
		m.renderRow("Autostart", fmt.Sprintf("%v", sel.Net.Autostart)),
	)
}

func (m NetworkModel) renderRow(key, value string) string {
	label := fmt.Sprintf("%-12s", key+":")
	return fmt.Sprintf("%s %s", m.Styles.Label.Render(label), m.Styles.Value.Render(value))
}
