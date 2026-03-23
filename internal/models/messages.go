package models

import (
	"time"

	"virt-tui/internal/virt"
)

type TickMsg time.Time
type RefreshMsg struct {
	VMs []virt.VM
	Err error
}
type ConfigMsg struct {
	Config virt.VMConfig
	Err    error
}
type StatsMsg struct {
	Stats virt.VMStats
	Err   error
}
type HostStatsMsg struct {
	Stats virt.HostStats
	Err   error
}
type RefreshSnapsMsg struct {
	VMName string
	Err    error
}
type SnapshotListMsg struct {
	Snapshots []virt.VMSnapshot
	Err       error
}
type VolListMsg struct {
	Vols []virt.StorageVolume
	Err  error
}
type NetworkListMsg struct {
	Networks []virt.Network
	Err      error
}
type PoolListMsg struct {
	Pools []virt.StoragePool
	Err   error
}
type ConsoleDataMsg string
