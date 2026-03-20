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
	jsonOut, args := hasFlag(args, "--json")
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
		return showRunDetail(ctx, client, id, jsonOut)
	}

	runs, err := client.ListRuns(ctx)
	if err != nil {
		return err
	}

	if jsonOut {
		return printDashboardJSON(runs)
	}

	printHeader()
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

	fmt.Printf("\n  Active: %d   Complete: %d   Failed: %d   Dropped: %d\n", active, complete, failed, dropped)

	if len(running) == 0 && len(recent) == 0 {
		return
	}

	fmt.Println()
	tw := newTable()
	_, _ = fmt.Fprintf(tw, "  ID\tSTATUS\tMODE\tBRANCH\tAGE\tPATCH\n")
	for _, r := range running {
		branch := r.Branch
		if branch == "" {
			branch = "-"
		}
		_, _ = fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%s\t\n",
			r.ID, r.Status, r.Mode, branch, formatAge(r.CreatedAt))
	}
	for _, r := range recent {
		age := formatAge(r.CreatedAt)
		if r.FinishedAt != nil {
			age = formatAge(*r.FinishedAt)
		}
		branch := r.Branch
		if branch == "" {
			branch = "-"
		}
		patch := "no"
		if r.PatchReady {
			patch = "yes"
		}
		_, _ = fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%s\t%s\n",
			r.ID, r.Status, r.Mode, branch, age, patch)
	}
	_ = tw.Flush()
}

type dashboardJSON struct {
	Server   string           `json:"server"`
	Active   int              `json:"active"`
	Complete int              `json:"complete"`
	Failed   int              `json:"failed"`
	Dropped  int              `json:"dropped"`
	Runs     []runSummaryJSON `json:"runs"`
}

type runSummaryJSON struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	Mode       string `json:"mode"`
	Branch     string `json:"branch"`
	Age        string `json:"age"`
	PatchReady bool   `json:"patch_ready,omitempty"`
}

func printDashboardJSON(runs []*store.Run) error {
	var active, complete, failed, dropped int
	var summaries []runSummaryJSON

	for _, r := range runs {
		switch r.Status {
		case store.StatusPending, store.StatusRunning:
			active++
			summaries = append(summaries, runSummaryJSON{
				ID:     r.ID,
				Status: string(r.Status),
				Mode:   string(r.Mode),
				Branch: r.Branch,
				Age:    formatAge(r.CreatedAt),
			})
		case store.StatusComplete:
			complete++
			age := formatAge(r.CreatedAt)
			if r.FinishedAt != nil {
				age = formatAge(*r.FinishedAt)
			}
			summaries = append(summaries, runSummaryJSON{
				ID:         r.ID,
				Status:     string(r.Status),
				Mode:       string(r.Mode),
				Branch:     r.Branch,
				Age:        age,
				PatchReady: r.PatchReady,
			})
		case store.StatusFailed:
			failed++
			age := formatAge(r.CreatedAt)
			if r.FinishedAt != nil {
				age = formatAge(*r.FinishedAt)
			}
			summaries = append(summaries, runSummaryJSON{
				ID:         r.ID,
				Status:     string(r.Status),
				Mode:       string(r.Mode),
				Branch:     r.Branch,
				Age:        age,
				PatchReady: r.PatchReady,
			})
		case store.StatusDropped:
			dropped++
		}
	}

	cfg, cfgErr := config.LoadClient()
	serverName := ""
	if cfgErr == nil {
		serverName = cfg.Server
	}

	if summaries == nil {
		summaries = []runSummaryJSON{}
	}

	return printJSON(dashboardJSON{
		Server:   serverName,
		Active:   active,
		Complete: complete,
		Failed:   failed,
		Dropped:  dropped,
		Runs:     summaries,
	})
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

func showRunDetail(ctx context.Context, client *grpcclient.Client, id string, jsonOut bool) error {
	r, err := client.GetRun(ctx, id)
	if err != nil {
		return err
	}
	if jsonOut {
		return printJSON(r)
	}

	printHeader()
	printRunDetail(r)
	return nil
}

func printRunDetail(r *store.Run) {
	fmt.Println()
	printField("Run:", r.ID)
	printField("Name:", r.Name)
	printField("Status:", string(r.Status))
	printField("Mode:", string(r.Mode))
	if r.Branch != "" {
		printField("Branch:", r.Branch)
	}
	if r.BaseSHA != "" {
		printField("Base SHA:", r.BaseSHA)
	}
	if r.FinalSHA != "" {
		printField("Final SHA:", r.FinalSHA)
	}
	patch := "no"
	if r.PatchReady {
		patch = "yes"
	}
	printField("Patch Ready:", patch)
	printField("Created:", r.CreatedAt.Format("2006-01-02 15:04:05"))
	if r.FinishedAt != nil {
		printField("Finished:", r.FinishedAt.Format("2006-01-02 15:04:05"))
	}
	if r.Attach != "" {
		printField("Attach:", r.Attach)
	}

	if r.LogTail != "" {
		fmt.Println()
		fmt.Println("--- log tail ---")
		for line := range strings.SplitSeq(r.LogTail, "\n") {
			fmt.Println(line)
		}
	}
}
