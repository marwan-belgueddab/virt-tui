package virt

import (
	"fmt"
	"sort"
	"time"

	"github.com/libvirt/libvirt-go"
	"github.com/libvirt/libvirt-go-xml"
)

// Client wraps the libvirt connection
type Client struct {
	Conn *libvirt.Connect
}

// VM represents a virtual machine with its state and basic info
type VM struct {
	ID        int
	Name      string
	UUID      string
	State     libvirt.DomainState
	Autostart bool
}

// VMConfig contains detailed configuration of a VM
type VMConfig struct {
	OS         string
	Arch       string
	Machine    string
	Emulator   string
	VCPUs      uint
	Memory     uint64 // KB
	MaxMemory  uint64 // KB
	Disks      []VMDisk
	Interfaces []VMInterface
	Graphics   []VMGraphics
}

type VMDisk struct {
	Device string
	Type   string
	Source string
	Target string
	Bus    string
	Size   uint64 // Estimated size if possible
}

type VMGraphics struct {
	Type   string
	Port   int
	Listen string
}

type VMInterface struct {
	Name   string
	MAC    string
	IPs    []string
	Source string
	Model  string
}

// VMStats represents resource usage of a VM
type VMStats struct {
	CPUNano     uint64
	CPUTimePrev time.Time
	CPUPercent  float64
	MemoryTotal uint64
	MemoryUsed  uint64
	NetIn       uint64 // Bytes
	NetOut      uint64 // Bytes
	BlockRead   uint64 // Bytes
	BlockWrite  uint64 // Bytes
}

// NewClient establishes a new connection to libvirt
func NewClient(uri string) (*Client, error) {
	if uri == "" {
		uri = "qemu:///system"
	}
	conn, err := libvirt.NewConnect(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to libvirt at %s: %w", uri, err)
	}
	return &Client{Conn: conn}, nil
}

func (c *Client) Close() error {
	if c.Conn != nil {
		res, err := c.Conn.Close()
		if err != nil {
			return err
		}
		if res < 0 {
			return fmt.Errorf("failed to completely close libvirt connection")
		}
	}
	return nil
}

func (c *Client) ListVMs() ([]VM, error) {
	var vms []VM
	flags := libvirt.CONNECT_LIST_DOMAINS_ACTIVE | libvirt.CONNECT_LIST_DOMAINS_INACTIVE
	domains, err := c.Conn.ListAllDomains(flags)
	if err != nil {
		return nil, err
	}
	for _, dom := range domains {
		defer dom.Free()
		name, _ := dom.GetName()
		uuidBytes, _ := dom.GetUUID()
		uuidStr := ""
		if len(uuidBytes) == 16 {
			uuidStr = fmt.Sprintf("%x-%x-%x-%x-%x", uuidBytes[0:4], uuidBytes[4:6], uuidBytes[6:8], uuidBytes[8:10], uuidBytes[10:16])
		}
		id, _ := dom.GetID()
		state, _, _ := dom.GetState()
		autostart, _ := dom.GetAutostart()
		vms = append(vms, VM{ID: int(id), Name: name, UUID: uuidStr, State: state, Autostart: autostart})
	}

	sort.Slice(vms, func(i, j int) bool {
		return vms[i].Name < vms[j].Name
	})

	return vms, nil
}

