package styles

import "github.com/charmbracelet/lipgloss"

// --- Neubrutalist Palette ---
const (
	SidebarWidth = 35
	ColorBG      = lipgloss.Color("#1a1b26")
	ColorAccent  = lipgloss.Color("#7aa2f7") // Blue for Active/Selection
	ColorGold    = lipgloss.Color("#D79921") // Gold for Labels
	ColorCritical = lipgloss.Color("#ff9e64") 
	ColorMint    = lipgloss.Color("#00FFC2")
	ColorYellow  = lipgloss.Color("#e0af68")
	ColorRed     = lipgloss.Color("#f7768e")
	ColorWhite   = lipgloss.Color("#ffffff")
	ColorDim     = lipgloss.Color("#565f89")
	ColorSubtle  = lipgloss.Color("#24283b")
)

type Styles struct {
	// ... (rest of fields remain same)
	Main        lipgloss.Style
	Header      lipgloss.Style
	Sidebar     lipgloss.Style
	ContentArea lipgloss.Style
	
	TreeFolder       lipgloss.Style
	TreeVM           lipgloss.Style
	TreeSelected     lipgloss.Style
	TreeUnfocused    lipgloss.Style
	TreeHeader       lipgloss.Style
	
	BadgeNormal   lipgloss.Style
	BadgeCritical lipgloss.Style
	
	TabRow      lipgloss.Style
	Tab         lipgloss.Style
	ActiveTab   lipgloss.Style
	FocusedTab  lipgloss.Style
	
	SectionHeader lipgloss.Style
	KeyStyle      lipgloss.Style
	ValueStyle    lipgloss.Style
	Label         lipgloss.Style
	Value         lipgloss.Style
	
	BarEmpty  lipgloss.Style
	
	Footer     lipgloss.Style
	FooterPill lipgloss.Style
	KeyCap     lipgloss.Style
	Modal      lipgloss.Style
}

func DefaultStyles() Styles {
	s := Styles{}

	s.Main = lipgloss.NewStyle().Background(ColorBG).Foreground(ColorWhite)
	
	s.Header = lipgloss.NewStyle().
		Background(ColorAccent).
		Foreground(ColorBG).
		Bold(true).
		Height(1)

	s.Sidebar = lipgloss.NewStyle().
		Width(SidebarWidth).
		MaxWidth(SidebarWidth).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(ColorSubtle).
		Padding(0, 0)

	s.ContentArea = lipgloss.NewStyle().
		Padding(0, 1)

	s.TreeFolder = lipgloss.NewStyle().Foreground(ColorDim)
	s.TreeVM = lipgloss.NewStyle().Foreground(ColorWhite)
	
	s.TreeSelected = lipgloss.NewStyle().
		Background(ColorAccent).
		Foreground(ColorBG).
		Bold(true)
	
	s.TreeUnfocused = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(ColorAccent).
		PaddingLeft(1).
		Foreground(ColorWhite)

	s.TreeHeader = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorWhite)

	s.BadgeNormal = lipgloss.NewStyle().
		Foreground(ColorMint).
		Bold(true)

	s.BadgeCritical = lipgloss.NewStyle().
		Foreground(ColorRed).
		Bold(true)

	s.TabRow = lipgloss.NewStyle().
		MarginBottom(1)

	s.Tab = lipgloss.NewStyle().
		Padding(0, 2).
		Foreground(ColorDim).
		Background(ColorSubtle).
		MarginRight(1)

	s.ActiveTab = s.Tab.Copy().
		Foreground(ColorWhite).
		Background(ColorDim).
		Bold(true)

	s.FocusedTab = s.Tab.Copy().
		Foreground(ColorBG).
		Background(ColorGold).
		Bold(true)

	s.SectionHeader = lipgloss.NewStyle().
		Background(ColorAccent).
		Foreground(ColorBG).
		Bold(true).
		Padding(0, 1).
		MarginTop(1).
		MarginBottom(1)

	s.KeyStyle = lipgloss.NewStyle().Foreground(ColorGold)
	s.ValueStyle = lipgloss.NewStyle().Foreground(ColorWhite)
	
	s.Label = lipgloss.NewStyle().Foreground(ColorGold)
	s.Value = lipgloss.NewStyle().Foreground(ColorWhite)

	s.BarEmpty = lipgloss.NewStyle().Foreground(ColorDim)

	s.Footer = lipgloss.NewStyle().
		Height(1).
		MarginTop(1).
		Foreground(ColorDim)

	s.FooterPill = lipgloss.NewStyle().
		MarginRight(3)

	s.KeyCap = lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)

	s.Modal = lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(ColorAccent).
		Padding(1, 2).
		Background(ColorSubtle)

	return s
}
