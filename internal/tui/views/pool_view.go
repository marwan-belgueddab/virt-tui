package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"virt-tui/internal/models"
	"virt-tui/internal/tui/styles"
	"virt-tui/internal/virt"
)

type StorageModel struct {
	List     list.Model
	Styles   styles.Styles
	Progress progress.Model
}

func NewStorageModel(s styles.Styles) StorageModel {
	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	pg := progress.New(progress.WithGradient("#7aa2f7", "#00FFC2"))
	return StorageModel{List: l, Styles: s, Progress: pg}
}

func (m StorageModel) Update(msg tea.Msg) (StorageModel, tea.Cmd) {
	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)
	return m, cmd
}

func (m StorageModel) View(w, h int) string {
	sel, ok := m.List.SelectedItem().(models.PoolItem)
	if !ok {
		return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, "No Storage Pool selected")
	}

	perc := 0.0
	if sel.Pool.Capacity > 0 { perc = float64(sel.Pool.Allocation) / float64(sel.Pool.Capacity) * 100 }
	
	m.Progress.Width = 30
	bar := m.Progress.ViewAs(perc / 100)

	return lipgloss.JoinVertical(lipgloss.Left,
		m.Styles.SectionHeader.Render(" [ STORAGE POOL SUMMARY ] "),
		m.renderRow("Name", sel.Pool.Name),
		m.renderRow("Active", fmt.Sprintf("%v", sel.Pool.Active)),
		m.renderRow("Path", sel.Pool.Path),
		m.renderRow("Capacity", virt.FormatBytes(sel.Pool.Capacity)),
		m.renderRow("Allocation", virt.FormatBytes(sel.Pool.Allocation)),
		m.renderRow("Available", virt.FormatBytes(sel.Pool.Available)),
		"",
		m.Styles.Label.Render("Usage:"),
		bar,
		fmt.Sprintf("%.1f%% used", perc),
	)
}

func (m StorageModel) renderRow(key, value string) string {
	label := fmt.Sprintf("%-12s", key+":")
	return fmt.Sprintf("%s %s", m.Styles.Label.Render(label), m.Styles.Value.Render(value))
}
