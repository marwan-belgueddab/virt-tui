package components

import (
	"time"

	"github.com/charmbracelet/lipgloss"
	"virt-tui/internal/tui/styles"
)

type ToastType int

const (
	ToastInfo ToastType = iota
	ToastSuccess
	ToastError
)

type Toast struct {
	Message   string
	Type      ToastType
	ExpiresAt time.Time
}

type ToastModel struct {
	Current *Toast
	styles  styles.Styles
}

func NewToastModel() ToastModel {
	return ToastModel{
		styles: styles.DefaultStyles(),
	}
}

func (m *ToastModel) Set(msg string, t ToastType, duration time.Duration) {
	m.Current = &Toast{
		Message:   msg,
		Type:      t,
		ExpiresAt: time.Now().Add(duration),
	}
}

func (m ToastModel) View() string {
	if m.Current == nil || time.Now().After(m.Current.ExpiresAt) {
		return ""
	}

	style := m.styles.BadgeNormal
	prefix := " INFO "
	
	switch m.Current.Type {
	case ToastSuccess:
		style = lipgloss.NewStyle().Background(styles.ColorMint).Foreground(styles.ColorBG).Bold(true)
		prefix = " SUCCESS "
	case ToastError:
		style = m.styles.BadgeCritical.Background(styles.ColorRed).Foreground(styles.ColorWhite).Bold(true)
		prefix = " ERROR "
	}

	content := lipgloss.NewStyle().
		Background(styles.ColorSubtle).
		Foreground(styles.ColorWhite).
		Padding(0, 1).
		Render(m.Current.Message)

	return lipgloss.JoinHorizontal(lipgloss.Top, style.Render(prefix), content)
}

func (m *ToastModel) Update() {
	if m.Current != nil && time.Now().After(m.Current.ExpiresAt) {
		m.Current = nil
	}
}
