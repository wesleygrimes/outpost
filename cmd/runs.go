package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/wesgrimes/outpost/internal/config"
	"github.com/wesgrimes/outpost/internal/store"
)

// Runs lists runs or shows detail for a single run.
// args is os.Args[2:].
func Runs(args []string) {
	cfg, err := config.Load()
	if err != nil {
		// Config not available; try disk fallback directly.
		fmt.Fprintln(os.Stderr, "Config not found, scanning disk...")
		diskFallbackList()

		return
	}

	if len(args) == 0 {
		listRuns(cfg)
	} else {
		showRun(cfg, args[0])
	}
}

func listRuns(cfg *config.Config) {
	url := fmt.Sprintf("http://localhost:%d/runs", cfg.Server.Port)

	body, err := apiGet(url, cfg.Server.Token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "API unavailable (%v), falling back to disk scan...\n", err)
		diskFallbackList()

		return
	}

	var runs []store.Run
	if err := json.Unmarshal(body, &runs); err != nil {
		fmt.Fprintf(os.Stderr, "parsing response: %v\n", err)
		os.Exit(1)
	}

	if len(runs) == 0 {
		fmt.Println("No runs found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintln(w, "ID\tSTATUS\tMODE\tCREATED") //nolint:errcheck // writing to stdout

	for i := range runs {
		r := &runs[i]
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.ID, r.Status, r.Mode, relativeTime(r.CreatedAt)) //nolint:errcheck // writing to stdout
	}

	_ = w.Flush()
}

func showRun(cfg *config.Config, id string) {
	url := fmt.Sprintf("http://localhost:%d/runs/%s", cfg.Server.Port, id)

	body, err := apiGet(url, cfg.Server.Token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "API unavailable: %v\n", err)
		os.Exit(1)
	}

	var run store.Run
	if err := json.Unmarshal(body, &run); err != nil {
		fmt.Fprintf(os.Stderr, "parsing response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("ID:          %s\n", run.ID)
	fmt.Printf("Name:        %s\n", run.Name)
	fmt.Printf("Mode:        %s\n", run.Mode)
	fmt.Printf("Status:      %s\n", run.Status)
	fmt.Printf("Base SHA:    %s\n", run.BaseSHA)

	if run.FinalSHA != "" {
		fmt.Printf("Final SHA:   %s\n", run.FinalSHA)
	}

	fmt.Printf("Created:     %s\n", run.CreatedAt.Format(time.RFC3339))

	if run.FinishedAt != nil {
		fmt.Printf("Finished:    %s\n", run.FinishedAt.Format(time.RFC3339))
	}

	fmt.Printf("Patch Ready: %v\n", run.PatchReady)

	if run.Attach != "" {
		fmt.Printf("Attach:      %s\n", run.Attach)
	}

	if run.LogTail != "" {
		fmt.Println()
		fmt.Println("--- Log Tail ---")
		fmt.Println(run.LogTail)
	}
}

func diskFallbackList() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot determine home directory: %v\n", err)
		return
	}

	runsDir := filepath.Join(home, ".outpost", "runs")

	entries, err := os.ReadDir(runsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot read %s: %v\n", runsDir, err)
		return
	}

	if len(entries) == 0 {
		fmt.Println("No runs found on disk.")
		return
	}

	type diskRun struct {
		id      string
		modTime time.Time
		status  string
	}

	var runs []diskRun

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		runDir := filepath.Join(runsDir, e.Name())

		info, err := e.Info()
		if err != nil {
			continue
		}

		status := "unknown"
		if fileExistsAt(filepath.Join(runDir, "result.patch")) {
			status = "complete (patch ready)"
		} else if fileExistsAt(filepath.Join(runDir, "output.log")) {
			status = "has output"
		}

		runs = append(runs, diskRun{
			id:      e.Name(),
			modTime: info.ModTime(),
			status:  status,
		})
	}

	sort.Slice(runs, func(i, j int) bool {
		return runs[i].modTime.After(runs[j].modTime)
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintln(w, "ID\tSTATUS\tMODIFIED") //nolint:errcheck // writing to stdout

	for _, r := range runs {
		fmt.Fprintf(w, "%s\t%s\t%s\n", r.id, r.status, relativeTime(r.modTime)) //nolint:errcheck // writing to stdout
	}

	_ = w.Flush()
}

func apiGet(url, token string) ([]byte, error) {
	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return body, nil
}

func relativeTime(t time.Time) string {
	d := time.Since(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d hours ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%d days ago", int(d.Hours()/24))
	}
}

func fileExistsAt(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
