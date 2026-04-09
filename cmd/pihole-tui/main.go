package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rodrigodealer/pihole-tui/internal/api"
	"github.com/rodrigodealer/pihole-tui/internal/config"
	"github.com/rodrigodealer/pihole-tui/internal/ui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if cfg.Host == "" {
		cfg.Host = "http://192.168.0.70"
	}

	client := api.NewClient(cfg.Host, cfg.Password)

	if cfg.Password != "" {
		if err := client.Authenticate(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: authentication failed: %v\n", err)
		}
	}

	m := ui.NewModel(client)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