func (c *Client) GetVMConfig(name string) (VMConfig, error) {
	dom, err := c.Conn.LookupDomainByName(name)
	if err != nil {
		return VMConfig{}, err
	}
	defer dom.Free()

	xmlDesc, err := dom.GetXMLDesc(0)
	if err != nil {
		return VMConfig{}, err
	}

	var domCfg libvirtxml.Domain
	err = domCfg.Unmarshal(xmlDesc)
	if err != nil {
		return VMConfig{}, err
	}

	config := VMConfig{
		VCPUs:  domCfg.VCPU.Value,
		Memory: uint64(domCfg.Memory.Value),
	}
	if domCfg.CurrentMemory != nil {
		config.Memory = uint64(domCfg.CurrentMemory.Value)
		config.MaxMemory = uint64(domCfg.Memory.Value)
	}
	if domCfg.OS != nil && domCfg.OS.Type != nil {
		config.OS = domCfg.OS.Type.Type
		config.Arch = domCfg.OS.Type.Arch
		config.Machine = domCfg.OS.Type.Machine
	}
	if domCfg.Devices != nil {
		config.Emulator = domCfg.Devices.Emulator
	}

	// Try to get IPs from multiple sources
	var ifaces []libvirt.DomainInterface
	// 1. Try guest agent first (most accurate for all interfaces)
	if agentIfaces, err := dom.ListAllInterfaceAddresses(libvirt.DOMAIN_INTERFACE_ADDRESSES_SRC_AGENT); err == nil {
		ifaces = agentIfaces
	} else {
		// 2. Fallback to DHCP leases
		if leaseIfaces, err := dom.ListAllInterfaceAddresses(libvirt.DOMAIN_INTERFACE_ADDRESSES_SRC_LEASE); err == nil {
			ifaces = leaseIfaces
		}
	}

	if domCfg.Devices != nil {
		for _, disk := range domCfg.Devices.Disks {
			vmDisk := VMDisk{
				Device: disk.Device,
				Target: disk.Target.Dev,
				Bus:    disk.Target.Bus,
			}
			if disk.Source != nil {
				if disk.Source.File != nil {
					vmDisk.Source = disk.Source.File.File
					vmDisk.Type = "file"
				} else if disk.Source.Block != nil {
					vmDisk.Source = disk.Source.Block.Dev
					vmDisk.Type = "block"
				}
			}
			config.Disks = append(config.Disks, vmDisk)
		}
		for _, iface := range domCfg.Devices.Interfaces {
			vmIface := VMInterface{MAC: iface.MAC.Address}
			if iface.Model != nil {
				vmIface.Model = iface.Model.Type
			}
			if iface.Source != nil {
				if iface.Source.Network != nil {
					vmIface.Source = iface.Source.Network.Network
				} else if iface.Source.Bridge != nil {
					vmIface.Source = iface.Source.Bridge.Bridge
				}
			}
			if iface.Target != nil {
				vmIface.Name = iface.Target.Dev
			}
			for _, ifAddress := range ifaces {
				if ifAddress.Hwaddr == vmIface.MAC {
					for _, addr := range ifAddress.Addrs {
						vmIface.IPs = append(vmIface.IPs, addr.Addr)
					}
				}
			}
			config.Interfaces = append(config.Interfaces, vmIface)
		}
		for _, graphics := range domCfg.Devices.Graphics {
			g := VMGraphics{}
			if graphics.VNC != nil {
				g.Type = "vnc"
				g.Port = graphics.VNC.Port
				if graphics.VNC.Listen != "" {
					g.Listen = graphics.VNC.Listen
				}
			} else if graphics.Spice != nil {
				g.Type = "spice"
				g.Port = graphics.Spice.Port
			}
			config.Graphics = append(config.Graphics, g)
		}
	}
	return config, nil
}

func (c *Client) GetVMStats(name string, prevStats VMStats) (VMStats, error) {
	dom, err := c.Conn.LookupDomainByName(name)
	if err != nil {
		return VMStats{}, err
	}
	defer dom.Free()

	stats := VMStats{
		CPUTimePrev: prevStats.CPUTimePrev,
	}

	// Memory
	memStats, err := dom.MemoryStats(10, 0)
	if err == nil {
		for _, s := range memStats {
			switch s.Tag {
			case int32(libvirt.DOMAIN_MEMORY_STAT_AVAILABLE):
				stats.MemoryTotal = s.Val
			case int32(libvirt.DOMAIN_MEMORY_STAT_RSS):
				stats.MemoryUsed = s.Val
			}
		}
	}

	// CPU
	info, err := dom.GetInfo()
	if err == nil {
		stats.CPUNano = info.CpuTime
		now := time.Now()
		if !prevStats.CPUTimePrev.IsZero() {
			deltaNano := stats.CPUNano - prevStats.CPUNano
			deltaTime := now.Sub(prevStats.CPUTimePrev).Nanoseconds()
			if deltaTime > 0 {
				stats.CPUPercent = float64(deltaNano) / float64(deltaTime) * 100.0
			}
		}
		stats.CPUTimePrev = now
	}

	// Disk & Network I/O (Optional but better for "Best Practices")
	xmlDesc, err := dom.GetXMLDesc(0)
	if err == nil {
		var domCfg libvirtxml.Domain
		if domCfg.Unmarshal(xmlDesc) == nil && domCfg.Devices != nil {
			// Network
			for _, iface := range domCfg.Devices.Interfaces {
				if iface.Target != nil {
					netStats, err := dom.InterfaceStats(iface.Target.Dev)
					if err == nil {
						stats.NetIn += uint64(netStats.RxBytes)
						stats.NetOut += uint64(netStats.TxBytes)
					}
				}
			}
			// Disk
			for _, disk := range domCfg.Devices.Disks {
				if disk.Target != nil {
					blkStats, err := dom.BlockStats(disk.Target.Dev)
					if err == nil {
						stats.BlockRead += uint64(blkStats.RdBytes)
						stats.BlockWrite += uint64(blkStats.WrBytes)
					}
				}
			}
		}
	}

	return stats, nil
}

