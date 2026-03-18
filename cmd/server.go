package cmd

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/wesgrimes/outpost/internal/config"
	"github.com/wesgrimes/outpost/internal/grpcclient"
)

// ServerSetup configures a server, locally or remotely via SSH.
func ServerSetup(args []string) error {
	jsonOutput := false
	var sshTarget string

	for _, arg := range args {
		if arg == "--json" {
			jsonOutput = true
		} else if !strings.HasPrefix(arg, "-") {
			sshTarget = arg
		}
	}

	if sshTarget != "" {
		return remoteServerSetup(sshTarget)
	}

	return localServerSetup(jsonOutput)
}

func localServerSetup(jsonOutput bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}

	base := filepath.Join(home, ".outpost")
	tlsDir := filepath.Join(base, "tls")
	runsDir := filepath.Join(base, "runs")

	if !jsonOutput {
		printBoxTop("Outpost Server Setup", "")
	}

	for _, d := range []string{base, tlsDir, runsDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}

	cfg := config.Default()
	cfg.TLSCert = filepath.Join(tlsDir, "server.pem")
	cfg.TLSKey = filepath.Join(tlsDir, "server-key.pem")
	cfg.TLSCA = filepath.Join(tlsDir, "ca.pem")
	if err := cfg.SaveTo(filepath.Join(base, "config.yaml")); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	if !jsonOutput {
		printCheckItem("Config", filepath.Join(base, "config.yaml"))
	}

	if err := generateTLSCerts(tlsDir); err != nil {
		return fmt.Errorf("generate TLS: %w", err)
	}

	if !jsonOutput {
		printCheckItem("TLS certs", tlsDir)
	}

	// Check prerequisites.
	prereqs := checkPrerequisites()
	if !jsonOutput {
		printCheckItem("Prerequisites", prereqs)
	}

	// Install systemd on Linux.
	if runtime.GOOS == "linux" {
		if err := installSystemd(); err != nil {
			if !jsonOutput {
				printFailItem("Systemd", err.Error())
			}
		} else if !jsonOutput {
			printCheckItem("Systemd", "outpost.service enabled and started")
		}
	}

	if jsonOutput {
		return printSetupJSON(cfg, tlsDir, base)
	}

	printSetupSummary(cfg)
	return nil
}

func printSetupJSON(cfg *config.ServerConfig, tlsDir, base string) error {
	out := map[string]string{
		"token":  cfg.Token,
		"port":   strconv.Itoa(cfg.Port),
		"ca":     filepath.Join(tlsDir, "ca.pem"),
		"config": filepath.Join(base, "config.yaml"),
	}
	data, err := json.Marshal(out)
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func printSetupSummary(cfg *config.ServerConfig) {
	fmt.Fprintln(os.Stderr)
	printBoxRow("  Token: " + cfg.Token)
	printBoxRow("  To connect from another machine:")
	hostname, _ := os.Hostname()
	printBoxRow(fmt.Sprintf("    outpost login %s:%d %s", hostname, cfg.Port, cfg.Token))
	printBoxBottom()
}

func remoteServerSetup(sshTarget string) error {
	printBoxTop("Outpost Server Setup", sshTarget)

	goarch, err := detectRemoteArch(sshTarget)
	if err != nil {
		return err
	}
	printCheckItem("Detected", "linux/"+goarch)

	if err := buildAndUpload(sshTarget, goarch); err != nil {
		return err
	}

	installRemotePrereqs(sshTarget)
	printCheckItem("Prerequisites", "tmux, node, claude")

	setupResult, err := runRemoteSetup(sshTarget)
	if err != nil {
		return err
	}
	printCheckItem("Config", setupResult["config"])

	if _, err := sshRun(sshTarget, "sudo systemctl daemon-reload && sudo systemctl enable outpost && sudo systemctl restart outpost"); err != nil {
		printFailItem("Systemd", err.Error())
	} else {
		printCheckItem("Systemd", "outpost.service enabled and started")
	}

	saveRemoteCredentials(sshTarget, setupResult)
	verifyRemoteConnection()

	fmt.Fprintln(os.Stderr)
	printBoxRow("  Token: " + setupResult["token"])
	printBoxRow("  To connect from another machine:")
	printBoxRow("    outpost login " + sshTarget + ":" + setupResult["port"] + " " + setupResult["token"])
	printBoxBottom()

	return nil
}

func detectRemoteArch(sshTarget string) (string, error) {
	arch, err := sshRun(sshTarget, "uname -m")
	if err != nil {
		return "", fmt.Errorf("detect arch: %w", err)
	}
	arch = strings.TrimSpace(arch)
	if arch == "aarch64" || arch == "arm64" {
		return "arm64", nil
	}
	return "amd64", nil
}

func buildAndUpload(sshTarget, goarch string) error {
	binPath := "/tmp/outpost-linux-" + goarch
	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH="+goarch)
	buildCmd.Dir = findProjectRoot()
	if out, err := buildCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("build: %w: %s", err, out)
	}
	printCheckItem("Binary", "built for linux/"+goarch)

	if err := scpFile(binPath, sshTarget, "/tmp/outpost-upload"); err != nil {
		return fmt.Errorf("upload: %w", err)
	}

	if _, err := sshRun(sshTarget, "sudo mv /tmp/outpost-upload /usr/local/bin/outpost && sudo chmod 755 /usr/local/bin/outpost"); err != nil {
		return fmt.Errorf("install: %w", err)
	}
	printCheckItem("Installed", "/usr/local/bin/outpost")
	return nil
}

