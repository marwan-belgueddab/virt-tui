package tui

import (
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
	"virt-tui/internal/virt"
)

func TestRefreshVMs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := virt.NewMockVirtManager(ctrl)
	
	expectedVMs := []virt.VM{
		{Name: "test-vm-1", State: 1},
		{Name: "test-vm-2", State: 5},
	}

	mockClient.EXPECT().ListVMs().Return(expectedVMs, nil)

	cmd := RefreshVMs(mockClient)
	msg := cmd()

	refreshResult, ok := msg.(RefreshMsg)
	if !ok {
		t.Fatalf("expected RefreshMsg, got %T", msg)
	}

	if refreshResult.Err != nil {
		t.Errorf("expected no error, got %v", refreshResult.Err)
	}

	if len(refreshResult.VMs) != len(expectedVMs) {
		t.Errorf("expected %d VMs, got %d", len(expectedVMs), len(refreshResult.VMs))
	}
}

func TestRefreshVMs_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := virt.NewMockVirtManager(ctrl)
	
	expectedErr := errors.New("libvirt connection failed")
	mockClient.EXPECT().ListVMs().Return(nil, expectedErr)

	cmd := RefreshVMs(mockClient)
	msg := cmd()

	refreshResult, ok := msg.(RefreshMsg)
	if !ok {
		t.Fatalf("expected RefreshMsg, got %T", msg)
	}

	if refreshResult.Err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, refreshResult.Err)
	}
}
