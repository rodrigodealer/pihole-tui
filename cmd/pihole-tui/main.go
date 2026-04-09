package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rodrigodealer/pihole-tui/internal/api"
	"github.com/rodrigodealer/pihole-tui/internal/config"
	"github.com/rodrigodealer/pihole-tui/internal/ui"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version")
	flag.Parse()

	if *showVersion {
		fmt.Println("pihole-tui", version)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if cfg.Host == "" {
		cfg.Host = "http://192.168.0.70"
	}

	var provider api.Provider

	switch cfg.Type {
	case "adguard":
		provider = api.NewAdGuardClient(cfg.Host, cfg.Username, cfg.Password)
	default:
		client := api.NewClient(cfg.Host, cfg.Password)
		if cfg.Password != "" {
			if err := client.Authenticate(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: authentication failed: %v\n", err)
			}
		}
		provider = client
	}

	m := ui.NewModel(provider)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
