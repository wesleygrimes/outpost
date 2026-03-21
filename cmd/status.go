package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/wesleygrimes/outpost/internal/config"
	"github.com/wesleygrimes/outpost/internal/grpcclient"
	"github.com/wesleygrimes/outpost/internal/store"
	"github.com/wesleygrimes/outpost/internal/ui"
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

	cfg, _ := config.LoadClient()
	serverName := ""
	if cfg != nil {
		serverName = cfg.Server
	}
	ui.Header("Runs on " + serverName)
	printDashboard(runs)
	return nil
}

func followLogs(ctx context.Context, client *grpcclient.Client, id string) error {
	ui.Errf("following run %s...\n", id)

	stream, err := client.TailLogs(ctx, id, true)
	if err != nil {
		return err
	}

	for {
		entry, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			ui.Errln("run completed")
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
	var visible []*store.Run

	for _, r := range runs {
		switch r.Status {
		case store.StatusPending, store.StatusRunning:
			active++
			visible = append(visible, r)
		case store.StatusComplete:
			complete++
			visible = append(visible, r)
		case store.StatusFailed:
			failed++
			visible = append(visible, r)
		case store.StatusDropped:
			dropped++
		}
	}

	if len(visible) == 0 {
		ui.Errln("\n  No active runs.")
		return
	}

	ui.Errln()
	t := ui.NewTable("ID", "Branch", "Mode", "Status", "Age", "Patch")
	for _, r := range visible {
		branch := r.Branch
		if branch == "" {
			branch = "-"
		}
		age := formatAge(r.CreatedAt)
		if r.FinishedAt != nil {
			age = formatAge(*r.FinishedAt)
		}
		patch := ""
		if r.PatchReady {
			patch = "yes"
		}
		t.Row(ui.Amber(r.ID), branch, string(r.Mode), statusSymbol(r.Status), age, patch)
	}

	total := active + complete + failed + dropped
	t.Footer(
		fmt.Sprintf("%d runs total", total),
		fmt.Sprintf("%d running", active),
		fmt.Sprintf("%d done", complete),
		fmt.Sprintf("%d failed", failed),
	)
	t.Render()
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

	statusSym := statusSymbol(r.Status)
	ui.Header(fmt.Sprintf("Run %s %s %s", ui.Amber(r.ID), ui.Dim("│"), statusSym))
	printRunDetail(r)
	return nil
}

func printRunDetail(r *store.Run) {
	ui.Errln()
	ui.Field("Name", r.Name)
	ui.Field("Mode", string(r.Mode))
	if r.Branch != "" {
		ui.Field("Branch", r.Branch)
	}
	if r.BaseSHA != "" {
		ui.Field("Base SHA", r.BaseSHA)
	}
	if r.FinalSHA != "" {
		ui.Field("Final SHA", r.FinalSHA)
	}
	patch := "no"
	if r.PatchReady {
		patch = "yes"
	}
	ui.Field("Patch Ready", patch)
	ui.Field("Created", r.CreatedAt.Format("2006-01-02 15:04:05"))
	if r.FinishedAt != nil {
		ui.Field("Finished", r.FinishedAt.Format("2006-01-02 15:04:05"))
	}
	if attach := attachCmd(r.Attach, r.AttachLocal); attach != "" {
		ui.Field("Attach", attach)
	}

	if r.LogTail != "" {
		ui.Errln()
		ui.Errln("  " + ui.Dim("--- log tail ---"))
		for line := range strings.SplitSeq(r.LogTail, "\n") {
			ui.Errln("  " + line)
		}
	}
}

func statusSymbol(s store.Status) string {
	switch s {
	case store.StatusRunning, store.StatusPending:
		return ui.Spin(string(s))
	case store.StatusComplete:
		return ui.Symbol(ui.SymDot, string(s))
	case store.StatusFailed:
		return ui.Fail(string(s))
	case store.StatusDropped:
		return ui.Dim(string(s))
	default:
		return string(s)
	}
}
