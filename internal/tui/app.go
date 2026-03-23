package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/libvirt/libvirt-go"
	"virt-tui/internal/models"
	"virt-tui/internal/tui/components"
	"virt-tui/internal/tui/styles"
	"virt-tui/internal/tui/views"
	"virt-tui/internal/virt"
)

type FocusArea int

const (
	AreaSidebar FocusArea = iota
	AreaTabs
	AreaContent
)

type SidebarView int

const (
	SidebarVMs SidebarView = iota
	SidebarNetworks
	SidebarPools
	SidebarHost
)

type TabIndex int

const (
	TabSummary TabIndex = iota
	TabConsole
	TabHardware
	TabSnapshots
)

var TabNames = []string{"Summary", "Console", "Hardware", "Snapshots"}

type Model struct {
	Client     virt.VirtManager
	Err        error
	Quitting   bool
	Width      int
	Height     int
	StatusMsg  string
	Styles     styles.Styles

	// Sub-Models
	VMView      views.VMModel
	NetworkView views.NetworkModel
	PoolView    views.StorageModel
	HostView    views.HostModel

	// UI Components
	Toast    components.ToastModel
	Help     components.HelpModel
	Wizard   components.WizardModel
	Menu     components.MenuModel
	Viewport viewport.Model

	ResourceWizard  components.WizardModel
	DiskWizard      components.WizardModel
	InterfaceWizard components.WizardModel
	NetWizard       components.WizardModel
	PoolWizard      components.WizardModel
	VolumeWizard    components.WizardModel

	// Navigation
	Focus       FocusArea
	ActiveTab   TabIndex
	SidebarMode SidebarView

	// Console state
	ConsoleStream *libvirt.Stream
	ConsoleBuf    strings.Builder
	ConsoleChan   chan string

	// Modal state
	ShowConfirm    bool
	ConfirmTitle   string
	ConfirmAction  func() tea.Cmd

	// Other lists (Snapshots, Media, etc.)
	SnapshotList  list.Model
	MediaList     list.Model
	HardwareList  list.Model

	ShowSnapInput bool
	SnapInput     textinput.Model

	ShowCloneInput bool
	CloneInput     textinput.Model

	ShowResourceWizard  bool
	ShowDiskWizard      bool
	ShowInterfaceWizard bool
	ShowNetWizard       bool
	ShowPoolWizard      bool
	ShowVolumeWizard    bool

	ShowDiskRemovePicker      bool
	ShowInterfaceRemovePicker bool

	// Media state
	ShowMediaPicker bool
}

func NewModel(client virt.VirtManager) Model {
	s := styles.DefaultStyles()
	
	sl := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	sl.SetShowTitle(false)
	sl.SetShowHelp(false)

	ml := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	ml.SetShowTitle(false)
	ml.SetShowHelp(false)

	hl := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	hl.SetShowTitle(false)
	hl.SetShowHelp(false)

	m := Model{
		Client:       client,
		Styles:       s,
		VMView:       views.NewVMModel(s),
		NetworkView:  views.NewNetworkModel(s),
		PoolView:     views.NewStorageModel(s),
		HostView:     views.NewHostModel(s),
		SnapshotList: sl,
		MediaList:    ml,
		HardwareList: hl,
		Viewport:     viewport.New(0, 0),
		SnapInput:    textinput.New(),
		CloneInput:   textinput.New(),
		Toast:        components.NewToastModel(),
		Help:         components.NewHelpModel(),
	}

	m.Wizard = components.NewWizardModel("VM Creation Wizard", []components.WizardStep{
		{Title: "VM Name", Input: m.newWizardInput("my-new-vm")},
		{Title: "vCPUs", Input: m.newWizardInput("2")},
		{Title: "Memory (KB)", Input: m.newWizardInput("2097152")},
		{Title: "ISO Path", Input: m.newWizardInput("/var/lib/libvirt/images/debian.iso")},
		{Title: "Disk Path", Input: m.newWizardInput("/var/lib/libvirt/images/my-new-vm.qcow2")},
	}, func(values map[string]string) tea.Cmd {
		return func() tea.Msg {
			var memory uint64
			fmt.Sscanf(values["Memory (KB)"], "%d", &memory)
			var vcpus uint
			fmt.Sscanf(values["vCPUs"], "%d", &vcpus)

			err := client.CreateVM(values["VM Name"], memory, vcpus, values["ISO Path"], values["Disk Path"])
			return models.RefreshMsg{Err: err}
		}
	})

	m.ResourceWizard = components.NewWizardModel("Edit Resources", []components.WizardStep{
		{Title: "vCPUs", Input: m.newWizardInput("2")},
		{Title: "Memory (KB)", Input: m.newWizardInput("2097152")},
	}, func(values map[string]string) tea.Cmd {
		return func() tea.Msg {
			if sel, ok := m.VMView.List.SelectedItem().(models.VMItem); ok {
				var memory uint64
				fmt.Sscanf(values["Memory (KB)"], "%d", &memory)
				var vcpus uint
				fmt.Sscanf(values["vCPUs"], "%d", &vcpus)

				err1 := m.Client.SetVCPUs(sel.VM.Name, vcpus)
				err2 := m.Client.SetMemory(sel.VM.Name, memory)
				
				if err1 != nil { return models.RefreshMsg{Err: err1} }
				return models.RefreshMsg{Err: err2}
			}
			return nil
		}
	})

	m.DiskWizard = components.NewWizardModel("Add Disk", []components.WizardStep{
		{Title: "Source Path", Input: m.newWizardInput("/var/lib/libvirt/images/disk.qcow2")},
		{Title: "Target Dev", Input: m.newWizardInput("vdb")},
		{Title: "Bus", Input: m.newWizardInput("virtio")},
	}, func(values map[string]string) tea.Cmd {
		return func() tea.Msg {
			if sel, ok := m.VMView.List.SelectedItem().(models.VMItem); ok {
				disk := virt.VMDisk{
					Device: "disk",
					Source: values["Source Path"],
					Target: values["Target Dev"],
					Bus:    values["Bus"],
				}
				err := m.Client.AttachDisk(sel.VM.Name, disk)
				return models.RefreshMsg{Err: err}
			}
			return nil
		}
	})

	m.InterfaceWizard = components.NewWizardModel("Add Interface", []components.WizardStep{
		{Title: "Source Network", Input: m.newWizardInput("default")},
		{Title: "Model", Input: m.newWizardInput("virtio")},
		{Title: "MAC", Input: m.newWizardInput("52:54:00:...")},
	}, func(values map[string]string) tea.Cmd {
		return func() tea.Msg {
			if sel, ok := m.VMView.List.SelectedItem().(models.VMItem); ok {
				iface := virt.VMInterface{
					Source: values["Source Network"],
					Model:  values["Model"],
					MAC:    values["MAC"],
				}
				err := m.Client.AttachInterface(sel.VM.Name, iface)
				return models.RefreshMsg{Err: err}
			}
			return nil
		}
	})

	m.NetWizard = components.NewWizardModel("Create Network", []components.WizardStep{
		{Title: "Name", Input: m.newWizardInput("my-net")},
		{Title: "Bridge", Input: m.newWizardInput("virbr1")},
		{Title: "IP Range (Gateway)", Input: m.newWizardInput("192.168.100.1")},
	}, func(values map[string]string) tea.Cmd {
		return func() tea.Msg {
			err := m.Client.CreateNetwork(values["Name"], values["Bridge"], values["IP Range (Gateway)"])
			return models.NetworkListMsg{Err: err}
		}
	})

	m.PoolWizard = components.NewWizardModel("Create Storage Pool", []components.WizardStep{
		{Title: "Name", Input: m.newWizardInput("my-pool")},
		{Title: "Path", Input: m.newWizardInput("/var/lib/libvirt/images/my-pool")},
	}, func(values map[string]string) tea.Cmd {
		return func() tea.Msg {
			err := m.Client.CreateStoragePool(values["Name"], values["Path"])
			return models.PoolListMsg{Err: err}
		}
	})

	m.VolumeWizard = components.NewWizardModel("Create Volume", []components.WizardStep{
		{Title: "Name", Input: m.newWizardInput("my-vol.qcow2")},
		{Title: "Capacity (GB)", Input: m.newWizardInput("20")},
	}, func(values map[string]string) tea.Cmd {
		return func() tea.Msg {
			if sel, ok := m.PoolView.List.SelectedItem().(models.PoolItem); ok {
				var capGB uint64
				fmt.Sscanf(values["Capacity (GB)"], "%d", &capGB)
				err := m.Client.CreateStorageVolume(sel.Pool.Name, values["Name"], capGB*1024*1024*1024)
				if err != nil { return models.PoolListMsg{Err: err} }
				return RefreshPools(m.Client)
			}
			return nil
		}
	})

	m.Menu = components.NewMenuModel()

	return m
}

