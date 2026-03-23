package components

import (
	"fmt"
	"strings"

	"virt-tui/internal/tui/styles"
)

type HelpEntry struct {
	Key  string
	Desc string
}

type HelpCategory struct {
	Title   string
	Entries []HelpEntry
}

var GlobalHelp = []HelpCategory{
	{
		Title: "Navigation & Views",
		Entries: []HelpEntry{
			{"Tab", "Cycle focus areas"},
			{"v", "Switch to VM view"},
			{"n", "Switch to Network view"},
			{"p", "Switch to Storage Pool view"},
			{"1-4", "Direct tab access"},
			{"Arrows", "Navigate tree or lists"},
			{"?", "Toggle help modal"},
			{"q", "Quit application"},
		},
	},
	{
		Title: "Actions",
		Entries: []HelpEntry{
			{"m / Space", "Open context menu for selected item"},
			{"C", "Create new VM (Wizard)"},
			{"R", "Refresh all resources"},
		},
	},
}

type HelpModel struct {
	Show   bool
	styles styles.Styles
}

func NewHelpModel() HelpModel {
	return HelpModel{
		styles: styles.DefaultStyles(),
	}
}

func (m HelpModel) View(w, h int) string {
	if !m.Show {
		return ""
	}

	var content []string
	content = append(content, m.styles.SectionHeader.Render(" GLOBAL HELP "))
	content = append(content, "")

	for _, cat := range GlobalHelp {
		content = append(content, m.styles.Label.Render(fmt.Sprintf(" [%s] ", strings.ToUpper(cat.Title))))
		for _, e := range cat.Entries {
			line := fmt.Sprintf("  %-12s %s", m.styles.KeyCap.Render(e.Key), e.Desc)
			content = append(content, line)
		}
		content = append(content, "")
	}

	content = append(content, " Press '?' or 'Esc' to close ")

	return m.styles.Modal.Render(strings.Join(content, "\n"))
}