func StateString(state libvirt.DomainState) string {
	switch state {
	case libvirt.DOMAIN_RUNNING:
		return "Running"
	case libvirt.DOMAIN_PAUSED:
		return "Paused"
	case libvirt.DOMAIN_SHUTOFF:
		return "Shut Off"
	case libvirt.DOMAIN_SHUTDOWN:
		return "Shutting Down"
	default:
		return "Other"
	}
}

func (c *Client) StartVM(name string) error {
	dom, err := c.Conn.LookupDomainByName(name)
	if err != nil { return err }
	defer dom.Free()
	return dom.Create()
}

func (c *Client) ShutdownVM(name string) error {
	dom, err := c.Conn.LookupDomainByName(name)
	if err != nil { return err }
	defer dom.Free()
	return dom.Shutdown()
}

func (c *Client) RebootVM(name string) error {
	dom, err := c.Conn.LookupDomainByName(name)
	if err != nil { return err }
	defer dom.Free()
	return dom.Reboot(0)
}

func (c *Client) ResetVM(name string) error {
	dom, err := c.Conn.LookupDomainByName(name)
	if err != nil { return err }
	defer dom.Free()
	return dom.Reset(0)
}

func (c *Client) DestroyVM(name string) error {
	dom, err := c.Conn.LookupDomainByName(name)
	if err != nil { return err }
	defer dom.Free()
	return dom.Destroy()
}

func (c *Client) SuspendVM(name string) error {
	dom, err := c.Conn.LookupDomainByName(name)
	if err != nil { return err }
	defer dom.Free()
	return dom.Suspend()
}

func (c *Client) ResumeVM(name string) error {
	dom, err := c.Conn.LookupDomainByName(name)
	if err != nil { return err }
	defer dom.Free()
	return dom.Resume()
}

func (c *Client) CloneVM(name, newName string) error {
	dom, err := c.Conn.LookupDomainByName(name)
	if err != nil { return err }
	defer dom.Free()

	xmlDesc, err := dom.GetXMLDesc(libvirt.DOMAIN_XML_SECURE)
	if err != nil { return err }

	var domCfg libvirtxml.Domain
	if err := domCfg.Unmarshal(xmlDesc); err != nil { return err }

	domCfg.Name = newName
	domCfg.UUID = "" // Let libvirt generate a new one

	newXML, err := domCfg.Marshal()
	if err != nil { return err }

	_, err = c.Conn.DomainDefineXML(newXML)
	return err
}

func (c *Client) DeleteVM(name string, removeStorage bool) error {
	dom, err := c.Conn.LookupDomainByName(name)
	if err != nil { return err }
	defer dom.Free()

	state, _, _ := dom.GetState()
	if state == libvirt.DOMAIN_RUNNING || state == libvirt.DOMAIN_PAUSED {
		_ = dom.Destroy()
	}

	// For simple implementation, we use UNDEFINE_SNAPSHOTS_METADATA and others
	// removing storage is more complex as we need to find all disks and delete them from pools
	return dom.UndefineFlags(libvirt.DOMAIN_UNDEFINE_SNAPSHOTS_METADATA | libvirt.DOMAIN_UNDEFINE_NVRAM)
}

func (c *Client) GetVMAutostart(name string) (bool, error) {
	dom, err := c.Conn.LookupDomainByName(name)
	if err != nil { return false, err }
	defer dom.Free()
	return dom.GetAutostart()
}

func (c *Client) SetVMAutostart(name string, autostart bool) error {
	dom, err := c.Conn.LookupDomainByName(name)
	if err != nil { return err }
	defer dom.Free()
	return dom.SetAutostart(autostart)
}