func (m Model) newWizardInput(placeholder string) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()
	return ti
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		Tick(),
		RefreshVMs(m.Client),
		RefreshNetworks(m.Client),
		RefreshPools(m.Client),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case models.ConsoleDataMsg:
		m.ConsoleBuf.WriteString(string(msg))
		lines := strings.Split(m.ConsoleBuf.String(), "\n")
		if len(lines) > 1000 {
			m.ConsoleBuf.Reset()
			m.ConsoleBuf.WriteString(strings.Join(lines[len(lines)-1000:], "\n"))
		}
		m.Viewport.SetContent(m.ConsoleBuf.String())
		m.Viewport.GotoBottom()
		return m, ListenForConsole(m.ConsoleChan)

	case tea.KeyMsg:
		if m.Menu.Show {
			var cmd tea.Cmd
			m.Menu, cmd = m.Menu.Update(msg)
			return m, cmd
		}

		if m.Wizard.Show {
			var cmd tea.Cmd
			m.Wizard, cmd = m.Wizard.Update(msg)
			return m, cmd
		}

		if m.ShowResourceWizard {
			var cmd tea.Cmd
			m.ResourceWizard, cmd = m.ResourceWizard.Update(msg)
			if m.ResourceWizard.Done {
				m.ShowResourceWizard = false
				m.ResourceWizard.Done = false
			}
			return m, cmd
		}

		if m.ShowDiskWizard {
			var cmd tea.Cmd
			m.DiskWizard, cmd = m.DiskWizard.Update(msg)
			if m.DiskWizard.Done {
				m.ShowDiskWizard = false
				m.DiskWizard.Done = false
			}
			return m, cmd
		}

		if m.ShowInterfaceWizard {
			var cmd tea.Cmd
			m.InterfaceWizard, cmd = m.InterfaceWizard.Update(msg)
			if m.InterfaceWizard.Done {
				m.ShowInterfaceWizard = false
				m.InterfaceWizard.Done = false
			}
			return m, cmd
		}

		if m.ShowNetWizard {
			var cmd tea.Cmd
			m.NetWizard, cmd = m.NetWizard.Update(msg)
			if m.ShowNetWizard && m.NetWizard.Done {
				m.ShowNetWizard = false
				m.NetWizard.Done = false
			}
			return m, cmd
		}

		if m.ShowPoolWizard {
			var cmd tea.Cmd
			m.PoolWizard, cmd = m.PoolWizard.Update(msg)
			if m.ShowPoolWizard && m.PoolWizard.Done {
				m.ShowPoolWizard = false
				m.PoolWizard.Done = false
			}
			return m, cmd
		}

		if m.ShowVolumeWizard {
			var cmd tea.Cmd
			m.VolumeWizard, cmd = m.VolumeWizard.Update(msg)
			if m.ShowVolumeWizard && m.VolumeWizard.Done {
				m.ShowVolumeWizard = false
				m.VolumeWizard.Done = false
			}
			return m, cmd
		}

		if m.Help.Show {
			switch msg.String() {
			case "?", "esc":
				m.Help.Show = false
			}
			return m, nil
		}

		if m.ShowConfirm {
			switch msg.String() {
			case "y", "Y":
				actionCmd := m.ConfirmAction()
				m.ShowConfirm = false
				return m, actionCmd
			case "n", "N", "esc":
				m.ShowConfirm = false
				return m, nil
			}
			return m, nil
		}

		if m.ShowSnapInput {
			switch msg.String() {
			case "enter":
				if m.SnapInput.Value() != "" {
					if sel, ok := m.VMView.List.SelectedItem().(models.VMItem); ok {
						vmName := sel.VM.Name
						snapName := m.SnapInput.Value()
						m.ShowSnapInput = false
						return m, func() tea.Msg {
							err := m.Client.CreateSnapshot(vmName, snapName)
							if err != nil { return models.RefreshSnapsMsg{Err: err} }
							return models.RefreshSnapsMsg{VMName: vmName}
						}
					}
				}
				m.ShowSnapInput = false
				return m, nil
			case "esc":
				m.ShowSnapInput = false
				return m, nil
			}
			var cmd tea.Cmd
			m.SnapInput, cmd = m.SnapInput.Update(msg)
			return m, cmd
		}

		if m.ShowCloneInput {
			switch msg.String() {
			case "enter":
				if m.CloneInput.Value() != "" {
					if sel, ok := m.VMView.List.SelectedItem().(models.VMItem); ok {
						vmName := sel.VM.Name
						newName := m.CloneInput.Value()
						m.ShowCloneInput = false
						return m, func() tea.Msg {
							err := m.Client.CloneVM(vmName, newName)
							return models.RefreshMsg{Err: err}
						}
					}
				}
				m.ShowCloneInput = false
				return m, nil
			case "esc":
				m.ShowCloneInput = false
				return m, nil
			}
			var cmd tea.Cmd
			m.CloneInput, cmd = m.CloneInput.Update(msg)
			return m, cmd
		}

		if m.ShowMediaPicker {
			switch msg.String() {
			case "enter":
				if selVM, ok := m.VMView.List.SelectedItem().(models.VMItem); ok {
					if selVol, ok := m.MediaList.SelectedItem().(models.VolItem); ok {
						m.ShowMediaPicker = false
						err := m.Client.ChangeMedia(selVM.VM.Name, selVol.Vol.Path)
						if err != nil { m.Err = err } else { m.StatusMsg = "Media inserted into " + selVM.VM.Name }
						return m, nil
					}
				}
				m.ShowMediaPicker = false
				return m, nil
			case "esc":
				m.ShowMediaPicker = false
				return m, nil
			}
			var cmd tea.Cmd
			m.MediaList, cmd = m.MediaList.Update(msg)
			return m, cmd
		}

		if m.ShowDiskRemovePicker {
			switch msg.String() {
			case "enter":
				if selVM, ok := m.VMView.List.SelectedItem().(models.VMItem); ok {
					if selDisk, ok := m.HardwareList.SelectedItem().(models.DiskItem); ok {
						m.ShowDiskRemovePicker = false
						err := m.Client.DetachDisk(selVM.VM.Name, selDisk.Disk.Target)
						return m, func() tea.Msg { return models.RefreshMsg{Err: err} }
					}
				}
				m.ShowDiskRemovePicker = false
				return m, nil
			case "esc":
				m.ShowDiskRemovePicker = false
				return m, nil
			}
			var cmd tea.Cmd
			m.HardwareList, cmd = m.HardwareList.Update(msg)
			return m, cmd
		}

		if m.ShowInterfaceRemovePicker {
			switch msg.String() {
			case "enter":
				if selVM, ok := m.VMView.List.SelectedItem().(models.VMItem); ok {
					if selIface, ok := m.HardwareList.SelectedItem().(models.IfaceItem); ok {
						m.ShowInterfaceRemovePicker = false
						err := m.Client.DetachInterface(selVM.VM.Name, selIface.Iface.MAC)
						return m, func() tea.Msg { return models.RefreshMsg{Err: err} }
					}
				}
				m.ShowInterfaceRemovePicker = false
				return m, nil
			case "esc":
				m.ShowInterfaceRemovePicker = false
				return m, nil
			}
			var cmd tea.Cmd
			m.HardwareList, cmd = m.HardwareList.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "q", "ctrl+c":
			m.CloseConsole()
			m.Quitting = true
			return m, tea.Quit
		
		case "?":
			m.Help.Show = true
			return m, nil

		case "C":
			m.Wizard.Show = true
			m.Wizard.ActiveStep = 0
			m.Wizard.Steps[0].Input.Focus()
			return m, nil

		case "v":
			m.SidebarMode = SidebarVMs
			return m, nil
		case "n":
			m.SidebarMode = SidebarNetworks
			return m, nil
		case "p":
			m.SidebarMode = SidebarPools
			return m, nil
		case "h":
			m.SidebarMode = SidebarHost
			return m, nil

		case "tab":
			m.Focus = (m.Focus + 1) % 3
			return m, nil

		case "left":
			if m.Focus == AreaTabs || m.Focus == AreaContent {
				if m.ActiveTab > 0 { m.ActiveTab-- }
				return m, m.HandleTabChange()
			}
		case "right":
			if m.Focus == AreaTabs || m.Focus == AreaContent {
				if m.ActiveTab < TabSnapshots { m.ActiveTab++ }
				return m, m.HandleTabChange()
			}

		case "1", "2", "3", "4":
			m.ActiveTab = TabIndex(msg.String()[0] - '1')
			return m, m.HandleTabChange()

		case "R":
			return m, m.Refresh()

		case " ": // Toggle Selection
			if m.Focus == AreaSidebar {
				switch m.SidebarMode {
				case SidebarVMs:
					if items := m.VMView.List.Items(); len(items) > 0 {
						idx := m.VMView.List.Index()
						item := items[idx].(models.VMItem)
						item.Selected = !item.Selected
						items[idx] = item
						m.VMView.List.SetItems(items)
					}
				case SidebarNetworks:
					if items := m.NetworkView.List.Items(); len(items) > 0 {
						idx := m.NetworkView.List.Index()
						item := items[idx].(models.NetItem)
						item.Selected = !item.Selected
						items[idx] = item
						m.NetworkView.List.SetItems(items)
					}
				case SidebarPools:
					if items := m.PoolView.List.Items(); len(items) > 0 {
						idx := m.PoolView.List.Index()
						item := items[idx].(models.PoolItem)
						item.Selected = !item.Selected
						items[idx] = item
						m.PoolView.List.SetItems(items)
					}
				}
			}
			return m, nil

		case "m": // Open Context Menu
			if m.Focus == AreaSidebar {
				switch m.SidebarMode {
				case SidebarVMs:
					if sel, ok := m.VMView.List.SelectedItem().(models.VMItem); ok {
						m.openVMMenu(sel.VM)
					}
				case SidebarNetworks:
					if sel, ok := m.NetworkView.List.SelectedItem().(models.NetItem); ok {
						m.openNetworkMenu(sel.Net)
					}
				case SidebarPools:
					if sel, ok := m.PoolView.List.SelectedItem().(models.PoolItem); ok {
						m.openPoolMenu(sel.Pool)
					}
				}
			} else if m.Focus == AreaContent && m.ActiveTab == TabSnapshots {
				if selVM, ok := m.VMView.List.SelectedItem().(models.VMItem); ok {
					if selSnap, ok := m.SnapshotList.SelectedItem().(models.SnapItem); ok {
						m.openSnapshotMenu(selVM.VM.Name, selSnap.Snap.Name)
					}
				}
			}
			return m, nil

		}

		if m.Focus == AreaSidebar {
			var cmd tea.Cmd
			switch m.SidebarMode {
			case SidebarVMs:
				m.VMView, cmd = m.VMView.Update(msg)
				if i, ok := m.VMView.List.SelectedItem().(models.VMItem); ok {
					cmds = append(cmds, FetchConfig(m.Client, i.VM.Name))
					if m.ActiveTab == TabSnapshots {
						cmds = append(cmds, RefreshSnapshots(m.Client, i.VM.Name))
					}
				}
			case SidebarNetworks:
				m.NetworkView, cmd = m.NetworkView.Update(msg)
			case SidebarPools:
				m.PoolView, cmd = m.PoolView.Update(msg)
			}
			cmds = append(cmds, cmd)
		} else if m.Focus == AreaContent {
			if m.ActiveTab == TabSnapshots {
				var cmd tea.Cmd
				m.SnapshotList, cmd = m.SnapshotList.Update(msg)
				cmds = append(cmds, cmd)
			} else if m.ActiveTab == TabConsole {
				var cmd tea.Cmd
				m.Viewport, cmd = m.Viewport.Update(msg)
				cmds = append(cmds, cmd)
			}
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.VMView.List.SetSize(styles.SidebarWidth, m.Height-5)
		m.NetworkView.List.SetSize(styles.SidebarWidth, m.Height-5)
		m.PoolView.List.SetSize(styles.SidebarWidth, m.Height-5)
		
		m.Viewport.Width = (m.Width * 4 / 5) - 6
		m.Viewport.Height = m.Height - 10
		m.SnapshotList.SetSize((m.Width * 4 / 5) - 6, m.Height-12)
		m.MediaList.SetSize(m.Width/2-10, m.Height-12)
		m.HardwareList.SetSize(m.Width/2-10, m.Height-12)

	case models.RefreshMsg:
		if msg.Err != nil {
			m.Toast.Set(fmt.Sprintf("Refresh VMs failed: %v", msg.Err), components.ToastError, 3*time.Second)
		} else if msg.VMs != nil {
			items := make([]list.Item, len(msg.VMs))
			for i, v := range msg.VMs { items[i] = models.VMItem{VM: v} }
			m.VMView.List.SetItems(items)
		}

	case models.NetworkListMsg:
		if msg.Err != nil {
			m.Toast.Set(fmt.Sprintf("Refresh Networks failed: %v", msg.Err), components.ToastError, 3*time.Second)
		} else if msg.Networks != nil {
			items := make([]list.Item, len(msg.Networks))
			for i, n := range msg.Networks { items[i] = models.NetItem{Net: n} }
			m.NetworkView.List.SetItems(items)
		}

	case models.PoolListMsg:
		if msg.Err != nil {
			m.Toast.Set(fmt.Sprintf("Refresh Pools failed: %v", msg.Err), components.ToastError, 3*time.Second)
		} else if msg.Pools != nil {
			items := make([]list.Item, len(msg.Pools))
			for i, p := range msg.Pools { items[i] = models.PoolItem{Pool: p} }
			m.PoolView.List.SetItems(items)
		}

	case models.StatsMsg:
		if msg.Err == nil {
			m.VMView.Stats = msg.Stats
			m.VMView.CPUHistory = append(m.VMView.CPUHistory, msg.Stats.CPUPercent)
			if len(m.VMView.CPUHistory) > 40 { m.VMView.CPUHistory = m.VMView.CPUHistory[1:] }
			if msg.Stats.MemoryTotal > 0 {
				m.VMView.MemHistory = append(m.VMView.MemHistory, float64(msg.Stats.MemoryUsed)/float64(msg.Stats.MemoryTotal)*100)
				if len(m.VMView.MemHistory) > 40 { m.VMView.MemHistory = m.VMView.MemHistory[1:] }
			}
		}

	case models.ConfigMsg:
		if msg.Err == nil { m.VMView.Config = msg.Config }

	case models.SnapshotListMsg:
		if msg.Err == nil {
			items := make([]list.Item, len(msg.Snapshots))
			for i, s := range msg.Snapshots { items[i] = models.SnapItem{Snap: s} }
			m.SnapshotList.SetItems(items)
		}

	case models.HostStatsMsg:
		if msg.Err == nil { m.HostView.Stats = msg.Stats }

	case models.VolListMsg:
		if msg.Err == nil {
			items := make([]list.Item, len(msg.Vols))
			for i, v := range msg.Vols { items[i] = models.VolItem{Vol: v} }
			m.MediaList.SetItems(items)
		}

	case models.TickMsg:
		m.Toast.Update()
		if i, ok := m.VMView.List.SelectedItem().(models.VMItem); ok {
			cmds = append(cmds, FetchStats(m.Client, i.VM.Name, m.VMView.Stats))
		}
		cmds = append(cmds, RefreshNetworks(m.Client))
		cmds = append(cmds, RefreshPools(m.Client))
		cmds = append(cmds, FetchHostStats(m.Client))
		cmds = append(cmds, Tick())
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) HandleTabChange() tea.Cmd {
	if m.ActiveTab == TabConsole {
		if sel, ok := m.VMView.List.SelectedItem().(models.VMItem); ok {
			m.CloseConsole()
			stream, err := m.Client.OpenConsole(sel.VM.Name)
			if err == nil {
				m.ConsoleStream = stream
				m.ConsoleChan = make(chan string, 10)
				m.ConsoleBuf.Reset()
				go func(s *libvirt.Stream, c chan string) {
					buf := make([]byte, 1024)
					for {
						n, err := s.Recv(buf)
						if err != nil { break }
						if n > 0 { c <- string(buf[:n]) }
					}
				}(m.ConsoleStream, m.ConsoleChan)
				return ListenForConsole(m.ConsoleChan)
			}
		}
	} else {
		m.CloseConsole()
	}
	
	if m.ActiveTab == TabSnapshots {
		if sel, ok := m.VMView.List.SelectedItem().(models.VMItem); ok {
			return RefreshSnapshots(m.Client, sel.VM.Name)
		}
	}
	return nil
}

