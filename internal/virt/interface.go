package virt

import (
	"github.com/libvirt/libvirt-go"
)

// VirtManager defines the interface for interacting with libvirt.
// This allows for easier mocking and testing without a real libvirt connection.
type VirtManager interface {
	Close() error
	ListVMs() ([]VM, error)
	GetVMConfig(name string) (VMConfig, error)
	GetVMStats(name string, prevStats VMStats) (VMStats, error)
	
	StartVM(name string) error
	ShutdownVM(name string) error
	RebootVM(name string) error
	ResetVM(name string) error
	DestroyVM(name string) error
	SuspendVM(name string) error
	ResumeVM(name string) error

	CreateVM(name string, memory uint64, vcpus uint, isoPath string, diskPath string) error
	CloneVM(name, newName string) error
	DeleteVM(name string, removeStorage bool) error

	GetVMAutostart(name string) (bool, error)
	SetVMAutostart(name string, autostart bool) error

	SetVCPUs(name string, vcpus uint) error
	SetMemory(name string, memory uint64) error

	AttachDisk(vmName string, disk VMDisk) error
	DetachDisk(vmName string, target string) error
	AttachInterface(vmName string, iface VMInterface) error
	DetachInterface(vmName string, mac string) error

	OpenConsole(name string) (*libvirt.Stream, error)

	ListSnapshots(vmName string) ([]VMSnapshot, error)
	CreateSnapshot(vmName string, name string) error
	RevertToSnapshot(vmName string, snapName string) error
	DeleteSnapshot(vmName string, snapName string) error

	ChangeMedia(vmName string, isoPath string) error

	ListNetworks() ([]Network, error)
	StartNetwork(name string) error
	StopNetwork(name string) error
	CreateNetwork(name string, bridge string, ipRange string) error
	DeleteNetwork(name string) error
	SetNetworkAutostart(name string, autostart bool) error

	ListStoragePools() ([]StoragePool, error)
	StartStoragePool(name string) error
	StopStoragePool(name string) error
	CreateStoragePool(name string, path string) error
	DeleteStoragePool(name string) error
	SetStoragePoolAutostart(name string, autostart bool) error

	ListStorageVolumes(poolName string) ([]StorageVolume, error)
	CreateStorageVolume(poolName string, name string, capacity uint64) error
	DeleteStorageVolume(poolName string, name string) error
	
	GetHostStats() (HostStats, error)
}
