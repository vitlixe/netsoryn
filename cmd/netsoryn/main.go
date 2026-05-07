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

			setupLogging(cfg)

			m := ui.New(cfg, version)
			p := tea.NewProgram(
				ui.NewSplash(m),
				tea.WithAltScreen(),
				tea.WithMouseCellMotion(),
			)

			if _, err := p.Run(); err != nil {
				return fmt.Errorf("running UI: %w", err)
			}
			return nil
		},
	}

	root.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.config/netsoryn/config.yaml)")

	root.AddCommand(versionCmd())
	root.AddCommand(checkCmd())

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

// checkCmd is a non-interactive one-shot diagnostic suitable for CI/scripts.
func checkCmd() *cobra.Command {
	var (
		urlFlag    string
		domainFlag string
	)

	cmd := &cobra.Command{
		Use:    "check",
		Short:  "Run a single diagnostic check and print JSON output",
		Long:   `Run a one-shot check (HTTP or DNS) and print the result as JSON to stdout.`,
		Hidden: true, // not yet implemented — will be available in a future release
		RunE: func(cmd *cobra.Command, args []string) error {
			if urlFlag == "" && domainFlag == "" {
				return fmt.Errorf("provide --url or --domain")
			}
			cfg, _ := config.Load("")
			_ = cfg
			// TODO: implement JSON output for headless check mode
			fmt.Println("check mode: coming soon")
			return nil
		},
	}

	cmd.Flags().StringVar(&urlFlag, "url", "", "URL to check (HTTP/TLS)")
	cmd.Flags().StringVar(&domainFlag, "domain", "", "Domain to resolve (DNS)")
	return cmd
}

func setupLogging(cfg *config.Config) {
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
			return
		}
	}
	// Default: write to stderr (hidden behind alt-screen during TUI)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}