func (m *Model) CloseConsole() {
	if m.ConsoleStream != nil { m.ConsoleStream.Free(); m.ConsoleStream = nil }
	m.ConsoleChan = nil
}

func (m Model) Refresh() tea.Cmd {
	return RefreshVMs(m.Client)
}

func (m Model) FetchAllVolumes() tea.Cmd {
	return func() tea.Msg {
		pools, _ := m.Client.ListStoragePools()
		var allVols []virt.StorageVolume
		for _, p := range pools {
			if p.Active {
				vols, _ := m.Client.ListStorageVolumes(p.Name)
				allVols = append(allVols, vols...)
			}
		}
		return models.VolListMsg{Vols: allVols}
	}
}

// --- View ---
func (m Model) View() string {
	if m.Quitting { return "Exiting..." }

	sidebarW := styles.SidebarWidth
	contentW := m.Width - sidebarW - 1
	mainH := m.Height - 3

	// Sidebar
	sidebar := m.Styles.Sidebar.Height(mainH).Render(m.renderSidebar())

	// Main Panel
	var mainPanel string
	isSelected := false
	switch m.SidebarMode {
	case SidebarVMs:
		_, isSelected = m.VMView.List.SelectedItem().(models.VMItem)
	case SidebarNetworks:
		_, isSelected = m.NetworkView.List.SelectedItem().(models.NetItem)
	case SidebarPools:
		_, isSelected = m.PoolView.List.SelectedItem().(models.PoolItem)
	case SidebarHost:
		isSelected = true
	}

	if isSelected {
		if m.SidebarMode == SidebarVMs {
			tabs := m.renderTabs(contentW)
			body := m.renderTabContent(contentW, mainH-3)
			mainPanel = lipgloss.JoinVertical(lipgloss.Left, tabs, body)
		} else {
			mainPanel = m.renderTabContent(contentW, mainH)
		}
	} else {
		mainPanel = lipgloss.Place(contentW, mainH, lipgloss.Center, lipgloss.Center, "Select a resource from the tree")
	}
	content := m.Styles.ContentArea.Width(contentW).Height(mainH).Render(mainPanel)

	header := m.renderHeader()
	footer := m.renderFooter()
	
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
	
	view := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	finalView := m.Styles.Main.Render(view)

	if m.ShowConfirm {
		modal := m.Styles.Modal.Render(lipgloss.JoinVertical(lipgloss.Center, m.ConfirmTitle, "", "Confirm with (y/n)"))
		return m.overlay(finalView, modal)
	}

	if m.ShowSnapInput {
		input := m.Styles.Modal.Render(lipgloss.JoinVertical(lipgloss.Center, m.Styles.SectionHeader.Render(" NEW SNAPSHOT "), "", m.SnapInput.View(), "", "Enter: Create | Esc: Cancel"))
		return m.overlay(finalView, input)
	}

	if m.ShowCloneInput {
		input := m.Styles.Modal.Render(lipgloss.JoinVertical(lipgloss.Center, m.Styles.SectionHeader.Render(" CLONE VM "), "", m.CloneInput.View(), "", "Enter: Clone | Esc: Cancel"))
		return m.overlay(finalView, input)
	}

	if m.ShowMediaPicker {
		picker := m.Styles.Modal.Render(lipgloss.JoinVertical(lipgloss.Center, m.Styles.SectionHeader.Render(" SELECT MEDIA "), "", m.MediaList.View(), "", "Enter: Insert | Esc: Cancel"))
		return m.overlay(finalView, picker)
	}

	if m.ShowDiskRemovePicker {
		picker := m.Styles.Modal.Render(lipgloss.JoinVertical(lipgloss.Center, m.Styles.SectionHeader.Render(" REMOVE DISK "), "", m.HardwareList.View(), "", "Enter: Remove | Esc: Cancel"))
		return m.overlay(finalView, picker)
	}

	if m.ShowInterfaceRemovePicker {
		picker := m.Styles.Modal.Render(lipgloss.JoinVertical(lipgloss.Center, m.Styles.SectionHeader.Render(" REMOVE INTERFACE "), "", m.HardwareList.View(), "", "Enter: Remove | Esc: Cancel"))
		return m.overlay(finalView, picker)
	}

	if m.Wizard.Show {
		return m.overlay(finalView, m.Wizard.View(m.Width, m.Height))
	}

	if m.ShowResourceWizard {
		return m.overlay(finalView, m.ResourceWizard.View(m.Width, m.Height))
	}

	if m.ShowDiskWizard {
		return m.overlay(finalView, m.DiskWizard.View(m.Width, m.Height))
	}

	if m.ShowInterfaceWizard {
		return m.overlay(finalView, m.InterfaceWizard.View(m.Width, m.Height))
	}

	if m.ShowNetWizard {
		return m.overlay(finalView, m.NetWizard.View(m.Width, m.Height))
	}

	if m.ShowPoolWizard {
		return m.overlay(finalView, m.PoolWizard.View(m.Width, m.Height))
	}

	if m.ShowVolumeWizard {
		return m.overlay(finalView, m.VolumeWizard.View(m.Width, m.Height))
	}

	if m.Menu.Show {
		return m.overlay(finalView, m.Menu.View(m.Width, m.Height))
	}

	if m.Help.Show {
		return m.overlay(finalView, m.Help.View(m.Width, m.Height))
	}

	return finalView
}