func (c *Client) SetVCPUs(name string, vcpus uint) error {
	dom, err := c.Conn.LookupDomainByName(name)
	if err != nil { return err }
	defer dom.Free()
	// Set for both current and next boot
	return dom.SetVcpusFlags(vcpus, libvirt.DOMAIN_VCPU_CURRENT | libvirt.DOMAIN_VCPU_CONFIG)
}

func (c *Client) SetMemory(name string, memory uint64) error {
	dom, err := c.Conn.LookupDomainByName(name)
	if err != nil { return err }
	defer dom.Free()
	// Set for both current and next boot
	return dom.SetMemoryFlags(memory, libvirt.DOMAIN_MEM_CURRENT | libvirt.DOMAIN_MEM_CONFIG)
}

func (c *Client) AttachDisk(vmName string, disk VMDisk) error {
	dom, err := c.Conn.LookupDomainByName(vmName)
	if err != nil { return err }
	defer dom.Free()

	libvirtDisk := libvirtxml.DomainDisk{
		Device: disk.Device,
		Driver: &libvirtxml.DomainDiskDriver{Name: "qemu", Type: "qcow2"},
		Source: &libvirtxml.DomainDiskSource{File: &libvirtxml.DomainDiskSourceFile{File: disk.Source}},
		Target: &libvirtxml.DomainDiskTarget{Dev: disk.Target, Bus: disk.Bus},
	}
	xml, err := libvirtDisk.Marshal()
	if err != nil { return err }

	return dom.AttachDeviceFlags(xml, libvirt.DOMAIN_DEVICE_MODIFY_CURRENT | libvirt.DOMAIN_DEVICE_MODIFY_CONFIG)
}

func (c *Client) DetachDisk(vmName string, target string) error {
	dom, err := c.Conn.LookupDomainByName(vmName)
	if err != nil { return err }
	defer dom.Free()

	// Need to find the exact disk XML to detach
	xmlDesc, err := dom.GetXMLDesc(0)
	if err != nil { return err }
	var domCfg libvirtxml.Domain
	if err := domCfg.Unmarshal(xmlDesc); err != nil { return err }

	var targetDisk *libvirtxml.DomainDisk
	for _, d := range domCfg.Devices.Disks {
		if d.Target != nil && d.Target.Dev == target {
			targetDisk = &d
			break
		}
	}
	if targetDisk == nil { return fmt.Errorf("disk %s not found", target) }

	xml, err := targetDisk.Marshal()
	if err != nil { return err }
	return dom.DetachDeviceFlags(xml, libvirt.DOMAIN_DEVICE_MODIFY_CURRENT | libvirt.DOMAIN_DEVICE_MODIFY_CONFIG)
}

func (c *Client) AttachInterface(vmName string, iface VMInterface) error {
	dom, err := c.Conn.LookupDomainByName(vmName)
	if err != nil { return err }
	defer dom.Free()

	libvirtIface := libvirtxml.DomainInterface{
		MAC: &libvirtxml.DomainInterfaceMAC{Address: iface.MAC},
		Source: &libvirtxml.DomainInterfaceSource{Network: &libvirtxml.DomainInterfaceSourceNetwork{Network: iface.Source}},
		Model: &libvirtxml.DomainInterfaceModel{Type: iface.Model},
	}
	xml, err := libvirtIface.Marshal()
	if err != nil { return err }
	return dom.AttachDeviceFlags(xml, libvirt.DOMAIN_DEVICE_MODIFY_CURRENT | libvirt.DOMAIN_DEVICE_MODIFY_CONFIG)
}

func (c *Client) DetachInterface(vmName string, mac string) error {
	dom, err := c.Conn.LookupDomainByName(vmName)
	if err != nil { return err }
	defer dom.Free()

	xmlDesc, err := dom.GetXMLDesc(0)
	if err != nil { return err }
	var domCfg libvirtxml.Domain
	if err := domCfg.Unmarshal(xmlDesc); err != nil { return err }

	var targetIface *libvirtxml.DomainInterface
	for _, i := range domCfg.Devices.Interfaces {
		if i.MAC != nil && i.MAC.Address == mac {
			targetIface = &i
			break
		}
	}
	if targetIface == nil { return fmt.Errorf("interface %s not found", mac) }

	xml, err := targetIface.Marshal()
	if err != nil { return err }
	return dom.DetachDeviceFlags(xml, libvirt.DOMAIN_DEVICE_MODIFY_CURRENT | libvirt.DOMAIN_DEVICE_MODIFY_CONFIG)
}