func runRemoteSetup(sshTarget string) (map[string]string, error) {
	setupJSON, err := sshRun(sshTarget, "outpost server setup --json")
	if err != nil {
		return nil, fmt.Errorf("remote setup: %w", err)
	}
	var result map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(setupJSON)), &result); err != nil {
		return nil, fmt.Errorf("parse setup result: %w", err)
	}
	return result, nil
}

func saveRemoteCredentials(sshTarget string, setupResult map[string]string) {
	caPath := setupResult["ca"]
	if caPath == "" {
		return
	}
	caData, err := sshRun(sshTarget, fmt.Sprintf("cat %q", caPath))
	if err != nil {
		return
	}
	localCADir := config.ClientConfigDir()
	if err := os.MkdirAll(localCADir, 0o755); err != nil {
		return
	}
	localCAPath := filepath.Join(localCADir, "ca.pem")
	if err := os.WriteFile(localCAPath, []byte(caData), 0o600); err != nil {
		return
	}
	clientCfg := &config.ClientConfig{
		Server: sshTarget + ":" + setupResult["port"],
		Token:  setupResult["token"],
		CACert: localCAPath,
	}
	if err := clientCfg.Save(); err == nil {
		printCheckItem("Credentials", "saved to "+config.ClientConfigDir())
	}
}