func (m Model) renderHeader() string {
	title := " VIRT-TUI "
	left := m.Styles.Header.Render(title)

	var metrics string
	var badgeStyle lipgloss.Style
	if m.HostView.Stats.MemoryTotal > 0 {
		used := m.HostView.Stats.MemoryTotal - m.HostView.Stats.MemoryFree
		perc := float64(used) / float64(m.HostView.Stats.MemoryTotal) * 100
		
		badgeStyle = m.Styles.BadgeNormal
		if perc > 90 {
			badgeStyle = m.Styles.BadgeCritical
		}
		metrics = badgeStyle.Render(fmt.Sprintf("RAM: %.1f%% ", perc))
	} else {
		metrics = m.Styles.BadgeNormal.Render("RAM: --% ")
	}

	// Calculate filler width
	fillerWidth := m.Width - lipgloss.Width(left) - lipgloss.Width(metrics)
	if fillerWidth < 0 { fillerWidth = 0 }
	
	filler := lipgloss.NewStyle().
		Width(fillerWidth).
		Background(styles.ColorAccent).
		Render("")

	header := lipgloss.JoinHorizontal(lipgloss.Top, left, filler, lipgloss.NewStyle().Background(styles.ColorAccent).Render(metrics))

	// Full-width container with bottom border
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(styles.ColorSubtle).
		Render(header)
}

