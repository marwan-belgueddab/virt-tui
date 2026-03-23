package views

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"virt-tui/internal/tui/styles"
	"virt-tui/internal/virt"
)

type HostModel struct {
	Stats  virt.HostStats
	Styles styles.Styles
}

func NewHostModel(s styles.Styles) HostModel {
	return HostModel{Styles: s}
}

func (m HostModel) View(w, h int) string {
	return lipgloss.JoinVertical(lipgloss.Left,
		m.Styles.SectionHeader.Render(" [ HOST INFORMATION ] "),
		m.renderRow("Hostname", m.Stats.Hostname),
		m.renderRow("Model", m.Stats.Model),
		m.renderRow("CPUs", fmt.Sprintf("%d nodes", m.Stats.CPUNodes)),
		m.renderRow("Topology", m.Stats.CPUTopology),
		"",
		m.Styles.SectionHeader.Render(" [ MEMORY ] "),
		m.renderRow("Total", virt.FormatBytes(m.Stats.MemoryTotal*1024)),
		m.renderRow("Free", virt.FormatBytes(m.Stats.MemoryFree*1024)),
		"",
		m.Styles.SectionHeader.Render(" [ SOFTWARE ] "),
		m.renderRow("Libvirt", m.Stats.LibvirtVersion),
	)
}

func (m HostModel) renderRow(key, value string) string {
	label := fmt.Sprintf("%-12s", key+":")
	return fmt.Sprintf("%s %s", m.Styles.Label.Render(label), m.Styles.Value.Render(value))
}