func (c *Client) CreateVM(name string, memory uint64, vcpus uint, isoPath string, diskPath string) error {
	domCfg := libvirtxml.Domain{
		Type: "kvm",
		Name: name,
		Memory: &libvirtxml.DomainMemory{
			Unit:  "KiB",
			Value: uint(memory),
		},
		CurrentMemory: &libvirtxml.DomainCurrentMemory{
			Unit:  "KiB",
			Value: uint(memory),
		},
		VCPU: &libvirtxml.DomainVCPU{
			Placement: "static",
			Value:     vcpus,
		},
		OS: &libvirtxml.DomainOS{
			Type: &libvirtxml.DomainOSType{
				Arch:    "x86_64",
				Machine: "pc-q35-6.2",
				Type:    "hvm",
			},
			BootDevices: []libvirtxml.DomainBootDevice{
				{Dev: "hd"},
				{Dev: "cdrom"},
			},
		},
		Features: &libvirtxml.DomainFeatureList{
			ACPI: &libvirtxml.DomainFeature{},
			APIC: &libvirtxml.DomainFeatureAPIC{},
			PAE:  &libvirtxml.DomainFeature{},
		},
		CPU: &libvirtxml.DomainCPU{
			Mode:  "host-passthrough",
			Check: "none",
		},
		Clock: &libvirtxml.DomainClock{
			Offset: "utc",
		},
		OnPoweroff: "destroy",
		OnReboot:   "restart",
		OnCrash:    "destroy",
		Devices: &libvirtxml.DomainDeviceList{
			Emulator: "/usr/bin/qemu-system-x86_64",
			Disks: []libvirtxml.DomainDisk{
				{
					Device: "disk",
					Driver: &libvirtxml.DomainDiskDriver{Name: "qemu", Type: "qcow2"},
					Source: &libvirtxml.DomainDiskSource{File: &libvirtxml.DomainDiskSourceFile{File: diskPath}},
					Target: &libvirtxml.DomainDiskTarget{Dev: "vda", Bus: "virtio"},
				},
				{
					Device: "cdrom",
					Driver: &libvirtxml.DomainDiskDriver{Name: "qemu", Type: "raw"},
					Source: &libvirtxml.DomainDiskSource{File: &libvirtxml.DomainDiskSourceFile{File: isoPath}},
					Target: &libvirtxml.DomainDiskTarget{Dev: "sda", Bus: "sata"},
					ReadOnly: &libvirtxml.DomainDiskReadOnly{},
				},
			},
			Interfaces: []libvirtxml.DomainInterface{
				{
					Source: &libvirtxml.DomainInterfaceSource{Network: &libvirtxml.DomainInterfaceSourceNetwork{Network: "default"}},
					Model:  &libvirtxml.DomainInterfaceModel{Type: "virtio"},
				},
			},
			Consoles: []libvirtxml.DomainConsole{
				{
					Target: &libvirtxml.DomainConsoleTarget{Type: "serial", Port: new(uint)},
				},
			},
			Graphics: []libvirtxml.DomainGraphic{
				{
					VNC: &libvirtxml.DomainGraphicVNC{
						Port:     -1,
						AutoPort: "yes",
						Listen:   "127.0.0.1",
						Listeners: []libvirtxml.DomainGraphicListener{
							{Address: &libvirtxml.DomainGraphicListenerAddress{Address: "127.0.0.1"}},
						},
					},
				},
			},
			Videos: []libvirtxml.DomainVideo{
				{
					Model: libvirtxml.DomainVideoModel{
						Type:  "virtio",
						VRam:  16384,
						Heads: 1,
					},
				},
			},
		},
	}

	xml, err := domCfg.Marshal()
	if err != nil {
		return err
	}

	_, err = c.Conn.DomainDefineXML(xml)
	return err
}

