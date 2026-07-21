package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/vitlixe/netsoryn/internal/config"
	"github.com/vitlixe/netsoryn/internal/sshclient"
)

var runSSHCommand = func(args []string) error {
	c := exec.Command("ssh", args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func sshCmd(cfgFile *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh <name> [-- remote command...]",
		Short: "Connect to a configured SSH host",
		Long: `Connect to a host from ssh_hosts in the Netsoryn config.

Netsoryn does not implement SSH itself. It builds a small, explicit argv and
executes the system ssh binary, so keys, agents, passphrases, and known_hosts
stay under normal OpenSSH control.

Examples:
  netsoryn ssh list
  netsoryn ssh prod
  netsoryn ssh prod -- uname -a
  netsoryn ssh dump prod -f text`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*cfgFile)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			host, ok := sshclient.FindHost(cfg, args[0])
			if !ok {
				return fmt.Errorf("unknown ssh host %q", args[0])
			}
			return runSSHCommand(sshclient.BuildArgs(host, args[1:]))
		},
	}

	cmd.AddCommand(sshListCmd(cfgFile))
	cmd.AddCommand(sshDumpCmd(cfgFile))

	return cmd
}

func sshListCmd(cfgFile *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured SSH hosts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*cfgFile)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			return renderSSHHostList(cmd.OutOrStdout(), cfg.SSHHosts)
		},
	}
}

func sshDumpCmd(cfgFile *string) *cobra.Command {
	var format string
	var sections string
	var pretty bool

	cmd := &cobra.Command{
		Use:   "dump <name>",
		Short: "Run netsoryn dump on a configured SSH host",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*cfgFile)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			host, ok := sshclient.FindHost(cfg, args[0])
			if !ok {
				return fmt.Errorf("unknown ssh host %q", args[0])
			}
			return runSSHCommand(sshclient.BuildArgs(host, sshclient.RemoteDumpCommand(format, sections, pretty)))
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "json", "output format: json, md, or text")
	cmd.Flags().StringVar(&sections, "sections", "", "comma-separated dump sections (default: system)")
	cmd.Flags().BoolVar(&pretty, "pretty", false, "indent the output (json format only)")
	return cmd
}

func renderSSHHostList(w io.Writer, hosts []config.SSHHost) error {
	if len(hosts) == 0 {
		_, err := fmt.Fprintln(w, "No SSH hosts configured.")
		return err
	}

	ordered := append([]config.SSHHost(nil), hosts...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Name < ordered[j].Name
	})

	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "NAME\tTARGET\tKEY\tOPTIONS"); err != nil {
		return err
	}
	for _, host := range ordered {
		target := sshclient.Target(host)
		if host.Port > 0 && host.Port != 22 {
			target += ":" + strconv.Itoa(host.Port)
		}
		key := host.Key
		if key == "" {
			key = "-"
		}
		opts := "-"
		if len(host.Options) > 0 {
			opts = strings.Join(host.Options, " ")
		}
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", host.Name, target, key, opts); err != nil {
			return err
		}
	}
	return tw.Flush()
}
