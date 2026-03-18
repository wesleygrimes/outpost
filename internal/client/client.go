package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wesgrimes/outpost/internal/store"
)

// Client talks to a remote Outpost server.
type Client struct {
	URL    string
	Token  string
	client *http.Client
}

// Load creates a Client from env vars or dotfiles (~/.outpost-url, ~/.outpost-token).
func Load() (*Client, error) {
	url := os.Getenv("OUTPOST_URL")
	token := os.Getenv("OUTPOST_TOKEN")

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("finding home directory: %w", err)
	}

	if url == "" {
		data, readErr := os.ReadFile(filepath.Join(home, ".outpost-url"))
		if readErr != nil {
			return nil, errors.New("OUTPOST_URL not set and ~/.outpost-url not found\nRun: outpost login <url> <token>")
		}

		url = strings.TrimSpace(string(data))
	}

	if token == "" {
		data, readErr := os.ReadFile(filepath.Join(home, ".outpost-token"))
		if readErr != nil {
			return nil, errors.New("OUTPOST_TOKEN not set and ~/.outpost-token not found\nRun: outpost login <url> <token>")
		}

		token = strings.TrimSpace(string(data))
	}

	return &Client{
		URL:    strings.TrimRight(url, "/"),
		Token:  token,
		client: &http.Client{},
	}, nil
}

// HandoffParams configures a handoff request.
type HandoffParams struct {
	PlanPath    string
	ArchivePath string
	Mode        string
	Name        string
	Branch      string
	MaxTurns    int
	Subdir      string
}

// HandoffResult holds the response from a successful handoff.
type HandoffResult struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Attach string `json:"attach"`
}

// Handoff submits a plan and archive to the server.
func (c *Client) Handoff(params *HandoffParams) (*HandoffResult, error) {
	plan, err := os.ReadFile(params.PlanPath)
	if err != nil {
		return nil, fmt.Errorf("reading plan: %w", err)
	}

	body, contentType, err := buildHandoffForm(plan, params)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL+"/handoff", body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", contentType)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connecting to outpost: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("outpost at capacity:\n%s", string(respBody))
	}

	if resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("handoff failed (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var result HandoffResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &result, nil
}

// ListRuns returns all runs from the server.
func (c *Client) ListRuns() ([]*store.Run, error) {
	data, err := c.get("/runs")
	if err != nil {
		return nil, err
	}

	var runs []*store.Run
	if err := json.Unmarshal(data, &runs); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return runs, nil
}

// GetRun returns a single run by ID.
func (c *Client) GetRun(id string) (*store.Run, error) {
	data, err := c.get("/runs/" + id)
	if err != nil {
		return nil, err
	}

	var run store.Run
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &run, nil
}

// DownloadPatch saves a run's result patch to the given path.
func (c *Client) DownloadPatch(id, destPath string) error {
	data, err := c.get("/runs/" + id + "/patch")
	if err != nil {
		return err
	}

	return os.WriteFile(destPath, data, 0o600)
}

// KillRun terminates a running session and returns the updated run.
func (c *Client) KillRun(id string) (*store.Run, error) {
	data, status, err := c.request(http.MethodDelete, "/runs/"+id)
	if err != nil {
		return nil, err
	}

	if status != http.StatusOK {
		return nil, fmt.Errorf("kill failed (HTTP %d): %s", status, strings.TrimSpace(string(data)))
	}

	var run store.Run
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &run, nil
}

// Cleanup removes a finished run's files from the server.
func (c *Client) Cleanup(id string) error {
	data, status, err := c.request(http.MethodPost, "/runs/"+id+"/cleanup")
	if err != nil {
		return err
	}

	if status != http.StatusOK {
		return fmt.Errorf("cleanup failed (HTTP %d): %s", status, strings.TrimSpace(string(data)))
	}

	return nil
}

// get sends a GET request and returns the response body.
func (c *Client) get(path string) ([]byte, error) {
	data, status, err := c.request(http.MethodGet, path)
	if err != nil {
		return nil, err
	}

	if status != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", status, strings.TrimSpace(string(data)))
	}

	return data, nil
}

// request sends an HTTP request with auth and returns the response body and status code.
func (c *Client) request(method, path string) (body []byte, status int, err error) {
	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, method, c.URL+path, http.NoBody)
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("connecting to outpost: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("reading response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

func buildHandoffForm(plan []byte, params *HandoffParams) (io.Reader, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	for _, field := range []struct{ key, val string }{
		{"plan", string(plan)},
		{"mode", params.Mode},
		{"name", params.Name},
		{"branch", params.Branch},
		{"max_turns", strconv.Itoa(params.MaxTurns)},
		{"subdir", params.Subdir},
	} {
		if err := w.WriteField(field.key, field.val); err != nil {
			return nil, "", fmt.Errorf("writing field %s: %w", field.key, err)
		}
	}

	part, err := w.CreateFormFile("archive", "archive.tar.gz")
	if err != nil {
		return nil, "", fmt.Errorf("creating archive field: %w", err)
	}

	f, err := os.Open(params.ArchivePath)
	if err != nil {
		return nil, "", fmt.Errorf("opening archive: %w", err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(part, f); err != nil {
		return nil, "", fmt.Errorf("writing archive: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, "", fmt.Errorf("closing form: %w", err)
	}

	return &buf, w.FormDataContentType(), nil
}
