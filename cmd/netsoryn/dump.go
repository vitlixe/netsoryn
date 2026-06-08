package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/vitlixe/netsoryn/internal/collectors"
)

// dumpOutput is the top-level JSON document produced by `netsoryn dump`.
// Sections are added as more collectors are wired in; omitempty keeps the
// document compact when a section is unavailable.
type dumpOutput struct {
	System *collectors.SystemData `json:"system,omitempty"`
}

func dumpCmd() *cobra.Command {
	var pretty bool
	var format string

	cmd := &cobra.Command{
		Use:   "dump",
		Short: "Print a one-shot system snapshot (json, md, or text)",
		Long: `Collect a single snapshot of system metrics and print it to stdout, then
exit. Unlike the interactive UI this needs no terminal, so it works over SSH,
in scripts, and in CI.

Formats (--format): json (default, machine-readable), md (Markdown, for
tickets/wikis/chat), text (plain table for the terminal).

  netsoryn dump > snapshot.json
  netsoryn dump --format md > snapshot.md
  netsoryn dump --format text
  ssh host 'netsoryn dump' | jq .system`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			sys, err := systemSnapshot(ctx)
			if err != nil {
				return err
			}

			switch format {
			case "json":
				enc := json.NewEncoder(os.Stdout)
				enc.SetEscapeHTML(false)
				if pretty {
					enc.SetIndent("", "  ")
				}
				return enc.Encode(dumpOutput{System: &sys})
			case "md", "markdown":
				_, err := fmt.Fprint(os.Stdout, renderMarkdown(sys))
				return err
			case "text", "txt":
				_, err := fmt.Fprint(os.Stdout, renderText(sys))
				return err
			default:
				return fmt.Errorf("unknown format %q (want json, md, or text)", format)
			}
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "json", "output format: json, md, or text")
	cmd.Flags().BoolVar(&pretty, "pretty", false, "indent the output (json format only)")
	return cmd
}

// systemSnapshot collects a single SystemData sample. It overrides CPUTotal with
// the mean of the per-core percentages: the collector's aggregate total uses a
// non-blocking sample that is only meaningful across repeated UI refreshes,
// whereas the per-core values come from a real 200ms sample and so give an
// accurate instantaneous reading for a one-shot dump.
func systemSnapshot(ctx context.Context) (collectors.SystemData, error) {
	raw, err := collectors.NewSystemCollector().Collect(ctx)
	if err != nil {
		return collectors.SystemData{}, fmt.Errorf("collecting system data: %w", err)
	}
	sys, ok := raw.(collectors.SystemData)
	if !ok {
		return collectors.SystemData{}, fmt.Errorf("unexpected system collector result type %T", raw)
	}
	if len(sys.CPUPercents) > 0 {
		sys.CPUTotal = mean(sys.CPUPercents)
	}
	return sys, nil
}

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs))
}
