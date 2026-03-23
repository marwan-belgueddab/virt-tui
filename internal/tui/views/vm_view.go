package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/libvirt/libvirt-go"
	"virt-tui/internal/models"
	"virt-tui/internal/tui/styles"
	"virt-tui/internal/virt"
)

type VMModel struct {
	List       list.Model
	Stats      virt.VMStats
	Config     virt.VMConfig
	CPUHistory []float64
	MemHistory []float64
	Styles     styles.Styles
	Progress   progress.Model
	Width      int
	Height     int
}

func NewVMModel(s styles.Styles) VMModel {
	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)

	pg := progress.New(progress.WithGradient("#7aa2f7", "#00FFC2"))

	return VMModel{
		List:     l,
		Styles:   s,
		Progress: pg,
	}
}

func (m VMModel) Update(msg tea.Msg) (VMModel, tea.Cmd) {
	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)
	return m, cmd
}

func (m VMModel) View(w, h int, activeTab int, focus bool) string {
	m.Width = w
	m.Height = h

	sel, ok := m.List.SelectedItem().(models.VMItem)
	if !ok {
		return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, "No VM selected")
	}

	switch activeTab {
	case 0: // TabSummary (Assuming index 0 is Summary)
		return m.renderSummary(sel, w, h)
	case 2: // TabHardware
		return m.renderHardware(w, h)
	default:
		return "Content for tab " + fmt.Sprintf("%d", activeTab)
	}
}

func (m VMModel) renderSummary(sel models.VMItem, w, h int) string {
	// Left Column: Basic Info & Metadata
	leftCol := lipgloss.JoinVertical(lipgloss.Left,
		m.Styles.SectionHeader.Render(" [ BASIC INFO ] "),
		m.renderRow("Name", sel.VM.Name),
		m.renderRow("State", virt.StateString(sel.VM.State)),
		m.renderRow("UUID", sel.VM.UUID),
		m.renderRow("OS", m.Config.OS),
		m.renderRow("Arch", m.Config.Arch),
		m.renderRow("Machine", m.Config.Machine),
		"",
		m.Styles.SectionHeader.Render(" [ RESOURCES ] "),
		m.renderRow("vCPUs", fmt.Sprintf("%d", m.Config.VCPUs)),
		m.renderRow("Memory", fmt.Sprintf("%.2f GB", float64(m.Config.Memory)/1024/1024)),
		m.renderRow("Max Mem", fmt.Sprintf("%.2f GB", float64(m.Config.MaxMemory)/1024/1024)),
	)

	// Center Column: Networking & Storage
	var netRows []string
	for _, iface := range m.Config.Interfaces {
		ips := strings.Join(iface.IPs, ", ")
		if ips == "" { ips = "No IP" }
		netRows = append(netRows, m.renderRow(iface.Name, ips))
		netRows = append(netRows, fmt.Sprintf("  %s %s (%s)", m.Styles.Label.Render("MAC:"), iface.MAC, iface.Source))
	}
	if len(netRows) == 0 { netRows = append(netRows, "  No interfaces") }

	var diskRows []string
	for _, disk := range m.Config.Disks {
		diskRows = append(diskRows, m.renderRow(disk.Target, disk.Source))
		diskRows = append(diskRows, fmt.Sprintf("  %s %s | %s", m.Styles.Label.Render("Bus:"), disk.Bus, disk.Device))
	}
	if len(diskRows) == 0 { diskRows = append(diskRows, "  No disks") }

	centerCol := lipgloss.JoinVertical(lipgloss.Left,
		m.Styles.SectionHeader.Render(" [ NETWORKING ] "),
		strings.Join(netRows, "\n"),
		"",
		m.Styles.SectionHeader.Render(" [ STORAGE ] "),
		strings.Join(diskRows, "\n"),
	)

	// Right Column: Stats & Graphics
	cpuPerc := m.Stats.CPUPercent
	memPerc := 0.0
	if m.Stats.MemoryTotal > 0 { memPerc = float64(m.Stats.MemoryUsed) / float64(m.Stats.MemoryTotal) * 100 }
	sparkData := m.CPUHistory
	if len(sparkData) > 15 { sparkData = sparkData[len(sparkData)-15:] }

	m.Progress.Width = 20
	cpuBar := m.Progress.ViewAs(cpuPerc / 100)
	memBar := m.Progress.ViewAs(memPerc / 100)

	var graphicsRows []string
	for _, g := range m.Config.Graphics {
		listen := g.Listen
		if listen == "" { listen = "localhost" }
		graphicsRows = append(graphicsRows, m.renderRow(g.Type, fmt.Sprintf("%s:%d", listen, g.Port)))
	}

	rightCol := lipgloss.JoinVertical(lipgloss.Left,
		m.Styles.SectionHeader.Render(" [ PERFORMANCE ] "),
		fmt.Sprintf("%s: %s  %s", m.Styles.Label.Render("CPU"), m.Styles.ValueStyle.Render(fmt.Sprintf("%5.1f%%", cpuPerc)), m.renderBrailleSparkline(sparkData)),
		cpuBar,
		"",
		fmt.Sprintf("%s: %s", m.Styles.Label.Render("RAM"), m.Styles.ValueStyle.Render(fmt.Sprintf("%5.1f%%", memPerc))),
		memBar,
		fmt.Sprintf("%s / %s", virt.FormatBytes(m.Stats.MemoryUsed*1024), virt.FormatBytes(m.Stats.MemoryTotal*1024)),
		"",
		m.Styles.SectionHeader.Render(" [ DISPLAY ] "),
		strings.Join(graphicsRows, "\n"),
	)

	gutter := lipgloss.NewStyle().Width(3).Render("")

	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(w/3-2).Render(leftCol),
		gutter,
		lipgloss.NewStyle().Width(w/3+2).Render(centerCol),
		gutter,
		lipgloss.NewStyle().Width(w/3-2).Render(rightCol),
	)
}

func (m VMModel) renderHardware(w, h int) string {
	return lipgloss.JoinVertical(lipgloss.Left,
		m.Styles.SectionHeader.Render(" [ SYSTEM ] "),
		m.renderRow("Emulator", m.Config.Emulator),
		m.renderRow("Machine", m.Config.Machine),
		m.renderRow("OS Type", m.Config.OS),
		"",
		m.Styles.SectionHeader.Render(" [ CPU & MEMORY ] "),
		m.renderRow("vCPUs", fmt.Sprintf("%d", m.Config.VCPUs)),
		m.renderRow("Current Mem", virt.FormatBytes(m.Config.Memory*1024)),
		m.renderRow("Maximum Mem", virt.FormatBytes(m.Config.MaxMemory*1024)),
	)
}

func (m VMModel) renderRow(key, value string) string {
	label := fmt.Sprintf("%-12s", key+":")
	return fmt.Sprintf("%s %s", m.Styles.Label.Render(label), m.Styles.Value.Render(value))
}

func (m VMModel) renderBrailleSparkline(data []float64) string {
	if len(data) == 0 { return "" }
	chars := []rune{'⡀', '⣀', '⣄', '⣤', '⣦', '⣶', '⣷', '⣿'}
	var res strings.Builder
	for _, v := range data {
		idx := int(v / 12.5)
		if idx > 7 { idx = 7 }
		if idx < 0 { idx = 0 }
		res.WriteRune(chars[idx])
	}
	return res.String()
}