// OpenConsole opens a stream to the domain's serial console
func (c *Client) OpenConsole(name string) (*libvirt.Stream, error) {
	dom, err := c.Conn.LookupDomainByName(name)
	if err != nil {
		return nil, err
	}
	defer dom.Free()

	// Check if VM is running
	state, _, err := dom.GetState()
	if err != nil || state != libvirt.DOMAIN_RUNNING {
		return nil, fmt.Errorf("VM must be running to access console")
	}

	stream, err := c.Conn.NewStream(0)
	if err != nil {
		return nil, err
	}

	// Use empty device name to connect to the default console
	// DOMAIN_CONSOLE_FORCE allows us to take over if another console is open
	err = dom.OpenConsole("", stream, libvirt.DOMAIN_CONSOLE_FORCE)
	if err != nil {
		stream.Free()
		return nil, err
	}

	return stream, nil
}

// VMSnapshot represents a VM snapshot
type VMSnapshot struct {
	Name        string
	Description string
	Creation    time.Time
}

// ListSnapshots retrieves all snapshots for a VM
func (c *Client) ListSnapshots(vmName string) ([]VMSnapshot, error) {
	dom, err := c.Conn.LookupDomainByName(vmName)
	if err != nil {
		return nil, err
	}
	defer dom.Free()

	snaps, err := dom.ListAllSnapshots(0)
	if err != nil {
		return nil, err
	}

	var results []VMSnapshot
	for _, snap := range snaps {
		defer snap.Free()
		name, _ := snap.GetName()
		
		// In a real app we'd parse XML for creation time, but for now:
		results = append(results, VMSnapshot{
			Name: name,
		})
	}
	return results, nil
}

// CreateSnapshot creates a new snapshot
func (c *Client) CreateSnapshot(vmName string, name string) error {
	dom, err := c.Conn.LookupDomainByName(vmName)
	if err != nil {
		return err
	}
	defer dom.Free()

	snapCfg := libvirtxml.DomainSnapshot{
		Name: name,
	}
	xml, err := snapCfg.Marshal()
	if err != nil {
		return err
	}
	_, err = dom.CreateSnapshotXML(xml, 0)
	return err
}

// RevertToSnapshot reverts the VM to a specific snapshot
func (c *Client) RevertToSnapshot(vmName string, snapName string) error {
	dom, err := c.Conn.LookupDomainByName(vmName)
	if err != nil {
		return err
	}
	defer dom.Free()

	snap, err := dom.SnapshotLookupByName(snapName, 0)
	if err != nil {
		return err
	}
	defer snap.Free()

	return snap.RevertToSnapshot(0)
}

// DeleteSnapshot deletes a specific snapshot
func (c *Client) DeleteSnapshot(vmName string, snapName string) error {
	dom, err := c.Conn.LookupDomainByName(vmName)
	if err != nil {
		return err
	}
	defer dom.Free()

	snap, err := dom.SnapshotLookupByName(snapName, 0)
	if err != nil {
		return err
	}
	defer snap.Free()

	return snap.Delete(0)
}

// ChangeMedia changes the media in a VM's CD-ROM drive
func (c *Client) ChangeMedia(vmName string, isoPath string) error {
	dom, err := c.Conn.LookupDomainByName(vmName)
	if err != nil {
		return err
	}
	defer dom.Free()

	// Locate the CD-ROM device in XML
	xmlDesc, err := dom.GetXMLDesc(0)
	if err != nil { return err }
	var domCfg libvirtxml.Domain
	if err := domCfg.Unmarshal(xmlDesc); err != nil { return err }

	var targetDev string
	for _, disk := range domCfg.Devices.Disks {
		if disk.Device == "cdrom" {
			targetDev = disk.Target.Dev
			break
		}
	}

	if targetDev == "" {
		return fmt.Errorf("no cdrom device found")
	}

	// Update the disk source
	newDisk := libvirtxml.DomainDisk{
		Device: "cdrom",
		Driver: &libvirtxml.DomainDiskDriver{Name: "qemu", Type: "raw"},
		Target: &libvirtxml.DomainDiskTarget{Dev: targetDev, Bus: "ide"},
	}
	if isoPath != "" {
		newDisk.Source = &libvirtxml.DomainDiskSource{File: &libvirtxml.DomainDiskSourceFile{File: isoPath}}
	}

	newXML, err := newDisk.Marshal()
	if err != nil { return err }
	return dom.UpdateDeviceFlags(newXML, libvirt.DOMAIN_DEVICE_MODIFY_CURRENT)
}

type Network struct {
	Name      string
	Active    bool
	Autostart bool
}

