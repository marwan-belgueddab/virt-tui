package main

import (
	"flag"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"virt-tui/internal/tui"
	"virt-tui/internal/virt"
)

func main() {
	uri := flag.String("uri", "qemu:///system", "Libvirt URI")
	flag.Parse()

	client, err := virt.NewClient(*uri)
	if err != nil {
		log.Fatalf("failed to connect to libvirt: %v", err)
	}
	defer client.Close()

	m := tui.NewModel(client)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("failed to run TUI: %v", err)
	}
}