func (m Model) renderSidebar() string {
	var rows []string
	sidebarW := styles.SidebarWidth

	// Category Headers
	vStyle, nStyle, pStyle, hStyle := m.Styles.TreeFolder, m.Styles.TreeFolder, m.Styles.TreeFolder, m.Styles.TreeFolder
	switch m.SidebarMode {
	case SidebarVMs:
		vStyle = m.Styles.BadgeNormal
	case SidebarNetworks:
		nStyle = m.Styles.BadgeNormal
	case SidebarPools:
		pStyle = m.Styles.BadgeNormal
	case SidebarHost:
		hStyle = m.Styles.BadgeNormal
	}

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		vStyle.Render(" [V] "),
		nStyle.Render(" [N] "),
		pStyle.Render(" [P] "),
		hStyle.Render(" [H] "),
	)
	rows = append(rows, header)
	rows = append(rows, "")

	switch m.SidebarMode {
	case SidebarVMs:
		rows = append(rows, m.renderVMList(sidebarW)...)
	case SidebarNetworks:
		rows = append(rows, m.renderNetworkList(sidebarW)...)
	case SidebarPools:
		rows = append(rows, m.renderPoolList(sidebarW)...)
	case SidebarHost:
		rows = append(rows, m.Styles.TreeHeader.Render("󱩊 Host Info"))
		rows = append(rows, m.Styles.TreeSelected.Width(sidebarW).Render("  • Details"))
	}

	return strings.Join(rows, "\n")
}

func (m Model) renderVMList(w int) []string {
	var rows []string
	rows = append(rows, m.Styles.TreeHeader.Render("󰇄 Virtual Machines"))
	items := m.VMView.List.Items()
	for i, li := range items {
		vm := li.(models.VMItem)
		glyph := "○"
		if vm.VM.State == libvirt.DOMAIN_RUNNING { glyph = "●" }
		as := ""
		if vm.VM.Autostart { as = "*" }
		
		sel := "[ ] "
		if vm.Selected { sel = "[x] " }
		
		prefix := " ├── "
		if i == len(items)-1 { prefix = " └── " }
		label := fmt.Sprintf("%s%s%s %s%s", prefix, sel, glyph, vm.VM.Name, as)
		rows = append(rows, m.renderSidebarRow(i, m.VMView.List.Index(), label, w))
	}
	return rows
}

