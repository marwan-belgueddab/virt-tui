package models

import (
	"virt-tui/internal/virt"
)

type VMItem struct {
	VM       virt.VM
	Selected bool
}
func (i VMItem) Title() string       { return i.VM.Name }
func (i VMItem) Description() string { return virt.StateString(i.VM.State) }
func (i VMItem) FilterValue() string { return i.VM.Name }

type NetItem struct {
	Net      virt.Network
	Selected bool
}
func (i NetItem) Title() string       { return i.Net.Name }
func (i NetItem) Description() string { if i.Net.Active { return "Active" }; return "Inactive" }
func (i NetItem) FilterValue() string { return i.Net.Name }

type PoolItem struct {
	Pool     virt.StoragePool
	Selected bool
}
func (i PoolItem) Title() string       { return i.Pool.Name }
func (i PoolItem) Description() string { if i.Pool.Active { return "Active" }; return "Inactive" }
func (i PoolItem) FilterValue() string { return i.Pool.Name }

type VolItem struct {
	Vol      virt.StorageVolume
	Selected bool
}
func (i VolItem) Title() string       { return i.Vol.Name }
func (i VolItem) Description() string { return i.Vol.Path }
func (i VolItem) FilterValue() string { return i.Vol.Name }

type SnapItem struct {
	Snap     virt.VMSnapshot
	Selected bool
}
func (i SnapItem) Title() string       { return i.Snap.Name }
func (i SnapItem) Description() string { return "Snapshot" }
func (i SnapItem) FilterValue() string { return i.Snap.Name }

type DiskItem struct {
	Disk     virt.VMDisk
	Selected bool
}
func (i DiskItem) Title() string       { return i.Disk.Target }
func (i DiskItem) Description() string { return i.Disk.Source }
func (i DiskItem) FilterValue() string { return i.Disk.Target }

type IfaceItem struct {
	Iface    virt.VMInterface
	Selected bool
}
func (i IfaceItem) Title() string       { return i.Iface.MAC }
func (i IfaceItem) Description() string { return i.Iface.Source }
func (i IfaceItem) FilterValue() string { return i.Iface.MAC }
