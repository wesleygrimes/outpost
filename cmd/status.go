package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/wesgrimes/outpost/internal/config"
	"github.com/wesgrimes/outpost/internal/grpcclient"
	"github.com/wesgrimes/outpost/internal/store"
)

// Status lists runs or shows detail for a specific run.
func Status(args []string) error {
	follow := false
	var id string

	for _, arg := range args {
		if arg == "--follow" || arg == "-f" {
			follow = true
		} else {
			id = arg
		}
	}

	client, err := grpcclient.Load()
	if err != nil {
		return err
	}
	defer logClose(client)

	ctx := context.Background()

	if follow && id != "" {
		return followLogs(ctx, client, id)
	}

	if id != "" {
		return showRunDetail(ctx, client, id)
	}

	runs, err := client.ListRuns(ctx)
	if err != nil {
		return err
	}
	printDashboard(runs)
	return nil
}

func followLogs(ctx context.Context, client *grpcclient.Client, id string) error {
	fmt.Fprintf(os.Stderr, "following run %s...\n", id)

	stream, err := client.TailLogs(ctx, id, true)
	if err != nil {
		return err
	}

	for {
		entry, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			fmt.Fprintln(os.Stderr, "run completed")
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Println(entry.GetLine())
	}
}

func printDashboard(runs []*store.Run) {
	// Count by status.
	var active, complete, failed, dropped int
	var running, recent []*store.Run

	for _, r := range runs {
		switch r.Status {
		case store.StatusPending, store.StatusRunning:
			active++
			running = append(running, r)
		case store.StatusComplete:
			complete++
			recent = append(recent, r)
		case store.StatusFailed:
			failed++
			recent = append(recent, r)
		case store.StatusDropped:
			dropped++
		}
	}

	cfg, cfgErr := config.LoadClient()
	serverName := ""
	if cfgErr == nil {
		serverName = cfg.Server
	}

	printBoxTop("Outpost", serverName)

	// Summary row.
	summary := fmt.Sprintf("  Active  %d    Complete  %d    Failed  %d    Dropped  %d",
		active, complete, failed, dropped)
	printBoxRow(summary)
	printBoxRow("")

	// Running table.
	if len(running) > 0 {
		printBoxDivider("Running")
		printBoxRow(fmt.Sprintf("  %-34s %-14s %-9s %s", "ID", "MODE", "AGE", "BRANCH"))
		for _, r := range running {
			age := formatAge(r.CreatedAt)
			printBoxRow(fmt.Sprintf("  %-34s %-14s %-9s %s",
				truncate(r.ID, 34), string(r.Mode), age, r.Branch))
		}
		printBoxRow("")
	}

	// Recent table.
	if len(recent) > 0 {
		printBoxDivider("Recent")
		printBoxRow(fmt.Sprintf("  %-34s %-14s %-9s %s", "ID", "STATUS", "AGE", "PATCH"))
		for _, r := range recent {
			age := formatAge(r.CreatedAt)
			if r.FinishedAt != nil {
				age = formatAge(*r.FinishedAt)
			}
			patch := "--"
			if r.PatchReady {
				patch = "ready"
			} else if r.Status == store.StatusComplete && !r.PatchReady {
				patch = "picked up"
			}
			printBoxRow(fmt.Sprintf("  %-34s %-14s %-9s %s",
				truncate(r.ID, 34), string(r.Status), age, patch))
		}
	}

	printBoxBottom()
}

func formatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "\u2026"
}

func showRunDetail(ctx context.Context, client *grpcclient.Client, id string) error {
	r, err := client.GetRun(ctx, id)
	if err != nil {
		return err
	}
	printRunKV(r)
	return nil
}

func printRunKV(r *store.Run) {
	fmt.Printf("id=%s\n", r.ID)
	fmt.Printf("name=%s\n", r.Name)
	fmt.Printf("status=%s\n", r.Status)
	fmt.Printf("mode=%s\n", r.Mode)
	fmt.Printf("branch=%s\n", r.Branch)
	fmt.Printf("base_sha=%s\n", r.BaseSHA)
	fmt.Printf("final_sha=%s\n", r.FinalSHA)
	fmt.Printf("patch_ready=%t\n", r.PatchReady)
	fmt.Printf("created_at=%s\n", r.CreatedAt.Format("2006-01-02 15:04:05"))
	if r.FinishedAt != nil {
		fmt.Printf("finished_at=%s\n", r.FinishedAt.Format("2006-01-02 15:04:05"))
	}
	if r.Attach != "" {
		fmt.Printf("attach=%s\n", r.Attach)
	}

	if r.LogTail != "" {
		fmt.Println()
		fmt.Println("--- log tail ---")
		for line := range strings.SplitSeq(r.LogTail, "\n") {
			fmt.Println(line)
		}
	}
}