func (m Model) renderNetworkList(w int) []string {
	var rows []string
	rows = append(rows, m.Styles.TreeHeader.Render("󱂇 Virtual Networks"))
	items := m.NetworkView.List.Items()
	for i, li := range items {
		net := li.(models.NetItem)
		glyph := "○"
		if net.Net.Active { glyph = "●" }
		as := ""
		if net.Net.Autostart { as = "*" }

		sel := "[ ] "
		if net.Selected { sel = "[x] " }

		prefix := " ├── "
		if i == len(items)-1 { prefix = " └── " }
		label := fmt.Sprintf("%s%s%s %s%s", prefix, sel, glyph, net.Net.Name, as)
		rows = append(rows, m.renderSidebarRow(i, m.NetworkView.List.Index(), label, w))
	}
	return rows
}

func (m Model) renderPoolList(w int) []string {
	var rows []string
	rows = append(rows, m.Styles.TreeHeader.Render("󰋊 Storage Pools"))
	items := m.PoolView.List.Items()
	for i, li := range items {
		pool := li.(models.PoolItem)
		glyph := "○"
		if pool.Pool.Active { glyph = "●" }
		as := ""
		if pool.Pool.Autostart { as = "*" }

		sel := "[ ] "
		if pool.Selected { sel = "[x] " }

		prefix := " ├── "
		if i == len(items)-1 { prefix = " └── " }
		label := fmt.Sprintf("%s%s%s %s%s", prefix, sel, glyph, pool.Pool.Name, as)
		rows = append(rows, m.renderSidebarRow(i, m.PoolView.List.Index(), label, w))
	}
	return rows
}

func (m Model) renderSidebarRow(idx, selectedIdx int, label string, w int) string {
	if idx == selectedIdx {
		if m.Focus == AreaSidebar {
			return m.Styles.TreeSelected.Width(w).Render(label)
		}
		return m.Styles.TreeUnfocused.Width(w).Render(label)
	}
	return m.Styles.TreeVM.Width(w).Render(label)
}

func (m Model) renderTabs(w int) string {
	var tabs []string
	for i, name := range TabNames {
		style := m.Styles.Tab
		if TabIndex(i) == m.ActiveTab { 
			if m.Focus == AreaTabs {
				style = m.Styles.FocusedTab
			} else {
				style = m.Styles.ActiveTab 
			}
		}
		tabs = append(tabs, style.Render(name))
	}
	return m.Styles.TabRow.Width(w).Render(lipgloss.JoinHorizontal(lipgloss.Top, tabs...))
}

func (m Model) renderTabContent(w, h int) string {
	switch m.SidebarMode {
	case SidebarVMs:
		if m.ActiveTab == TabConsole {
			return m.Viewport.View()
		}
		if m.ActiveTab == TabSnapshots {
			return m.SnapshotList.View()
		}
		return m.VMView.View(w, h, int(m.ActiveTab), m.Focus == AreaContent)
	case SidebarNetworks:
		return m.NetworkView.View(w, h)
	case SidebarPools:
		return m.PoolView.View(w, h)
	case SidebarHost:
		return m.HostView.View(w, h)
	}
	return ""
}

func (m Model) overlay(base, overlayStr string) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlayStr, "\n")

	overlayHeight := len(overlayLines)
	startY := (m.Height - overlayHeight) / 2

	res := make([]string, len(baseLines))
	copy(res, baseLines)
	
	for i, oLine := range overlayLines {
		y := startY + i
		if y >= 0 && y < len(res) {
			res[y] = lipgloss.PlaceHorizontal(m.Width, lipgloss.Center, oLine)
		}
	}

	return strings.Join(res, "\n")
}

func (m Model) renderFooter() string {
	groups := []struct {
		title string
		keys  [][]string
	}{
		{"Nav", [][]string{{"tab", "Focus"}, {"arrows", "Nav"}}},
		{"Action", [][]string{{"m/Space", "Menu"}, {"C", "Create"}, {"R", "Refresh"}}},
		{"Views", [][]string{{"v", "VMs"}, {"n", "Nets"}, {"p", "Pools"}}},
		{"App", [][]string{{"q", "Quit"}}},
	}

	var items []string
	for _, g := range groups {
		for _, k := range g.keys {
			items = append(items, fmt.Sprintf("%s %s", m.Styles.KeyCap.Render(k[0]), k[1]))
		}
	}

	footer := strings.Join(items, "  ")
	toast := m.Toast.View()
	if toast != "" {
		// Calculate padding to push toast to the right
		padding := m.Width - lipgloss.Width(footer) - lipgloss.Width(toast) - 2
		if padding > 0 {
			footer += strings.Repeat(" ", padding) + toast
		} else {
			footer = toast // Fallback if no space
		}
	}

	return m.Styles.Footer.Width(m.Width).Render(footer)
}

// --- Commands & Helpers ---

func ListenForConsole(c chan string) tea.Cmd {
	return func() tea.Msg {
		if c == nil { return nil }
		data, ok := <-c
		if !ok { return nil }
		return models.ConsoleDataMsg(data)
	}
}

func Tick() tea.Cmd { return tea.Tick(time.Second, func(t time.Time) tea.Msg { return models.TickMsg(t) }) }

func RefreshVMs(c virt.VirtManager) tea.Cmd {
	return func() tea.Msg {
		vms, err := c.ListVMs()
		return models.RefreshMsg{VMs: vms, Err: err}
	}
}

func RefreshNetworks(c virt.VirtManager) tea.Cmd {
	return func() tea.Msg {
		nets, err := c.ListNetworks()
		return models.NetworkListMsg{Networks: nets, Err: err}
	}
}

func RefreshPools(c virt.VirtManager) tea.Cmd {
	return func() tea.Msg {
		pools, err := c.ListStoragePools()
		return models.PoolListMsg{Pools: pools, Err: err}
	}
}

func FetchConfig(c virt.VirtManager, n string) tea.Cmd {
	return func() tea.Msg {
		cfg, err := c.GetVMConfig(n)
		return models.ConfigMsg{Config: cfg, Err: err}
	}
}

func FetchStats(c virt.VirtManager, n string, p virt.VMStats) tea.Cmd {
	return func() tea.Msg {
		s, err := c.GetVMStats(n, p)
		return models.StatsMsg{Stats: s, Err: err}
	}
}

func FetchHostStats(c virt.VirtManager) tea.Cmd {
	return func() tea.Msg {
		s, err := c.GetHostStats()
		return models.HostStatsMsg{Stats: s, Err: err}
	}
}

func RefreshSnapshots(c virt.VirtManager, vmName string) tea.Cmd {
	return func() tea.Msg {
		s, err := c.ListSnapshots(vmName)
		return models.SnapshotListMsg{Snapshots: s, Err: err}
	}
}

