package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/wesgrimes/outpost/internal/config"
	"github.com/wesgrimes/outpost/internal/grpcclient"
)

// Runs lists runs via the local server using server config credentials.
func Runs(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	target := fmt.Sprintf("localhost:%d", cfg.Port)

	dialOpt, err := grpcclient.TLSDialOption(cfg.TLSCA)
	if err != nil {
		return err
	}

	client, err := grpcclient.New(target, cfg.Token, dialOpt)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer logClose(client)

	ctx := context.Background()

	if len(args) > 0 {
		return showRunDetail(ctx, client, args[0])
	}

	runs, err := client.ListRuns(ctx)
	if err != nil {
		return diskFallbackList()
	}
	return printRunsTable(runs)
}

func diskFallbackList() error {
	runsDir := config.RunsDir()
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return errors.New("no runs found (server unreachable, no runs on disk)")
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tSTATUS\tMODE\tCREATED")
	for _, e := range entries {
		if e.IsDir() {
			info, _ := e.Info()
			created := ""
			if info != nil {
				created = info.ModTime().Format("2006-01-02 15:04:05")
			}
			_, _ = fmt.Fprintf(w, "%s\t(disk)\t-\t%s\n", e.Name(), created)
		}
	}
	return w.Flush()
}
