package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"virt-tui/internal/tui/styles"
)

type WizardStep struct {
	Title string
	Input textinput.Model
}

type WizardModel struct {
	Title       string
	Steps       []WizardStep
	ActiveStep  int
	styles      styles.Styles
	Show        bool
	Done        bool
	OnSubmit    func(values map[string]string) tea.Cmd
}

func NewWizardModel(title string, steps []WizardStep, onSubmit func(map[string]string) tea.Cmd) WizardModel {
	return WizardModel{
		Title:    title,
		Steps:    steps,
		styles:   styles.DefaultStyles(),
		OnSubmit: onSubmit,
	}
}

func (m WizardModel) Update(msg tea.Msg) (WizardModel, tea.Cmd) {
	if !m.Show {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.Show = false
			return m, nil
		case "enter":
			if m.ActiveStep < len(m.Steps)-1 {
				m.ActiveStep++
				m.Steps[m.ActiveStep].Input.Focus()
				return m, nil
			}
			// Submit
			values := make(map[string]string)
			for _, s := range m.Steps {
				values[s.Title] = s.Input.Value()
			}
			m.Show = false
			m.Done = true
			return m, m.OnSubmit(values)
		case "up":
			if m.ActiveStep > 0 {
				m.ActiveStep--
				m.Steps[m.ActiveStep].Input.Focus()
			}
		case "down":
			if m.ActiveStep < len(m.Steps)-1 {
				m.ActiveStep++
				m.Steps[m.ActiveStep].Input.Focus()
			}
		}
	}

	var cmd tea.Cmd
	m.Steps[m.ActiveStep].Input, cmd = m.Steps[m.ActiveStep].Input.Update(msg)
	return m, cmd
}

func (m WizardModel) View(w, h int) string {
	if !m.Show {
		return ""
	}

	var content []string
	content = append(content, m.styles.SectionHeader.Render(" "+strings.ToUpper(m.Title)+" "))
	content = append(content, "")

	for i, s := range m.Steps {
		prefix := "  "
		style := m.styles.Label
		if i == m.ActiveStep {
			prefix = "> "
			style = m.styles.BadgeNormal
		}
		content = append(content, fmt.Sprintf("%s%s", prefix, style.Render(s.Title)))
		content = append(content, fmt.Sprintf("  %s", s.Input.View()))
		content = append(content, "")
	}

	content = append(content, fmt.Sprintf(" Step %d of %d | Enter: Next/Finish | Esc: Cancel", m.ActiveStep+1, len(m.Steps)))

	return m.styles.Modal.Render(strings.Join(content, "\n"))
}