func (c *Client) ListNetworks() ([]Network, error) {
	var nets []Network
	flags := libvirt.CONNECT_LIST_NETWORKS_ACTIVE | libvirt.CONNECT_LIST_NETWORKS_INACTIVE
	networks, err := c.Conn.ListAllNetworks(flags)
	if err != nil {
		return nil, err
	}
	for _, net := range networks {
		defer net.Free()
		name, _ := net.GetName()
		active, _ := net.IsActive()
		autostart, _ := net.GetAutostart()
		nets = append(nets, Network{Name: name, Active: active, Autostart: autostart})
	}

	sort.Slice(nets, func(i, j int) bool {
		return nets[i].Name < nets[j].Name
	})

	return nets, nil
}

func (c *Client) StartNetwork(name string) error {
	net, err := c.Conn.LookupNetworkByName(name)
	if err != nil { return err }
	defer net.Free()
	return net.Create()
}

func (c *Client) StopNetwork(name string) error {
	net, err := c.Conn.LookupNetworkByName(name)
	if err != nil { return err }
	defer net.Free()
	return net.Destroy()
}

func (c *Client) CreateNetwork(name string, bridge string, ipRange string) error {
	netCfg := libvirtxml.Network{
		Name: name,
		Forward: &libvirtxml.NetworkForward{Mode: "nat"},
		Bridge: &libvirtxml.NetworkBridge{Name: bridge, STP: "on", Delay: "0"},
		IPs: []libvirtxml.NetworkIP{
			{
				Address: ipRange,
				Netmask: "255.255.255.0",
				DHCP: &libvirtxml.NetworkDHCP{
					Ranges: []libvirtxml.NetworkDHCPRange{
						{Start: "192.168.122.2", End: "192.168.122.254"},
					},
				},
			},
		},
	}
	xml, err := netCfg.Marshal()
	if err != nil {
		return err
	}
	_, err = c.Conn.NetworkDefineXML(xml)
	return err
}

func (c *Client) DeleteNetwork(name string) error {
	net, err := c.Conn.LookupNetworkByName(name)
	if err != nil { return err }
	defer net.Free()
	active, _ := net.IsActive()
	if active { _ = net.Destroy() }
	return net.Undefine()
}

func (c *Client) SetNetworkAutostart(name string, autostart bool) error {
	net, err := c.Conn.LookupNetworkByName(name)
	if err != nil { return err }
	defer net.Free()
	return net.SetAutostart(autostart)
}

// StoragePool represents a libvirt storage pool
type StoragePool struct {
	Name       string
	Active     bool
	Autostart  bool
	Capacity   uint64 // Bytes
	Allocation uint64 // Bytes
	Available  uint64 // Bytes
	Path       string
}

// StorageVolume represents a volume within a pool
type StorageVolume struct {
	Name       string
	Path       string
	Capacity   uint64
	Allocation uint64
}

