package components

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"virt-tui/internal/tui/styles"
)

type MenuOption struct {
	Label string
	Action func() tea.Cmd
}

func (o MenuOption) Title() string       { return o.Label }
func (o MenuOption) Description() string { return "" }
func (o MenuOption) FilterValue() string { return o.Label }

type MenuModel struct {
	List   list.Model
	Show   bool
	Title  string
	styles styles.Styles
}

type menuDelegate struct {
	styles styles.Styles
}

func (d menuDelegate) Height() int                               { return 1 }
func (d menuDelegate) Spacing() int                              { return 0 }
func (d menuDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d menuDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(MenuOption)
	if !ok {
		return
	}

	str := fmt.Sprintf("  %s", i.Label)

	fn := lipgloss.NewStyle().PaddingLeft(2).Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return d.styles.TreeSelected.Render("> " + strings.Join(s, ""))
		}
	}

	fmt.Fprint(w, fn(str))
}

func NewMenuModel() MenuModel {
	s := styles.DefaultStyles()
	l := list.New(nil, menuDelegate{styles: s}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	return MenuModel{
		List:   l,
		styles: styles.DefaultStyles(),
	}
}

func (m *MenuModel) SetOptions(title string, options []MenuOption) {
	m.Title = title
	items := make([]list.Item, len(options))
	for i, o := range options {
		items[i] = o
	}
	m.List.SetItems(items)
	m.List.Select(0)
	m.Show = true
}

func (m MenuModel) Update(msg tea.Msg) (MenuModel, tea.Cmd) {
	if !m.Show {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			m.Show = false
			return m, nil
		case "enter":
			if sel, ok := m.List.SelectedItem().(MenuOption); ok {
				m.Show = false
				return m, sel.Action()
			}
		}
	}

	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)
	return m, cmd
}

func (m MenuModel) View(w, h int) string {
	if !m.Show {
		return ""
	}

	m.List.SetSize(30, len(m.List.Items()))
	
	header := m.styles.SectionHeader.Render(fmt.Sprintf(" %s ", strings.ToUpper(m.Title)))
	body := m.List.View()
	
	return m.styles.Modal.Render(lipgloss.JoinVertical(lipgloss.Left, header, body))
}