func (m *Model) openVMMenu(vm virt.VM) {
	selectedItems := []models.VMItem{}
	for _, item := range m.VMView.List.Items() {
		if v, ok := item.(models.VMItem); ok && v.Selected {
			selectedItems = append(selectedItems, v)
		}
	}

	if len(selectedItems) > 1 {
		m.openBulkVMMenu(selectedItems)
		return
	}

	options := []components.MenuOption{
		{Label: "Start", Action: func() tea.Cmd {
			return func() tea.Msg { err := m.Client.StartVM(vm.Name); return models.RefreshMsg{Err: err} }
		}},
		{Label: "Shutdown", Action: func() tea.Cmd {
			return func() tea.Msg { err := m.Client.ShutdownVM(vm.Name); return models.RefreshMsg{Err: err} }
		}},
		{Label: "Reboot", Action: func() tea.Cmd {
			return func() tea.Msg { err := m.Client.RebootVM(vm.Name); return models.RefreshMsg{Err: err} }
		}},
		{Label: "Suspend", Action: func() tea.Cmd {
			return func() tea.Msg { err := m.Client.SuspendVM(vm.Name); return models.RefreshMsg{Err: err} }
		}},
		{Label: "Resume", Action: func() tea.Cmd {
			return func() tea.Msg { err := m.Client.ResumeVM(vm.Name); return models.RefreshMsg{Err: err} }
		}},
		{Label: "Reset", Action: func() tea.Cmd {
			return func() tea.Msg { err := m.Client.ResetVM(vm.Name); return models.RefreshMsg{Err: err} }
		}},
		{Label: "Force Stop", Action: func() tea.Cmd {
			m.ShowConfirm = true
			m.ConfirmTitle = "FORCE STOP " + vm.Name + "?"
			m.ConfirmAction = func() tea.Cmd {
				return func() tea.Msg { err := m.Client.DestroyVM(vm.Name); return models.RefreshMsg{Err: err} }
			}
			return nil
		}},
		{Label: "Clone VM", Action: func() tea.Cmd {
			m.ShowCloneInput = true
			m.CloneInput.Focus()
			m.CloneInput.SetValue(vm.Name + "-clone")
			return nil
		}},
		{Label: "Delete VM", Action: func() tea.Cmd {
			m.ShowConfirm = true
			m.ConfirmTitle = "DELETE VM " + vm.Name + "?"
			m.ConfirmAction = func() tea.Cmd {
				return func() tea.Msg { err := m.Client.DeleteVM(vm.Name, false); return models.RefreshMsg{Err: err} }
			}
			return nil
		}},
		{Label: "Toggle Autostart", Action: func() tea.Cmd {
			return func() tea.Msg {
				err := m.Client.SetVMAutostart(vm.Name, !vm.Autostart)
				return models.RefreshMsg{Err: err}
			}
		}},
		{Label: "Edit Resources", Action: func() tea.Cmd {
			m.ShowResourceWizard = true
			m.ResourceWizard.ActiveStep = 0
			m.ResourceWizard.Steps[0].Input.SetValue(fmt.Sprintf("%d", m.VMView.Config.VCPUs))
			m.ResourceWizard.Steps[1].Input.SetValue(fmt.Sprintf("%d", m.VMView.Config.Memory))
			m.ResourceWizard.Steps[0].Input.Focus()
			return nil
		}},
		{Label: "Add Disk", Action: func() tea.Cmd {
			m.ShowDiskWizard = true
			m.DiskWizard.ActiveStep = 0
			m.DiskWizard.Steps[0].Input.Focus()
			return nil
		}},
		{Label: "Add Interface", Action: func() tea.Cmd {
			m.ShowInterfaceWizard = true
			m.InterfaceWizard.ActiveStep = 0
			m.InterfaceWizard.Steps[0].Input.Focus()
			return nil
		}},
		{Label: "Remove Disk", Action: func() tea.Cmd {
			m.ShowDiskRemovePicker = true
			items := make([]list.Item, len(m.VMView.Config.Disks))
			for i, d := range m.VMView.Config.Disks { items[i] = models.DiskItem{Disk: d} }
			m.HardwareList.SetItems(items)
			return nil
		}},
		{Label: "Remove Interface", Action: func() tea.Cmd {
			m.ShowInterfaceRemovePicker = true
			items := make([]list.Item, len(m.VMView.Config.Interfaces))
			for i, iface := range m.VMView.Config.Interfaces { items[i] = models.IfaceItem{Iface: iface} }
			m.HardwareList.SetItems(items)
			return nil
		}},
		{Label: "Console", Action: func() tea.Cmd {
			m.ActiveTab = TabConsole
			return m.HandleTabChange()
		}},
		{Label: "Snapshots", Action: func() tea.Cmd {
			m.ActiveTab = TabSnapshots
			return m.HandleTabChange()
		}},
		{Label: "Create Snapshot", Action: func() tea.Cmd {
			m.ShowSnapInput = true
			m.SnapInput.Focus()
			m.SnapInput.SetValue("")
			return nil
		}},
		{Label: "Change Media", Action: func() tea.Cmd {
			m.ShowMediaPicker = true
			return m.FetchAllVolumes()
		}},
	}
	m.Menu.SetOptions("VM: "+vm.Name, options)
}

func (m *Model) openBulkVMMenu(vms []models.VMItem) {
	options := []components.MenuOption{
		{Label: "Bulk Start", Action: func() tea.Cmd {
			return func() tea.Msg {
				for _, v := range vms { _ = m.Client.StartVM(v.VM.Name) }
				return models.RefreshMsg{}
			}
		}},
		{Label: "Bulk Shutdown", Action: func() tea.Cmd {
			return func() tea.Msg {
				for _, v := range vms { _ = m.Client.ShutdownVM(v.VM.Name) }
				return models.RefreshMsg{}
			}
		}},
		{Label: "Bulk Reboot", Action: func() tea.Cmd {
			return func() tea.Msg {
				for _, v := range vms { _ = m.Client.RebootVM(v.VM.Name) }
				return models.RefreshMsg{}
			}
		}},
		{Label: "Bulk Force Stop", Action: func() tea.Cmd {
			m.ShowConfirm = true
			m.ConfirmTitle = fmt.Sprintf("FORCE STOP %d VMs?", len(vms))
			m.ConfirmAction = func() tea.Cmd {
				return func() tea.Msg {
					for _, v := range vms { _ = m.Client.DestroyVM(v.VM.Name) }
					return models.RefreshMsg{}
				}
			}
			return nil
		}},
		{Label: "Deselect All", Action: func() tea.Cmd {
			items := m.VMView.List.Items()
			for i, item := range items {
				if v, ok := item.(models.VMItem); ok {
					v.Selected = false
					items[i] = v
				}
			}
			m.VMView.List.SetItems(items)
			return nil
		}},
	}
	m.Menu.SetOptions(fmt.Sprintf("Bulk Action (%d VMs)", len(vms)), options)
}