// ListStoragePools retrieves all storage pools
func (c *Client) ListStoragePools() ([]StoragePool, error) {
	flags := libvirt.CONNECT_LIST_STORAGE_POOLS_ACTIVE | libvirt.CONNECT_LIST_STORAGE_POOLS_INACTIVE
	pools, err := c.Conn.ListAllStoragePools(flags)
	if err != nil {
		return nil, err
	}

	var results []StoragePool
	for _, pool := range pools {
		defer pool.Free()
		name, _ := pool.GetName()
		active, _ := pool.IsActive()
		autostart, _ := pool.GetAutostart()
		
		info, err := pool.GetInfo()
		capacity, allocation, available := uint64(0), uint64(0), uint64(0)
		if err == nil {
			capacity = info.Capacity
			allocation = info.Allocation
			available = info.Available
		}

		path := ""
		xmlDesc, err := pool.GetXMLDesc(0)
		if err == nil {
			var poolCfg libvirtxml.StoragePool
			if poolCfg.Unmarshal(xmlDesc) == nil && poolCfg.Target != nil {
				path = poolCfg.Target.Path
			}
		}

		results = append(results, StoragePool{
			Name:       name,
			Active:     active,
			Autostart:  autostart,
			Capacity:   capacity,
			Allocation: allocation,
			Available:  available,
			Path:       path,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	return results, nil
}

func (c *Client) StartStoragePool(name string) error {
	pool, err := c.Conn.LookupStoragePoolByName(name)
	if err != nil { return err }
	defer pool.Free()
	return pool.Create(0)
}

func (c *Client) StopStoragePool(name string) error {
	pool, err := c.Conn.LookupStoragePoolByName(name)
	if err != nil { return err }
	defer pool.Free()
	return pool.Destroy()
}

func (c *Client) SetStoragePoolAutostart(name string, autostart bool) error {
	pool, err := c.Conn.LookupStoragePoolByName(name)
	if err != nil { return err }
	defer pool.Free()
	return pool.SetAutostart(autostart)
}

func (c *Client) CreateStoragePool(name string, path string) error {
	poolCfg := libvirtxml.StoragePool{
		Type: "dir",
		Name: name,
		Target: &libvirtxml.StoragePoolTarget{
			Path: path,
		},
	}
	xml, err := poolCfg.Marshal()
	if err != nil {
		return err
	}
	_, err = c.Conn.StoragePoolDefineXML(xml, 0)
	return err
}

func (c *Client) DeleteStoragePool(name string) error {
	pool, err := c.Conn.LookupStoragePoolByName(name)
	if err != nil { return err }
	defer pool.Free()
	active, _ := pool.IsActive()
	if active { _ = pool.Destroy() }
	return pool.Undefine()
}

// ListStorageVolumes retrieves all volumes in a pool
func (c *Client) ListStorageVolumes(poolName string) ([]StorageVolume, error) {
	pool, err := c.Conn.LookupStoragePoolByName(poolName)
	if err != nil {
		return nil, err
	}
	defer pool.Free()

	vols, err := pool.ListAllStorageVolumes(0)
	if err != nil {
		return nil, err
	}

	var results []StorageVolume
	for _, vol := range vols {
		defer vol.Free()
		name, _ := vol.GetName()
		path, _ := vol.GetPath()
		info, _ := vol.GetInfo()
		
		results = append(results, StorageVolume{
			Name:       name,
			Path:       path,
			Capacity:   info.Capacity,
			Allocation: info.Allocation,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	return results, nil
}

func (c *Client) CreateStorageVolume(poolName string, name string, capacity uint64) error {
	pool, err := c.Conn.LookupStoragePoolByName(poolName)
	if err != nil { return err }
	defer pool.Free()

	volCfg := libvirtxml.StorageVolume{
		Name:     name,
		Capacity: &libvirtxml.StorageVolumeSize{Unit: "bytes", Value: capacity},
		Target: &libvirtxml.StorageVolumeTarget{
			Format: &libvirtxml.StorageVolumeTargetFormat{Type: "qcow2"},
		},
	}
	xml, err := volCfg.Marshal()
	if err != nil {
		return err
	}
	_, err = pool.StorageVolCreateXML(xml, 0)
	return err
}

func (c *Client) DeleteStorageVolume(poolName string, name string) error {
	pool, err := c.Conn.LookupStoragePoolByName(poolName)
	if err != nil { return err }
	defer pool.Free()

	vol, err := pool.LookupStorageVolByName(name)
	if err != nil { return err }
	defer vol.Free()

	return vol.Delete(0)
}

// HostStats represents physical host resource usage
type HostStats struct {
	MemoryTotal     uint64 // KB
	MemoryFree      uint64 // KB
	CPUNodes        uint   // Number of CPU nodes
	CPUTopology     string // e.g. "1 socket, 4 cores, 2 threads"
	Model           string
	LibvirtVersion  string
	Hostname        string
}

// GetHostStats fetches information about the physical host
func (c *Client) GetHostStats() (HostStats, error) {
	info, err := c.Conn.GetNodeInfo()
	if err != nil {
		return HostStats{}, err
	}

	memFree, err := c.Conn.GetFreeMemory()
	if err != nil {
		memFree = 0
	}

	ver, _ := c.Conn.GetLibVersion()
	hostname, _ := c.Conn.GetHostname()

	return HostStats{
		MemoryTotal:    info.Memory,
		MemoryFree:     memFree / 1024,
		CPUNodes:       uint(info.Nodes),
		CPUTopology:    fmt.Sprintf("%d socket(s), %d core(s), %d thread(s)", info.Sockets, info.Cores, info.Threads),
		Model:          info.Model,
		LibvirtVersion: fmt.Sprintf("%d.%d.%d", ver/1000000, (ver/1000)%1000, ver%1000),
		Hostname:       hostname,
	}, nil
}