func verifyRemoteConnection() {
	client, err := grpcclient.Load()
	if err != nil {
		printFailItem("Connection", err.Error())
		return
	}
	defer logClose(client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := client.HealthCheck(ctx); err != nil {
		printFailItem("Connection", err.Error())
	} else {
		printCheckItem("Connection", "healthy")
	}
}

// ServerDoctor checks server health via the ServerDoctor RPC.
func ServerDoctor(args []string) error {
	client, err := grpcclient.Load()
	if err != nil {
		return err
	}
	defer logClose(client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := client.ServerDoctor(ctx)
	if err != nil {
		return fmt.Errorf("server doctor: %w", err)
	}

	cfg, cfgErr := config.LoadClient()
	serverName := "unknown"
	if cfgErr == nil {
		serverName = cfg.Server
	}

	printBoxTop("Outpost Server Doctor", serverName)
	printCheckItem("Version", result.Version)
	printCheckItem("Uptime", result.Uptime)
	printCheckItem("Disk (runs)", result.DiskFree)

	if result.ClaudeInstalled {
		printCheckItem("Claude Code", "installed")
	} else {
		printFailItem("Claude Code", "not found")
	}

	if result.TmuxInstalled {
		printCheckItem("tmux", "installed")
	} else {
		printFailItem("tmux", "not found")
	}

	printCheckItem("Runs", fmt.Sprintf("%d active / %d max, %d total",
		result.ActiveRuns, result.MaxRuns, result.TotalRuns))

	printBoxBottom()
	return nil
}

func checkPrerequisites() string {
	var found []string
	for _, bin := range []string{"tmux", "node", "claude"} {
		if _, err := exec.LookPath(bin); err == nil {
			found = append(found, bin)
		}
	}
	if len(found) == 0 {
		return "none found"
	}
	return strings.Join(found, ", ")
}

func installSystemd() error {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("detect user: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}

	unit := fmt.Sprintf(`[Unit]
Description=Outpost gRPC Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=%s
ExecStart=/usr/local/bin/outpost serve
WorkingDirectory=%s
Environment=HOME=%s PATH=/usr/local/bin:/usr/bin:/bin
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`, currentUser.Username, home, home)

	unitPath := "/etc/systemd/system/outpost.service"
	if err := os.WriteFile(unitPath, []byte(unit), 0o644); err != nil {
		return fmt.Errorf("write unit: %w", err)
	}

	// Enable and start.
	for _, cmd := range []string{
		"systemctl daemon-reload",
		"systemctl enable outpost",
		"systemctl restart outpost",
	} {
		parts := strings.Fields(cmd)
		if out, err := exec.Command(parts[0], parts[1:]...).CombinedOutput(); err != nil {
			return fmt.Errorf("%s: %w: %s", cmd, err, out)
		}
	}

	return nil
}

func installRemotePrereqs(sshTarget string) {
	// Install tmux if missing.
	_, _ = sshRun(sshTarget, "command -v tmux || sudo apt-get install -y -qq tmux")

	// Install Node.js if missing.
	_, _ = sshRun(sshTarget, "command -v node || (curl -fsSL https://deb.nodesource.com/setup_20.x | sudo bash - && sudo apt-get install -y -qq nodejs)")

	// Install Claude Code if missing.
	_, _ = sshRun(sshTarget, "command -v claude || npm install -g @anthropic-ai/claude-code")
}

func sshRun(target, cmd string) (string, error) {
	out, err := exec.Command("ssh", "-o", "StrictHostKeyChecking=accept-new", target, cmd).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ssh %s: %w: %s", target, err, out)
	}
	return string(out), nil
}

func scpFile(localPath, target, remotePath string) error {
	out, err := exec.Command("scp", "-o", "StrictHostKeyChecking=accept-new", localPath, target+":"+remotePath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("scp: %w: %s", err, out)
	}
	return nil
}

func findProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}

func generateTLSCerts(tlsDir string) error {
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generate CA key: %w", err)
	}

	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Outpost CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("create CA cert: %w", err)
	}

	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		return fmt.Errorf("parse CA cert: %w", err)
	}

	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generate server key: %w", err)
	}

	sans := collectSANs()

	serverTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "Outpost Server"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     sans.dns,
		IPAddresses:  sans.ips,
	}

	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTmpl, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("create server cert: %w", err)
	}

	if err := writePEM(filepath.Join(tlsDir, "ca.pem"), "CERTIFICATE", caCertDER); err != nil {
		return err
	}
	caKeyDER, err := x509.MarshalECPrivateKey(caKey)
	if err != nil {
		return fmt.Errorf("marshal CA key: %w", err)
	}
	if err := writePEM(filepath.Join(tlsDir, "ca-key.pem"), "EC PRIVATE KEY", caKeyDER); err != nil {
		return err
	}
	if err := writePEM(filepath.Join(tlsDir, "server.pem"), "CERTIFICATE", serverCertDER); err != nil {
		return err
	}
	serverKeyDER, err := x509.MarshalECPrivateKey(serverKey)
	if err != nil {
		return fmt.Errorf("marshal server key: %w", err)
	}
	return writePEM(filepath.Join(tlsDir, "server-key.pem"), "EC PRIVATE KEY", serverKeyDER)
}

type sanSet struct {
	dns []string
	ips []net.IP
}

func collectSANs() sanSet {
	s := sanSet{
		dns: []string{"localhost"},
	}

	if hostname, err := os.Hostname(); err == nil {
		s.dns = append(s.dns, hostname)
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return s
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				s.ips = append(s.ips, ipNet.IP)
			}
		}
	}

	return s
}

func writePEM(path, blockType string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer logClose(f)

	return pem.Encode(f, &pem.Block{Type: blockType, Bytes: data})
}

// Server dispatches server subcommands.
func Server(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: outpost server <setup|doctor> [args...]")
	}

	switch args[0] {
	case "setup":
		return ServerSetup(args[1:])
	case "doctor":
		return ServerDoctor(args[1:])
	default:
		return fmt.Errorf("unknown server subcommand: %s", args[0])
	}
}
