package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/vitlixe/netsoryn/internal/config"
	"github.com/vitlixe/netsoryn/internal/ui"
)

var version = "dev"

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	var cfgFile string

	root := &cobra.Command{
		Use:   "netsoryn",
		Short: "A terminal dashboard for system, network, and service diagnostics.",
		Long: `Netsoryn is a fast terminal UI for system, network, and service diagnostics.
Open it on any host to instantly see system metrics, open ports,
processes, services, Docker containers, DNS, HTTP endpoints and more.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			if f := setupLogging(cfg); f != nil {
				defer f.Close()
			}

			m := ui.New(cfg, version)
			p := tea.NewProgram(
				ui.NewSplash(m),
				tea.WithAltScreen(),
			)

			if _, err := p.Run(); err != nil {
				return fmt.Errorf("running UI: %w", err)
			}
			return nil
		},
	}

	root.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.config/netsoryn/config.yaml)")

	root.AddCommand(versionCmd())
	root.AddCommand(dumpCmd())

	return root
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("netsoryn %s\n", version)
		},
	}
}

func setupLogging(cfg *config.Config) *os.File {
	level := zerolog.Disabled
	switch cfg.LogLevel {
	case "debug":
		level = zerolog.DebugLevel
	case "info":
		level = zerolog.InfoLevel
	case "warn":
		level = zerolog.WarnLevel
	case "error":
		level = zerolog.ErrorLevel
	}
	zerolog.SetGlobalLevel(level)

	if cfg.LogFile != "" {
		f, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err == nil {
			log.Logger = log.Output(f)
			return f
		}
	}
	// Default: write to stderr (hidden behind alt-screen during TUI)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	return nil
}