func (m *Model) openNetworkMenu(net virt.Network) {
	selectedItems := []models.NetItem{}
	for _, item := range m.NetworkView.List.Items() {
		if v, ok := item.(models.NetItem); ok && v.Selected {
			selectedItems = append(selectedItems, v)
		}
	}

	if len(selectedItems) > 1 {
		m.openBulkNetworkMenu(selectedItems)
		return
	}

	options := []components.MenuOption{
		{Label: "Start", Action: func() tea.Cmd {
			return func() tea.Msg { err := m.Client.StartNetwork(net.Name); return models.NetworkListMsg{Err: err} }
		}},
		{Label: "Stop", Action: func() tea.Cmd {
			return func() tea.Msg { err := m.Client.StopNetwork(net.Name); return models.NetworkListMsg{Err: err} }
		}},
		{Label: "Toggle Autostart", Action: func() tea.Cmd {
			return func() tea.Msg {
				err := m.Client.SetNetworkAutostart(net.Name, !net.Autostart)
				return models.NetworkListMsg{Err: err}
			}
		}},
		{Label: "Delete Network", Action: func() tea.Cmd {
			m.ShowConfirm = true
			m.ConfirmTitle = "DELETE Network " + net.Name + "?"
			m.ConfirmAction = func() tea.Cmd {
				return func() tea.Msg { err := m.Client.DeleteNetwork(net.Name); return models.NetworkListMsg{Err: err} }
			}
			return nil
		}},
		{Label: "Create New Network", Action: func() tea.Cmd {
			m.ShowNetWizard = true
			m.NetWizard.ActiveStep = 0
			m.NetWizard.Steps[0].Input.Focus()
			return nil
		}},
	}
	m.Menu.SetOptions("Network: "+net.Name, options)
}

func (m *Model) openBulkNetworkMenu(nets []models.NetItem) {
	options := []components.MenuOption{
		{Label: "Bulk Start", Action: func() tea.Cmd {
			return func() tea.Msg {
				for _, n := range nets { _ = m.Client.StartNetwork(n.Net.Name) }
				return models.NetworkListMsg{}
			}
		}},
		{Label: "Bulk Stop", Action: func() tea.Cmd {
			return func() tea.Msg {
				for _, n := range nets { _ = m.Client.StopNetwork(n.Net.Name) }
				return models.NetworkListMsg{}
			}
		}},
		{Label: "Deselect All", Action: func() tea.Cmd {
			items := m.NetworkView.List.Items()
			for i, item := range items {
				if v, ok := item.(models.NetItem); ok {
					v.Selected = false
					items[i] = v
				}
			}
			m.NetworkView.List.SetItems(items)
			return nil
		}},
	}
	m.Menu.SetOptions(fmt.Sprintf("Bulk Action (%d Networks)", len(nets)), options)
}

func (m *Model) openPoolMenu(pool virt.StoragePool) {
	selectedItems := []models.PoolItem{}
	for _, item := range m.PoolView.List.Items() {
		if v, ok := item.(models.PoolItem); ok && v.Selected {
			selectedItems = append(selectedItems, v)
		}
	}

	if len(selectedItems) > 1 {
		m.openBulkPoolMenu(selectedItems)
		return
	}

	options := []components.MenuOption{
		{Label: "Start", Action: func() tea.Cmd {
			return func() tea.Msg { err := m.Client.StartStoragePool(pool.Name); return models.PoolListMsg{Err: err} }
		}},
		{Label: "Stop", Action: func() tea.Cmd {
			return func() tea.Msg { err := m.Client.StopStoragePool(pool.Name); return models.PoolListMsg{Err: err} }
		}},
		{Label: "Toggle Autostart", Action: func() tea.Cmd {
			return func() tea.Msg {
				err := m.Client.SetStoragePoolAutostart(pool.Name, !pool.Autostart)
				return models.PoolListMsg{Err: err}
			}
		}},
		{Label: "Delete Pool", Action: func() tea.Cmd {
			m.ShowConfirm = true
			m.ConfirmTitle = "DELETE Pool " + pool.Name + "?"
			m.ConfirmAction = func() tea.Cmd {
				return func() tea.Msg { err := m.Client.DeleteStoragePool(pool.Name); return models.PoolListMsg{Err: err} }
			}
			return nil
		}},
		{Label: "Create New Pool", Action: func() tea.Cmd {
			m.ShowPoolWizard = true
			m.PoolWizard.ActiveStep = 0
			m.PoolWizard.Steps[0].Input.Focus()
			return nil
		}},
		{Label: "Add Volume", Action: func() tea.Cmd {
			m.ShowVolumeWizard = true
			m.VolumeWizard.ActiveStep = 0
			m.VolumeWizard.Steps[0].Input.Focus()
			return nil
		}},
	}
	m.Menu.SetOptions("Pool: "+pool.Name, options)
}

func (m *Model) openBulkPoolMenu(pools []models.PoolItem) {
	options := []components.MenuOption{
		{Label: "Bulk Start", Action: func() tea.Cmd {
			return func() tea.Msg {
				for _, p := range pools { _ = m.Client.StartStoragePool(p.Pool.Name) }
				return models.PoolListMsg{}
			}
		}},
		{Label: "Bulk Stop", Action: func() tea.Cmd {
			return func() tea.Msg {
				for _, p := range pools { _ = m.Client.StopStoragePool(p.Pool.Name) }
				return models.PoolListMsg{}
			}
		}},
		{Label: "Deselect All", Action: func() tea.Cmd {
			items := m.PoolView.List.Items()
			for i, item := range items {
				if v, ok := item.(models.PoolItem); ok {
					v.Selected = false
					items[i] = v
				}
			}
			m.PoolView.List.SetItems(items)
			return nil
		}},
	}
	m.Menu.SetOptions(fmt.Sprintf("Bulk Action (%d Pools)", len(pools)), options)
}

func (m *Model) openSnapshotMenu(vmName, snapName string) {
	options := []components.MenuOption{
		{Label: "Revert", Action: func() tea.Cmd {
			return func() tea.Msg {
				err := m.Client.RevertToSnapshot(vmName, snapName)
				return models.RefreshSnapsMsg{Err: err, VMName: vmName}
			}
		}},
		{Label: "Delete", Action: func() tea.Cmd {
			m.ShowConfirm = true
			m.ConfirmTitle = "DELETE snapshot " + snapName + "?"
			m.ConfirmAction = func() tea.Cmd {
				return func() tea.Msg {
					err := m.Client.DeleteSnapshot(vmName, snapName)
					return models.RefreshSnapsMsg{Err: err, VMName: vmName}
				}
			}
			return nil
		}},
	}
	m.Menu.SetOptions("Snapshot: "+snapName, options)
}
