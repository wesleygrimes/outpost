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

	"github.com/wesleygrimes/outpost/internal/config"
	"github.com/wesleygrimes/outpost/internal/grpcclient"
	"github.com/wesleygrimes/outpost/internal/ui"
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

	var cl *ui.Checklist
	if !jsonOutput {
		cl = ui.NewChecklist("Server Setup")
	}

	for _, d := range []string{base, tlsDir, runsDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}

	cfg, err := ensureConfigAndTLS(base, tlsDir)
	if err != nil {
		return err
	}

	if cl != nil {
		cl.Success("Config " + filepath.Join(base, "config.yaml"))
		cl.Success("TLS certs generated (" + tlsDir + ")")
	}

	prereqs := checkPrerequisites()
	if cl != nil {
		cl.Success("Prerequisites " + prereqs)
	}

	if runtime.GOOS == "linux" {
		if err := installSystemd(); err != nil {
			if cl != nil {
				cl.Fail("Systemd " + err.Error())
			}
		} else if cl != nil {
			cl.Success("Systemd outpost.service enabled and started")
		}
	}

	if jsonOutput {
		return printSetupJSON(cfg, tlsDir, base)
	}

	hostname, _ := os.Hostname()
	cl.Row("")
	cl.CloseWith("Ready. Token: " + cfg.Token)
	ui.Errln()
	ui.Hint("Connect", fmt.Sprintf("outpost login %s:%d %s", hostname, cfg.Port, cfg.Token))
	return nil
}

// ensureConfigAndTLS loads existing config or creates a fresh one, and
// generates TLS certificates if they don't already exist.
func ensureConfigAndTLS(base, tlsDir string) (*config.ServerConfig, error) {
	configPath := filepath.Join(base, "config.yaml")
	cfg, loadErr := config.LoadFrom(configPath)
	if loadErr != nil || cfg.Port == 0 || cfg.Token == "" {
		cfg = config.Default()
		cfg.TLSCert = filepath.Join(tlsDir, "server.pem")
		cfg.TLSKey = filepath.Join(tlsDir, "server-key.pem")
		cfg.TLSCA = filepath.Join(tlsDir, "ca.pem")
		if u, err := user.Current(); err == nil {
			cfg.SSHUser = u.Username
		}
		if err := cfg.SaveTo(configPath); err != nil {
			return nil, fmt.Errorf("save config: %w", err)
		}
	}

	if _, err := os.Stat(cfg.TLSCA); os.IsNotExist(err) {
		if err := generateTLSCerts(tlsDir); err != nil {
			return nil, fmt.Errorf("generate TLS: %w", err)
		}
	}

	return cfg, nil
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

func remoteServerSetup(sshTarget string) error {
	if err := validateSSHHost(sshTarget); err != nil {
		return err
	}

	cleanup, err := startSSHControlMaster(sshTarget)
	if err != nil {
		return err
	}
	defer cleanup()

	cl := ui.NewChecklist("Provisioning " + sshTarget)
	cl.Success("SSH connection established (" + sshTarget + ")")

	goarch, err := detectRemoteArch(sshTarget)
	if err != nil {
		return err
	}
	cl.Success("Detected linux/" + goarch)

	if err := buildAndUpload(cl, sshTarget, goarch); err != nil {
		return err
	}

	ensureRemoteUser(cl, sshTarget)
	installRemotePrereqs(sshTarget)
	cl.Success("Prerequisites installed (tmux, node, claude)")

	setupResult, err := runRemoteSetup(sshTarget)
	if err != nil {
		return err
	}
	cl.Success("Config " + setupResult["config"])

	if err := installRemoteSystemd(sshTarget); err != nil {
		cl.Fail("Systemd " + err.Error())
	} else {
		cl.Success("Systemd outpost.service enabled and started")
	}

	remoteAddr := resolveSSHHostname(sshTarget)

	saveRemoteCredentials(cl, sshTarget, remoteAddr, setupResult)
	verifyRemoteConnection(cl)

	cl.Row("")
	cl.CloseWith("Ready. This machine is already connected.")
	ui.Errln()
	ui.Hint("Connect", "outpost login "+remoteAddr+":"+setupResult["port"]+" "+setupResult["token"])

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

const releaseBaseURL = "https://git.grimes.pro/wesleygrimes/outpost"

func buildAndUpload(cl *ui.Checklist, sshTarget, goarch string) error {
	binaryName := "outpost-linux-" + goarch
	binPath := filepath.Join(os.TempDir(), binaryName)

	// Download the release binary for the target platform.
	latestURL := releaseBaseURL + "/releases/latest"
	resp, err := exec.Command("curl", "-sS", "-o", "/dev/null", "-w", "%{redirect_url}", latestURL).Output()
	if err != nil {
		return fmt.Errorf("detect latest release: %w", err)
	}

	version := filepath.Base(strings.TrimSpace(string(resp)))
	if version == "" || version == "." {
		return errors.New("could not detect latest release version")
	}

	downloadURL := fmt.Sprintf("%s/releases/download/%s/%s", releaseBaseURL, version, binaryName)
	dlCmd := exec.Command("curl", "-fsSL", "-o", binPath, downloadURL)
	if out, err := dlCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("download %s: %w: %s", downloadURL, err, out)
	}
	cl.Success("Binary " + version + " linux/" + goarch)

	if err := scpFile(binPath, sshTarget, "/tmp/outpost-upload"); err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	_ = os.Remove(binPath)

	if _, err := sshRun(sshTarget, "sudo mv /tmp/outpost-upload /usr/local/bin/outpost && sudo chmod 755 /usr/local/bin/outpost"); err != nil {
		return fmt.Errorf("install: %w", err)
	}
	cl.Success("Installed /usr/local/bin/outpost")
	return nil
}

const serviceUser = "outpost"

func ensureRemoteUser(cl *ui.Checklist, sshTarget string) {
	_, _ = sshRun(sshTarget, fmt.Sprintf(
		"id %s >/dev/null 2>&1 || sudo useradd -r -m -s /bin/bash %s",
		serviceUser, serviceUser,
	))
	cl.Success("User " + serviceUser)
}

func installRemoteSystemd(sshTarget string) error {
	home, err := sshRun(sshTarget, "eval echo ~"+serviceUser)
	if err != nil {
		return fmt.Errorf("resolve home: %w", err)
	}
	home = strings.TrimSpace(home)

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
`, serviceUser, home, home)

	writeCmd := fmt.Sprintf("sudo tee /etc/systemd/system/outpost.service > /dev/null <<'UNIT'\n%sUNIT", unit)
	if _, err := sshRun(sshTarget, writeCmd); err != nil {
		return fmt.Errorf("write unit: %w", err)
	}

	if _, err := sshRun(sshTarget, "sudo systemctl daemon-reload && sudo systemctl enable outpost && sudo systemctl restart outpost"); err != nil {
		return fmt.Errorf("systemctl: %w", err)
	}
	return nil
}

func runRemoteSetup(sshTarget string) (map[string]string, error) {
	setupJSON, err := sshRun(sshTarget, fmt.Sprintf("sudo -u %s outpost server setup --json", serviceUser))
	if err != nil {
		return nil, fmt.Errorf("remote setup: %w", err)
	}
	var result map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(setupJSON)), &result); err != nil {
		return nil, fmt.Errorf("parse setup result: %w", err)
	}
	return result, nil
}

func saveRemoteCredentials(cl *ui.Checklist, sshAlias, remoteAddr string, setupResult map[string]string) {
	caPath := setupResult["ca"]
	if caPath == "" {
		return
	}
	caData, err := sshRun(sshAlias, fmt.Sprintf("cat %q", caPath))
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
		Server: remoteAddr + ":" + setupResult["port"],
		Token:  setupResult["token"],
		CACert: localCAPath,
	}
	if err := clientCfg.Save(); err == nil {
		cl.Success("Credentials saved to " + config.ClientConfigDir())
	}
}

func verifyRemoteConnection(cl *ui.Checklist) {
	client, err := grpcclient.Load()
	if err != nil {
		cl.Fail("Connection " + err.Error())
		return
	}
	defer logClose(client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := client.HealthCheck(ctx); err != nil {
		cl.Fail("Connection " + err.Error())
	} else {
		cl.Success("Connection healthy")
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

	cl := ui.NewChecklist("Server Health " + serverName)
	cl.Field("Version", result.Version)
	cl.Field("Uptime", result.Uptime)
	cl.Field("Disk (runs)", result.DiskFree)
	cl.Row("")

	if result.ClaudeInstalled {
		cl.Success("Claude Code installed")
	} else {
		cl.Fail("Claude Code not found")
	}

	if result.TmuxInstalled {
		cl.Success("tmux installed")
	} else {
		cl.Fail("tmux not found")
	}

	cl.Success(fmt.Sprintf("Runs %d active / %d max, %d total",
		result.ActiveRuns, result.MaxRuns, result.TotalRuns))

	cl.Close()
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

// resolveSSHHostname resolves an SSH config Host alias to the actual HostName
// using `ssh -G`, which dumps the resolved config without connecting.
func resolveSSHHostname(alias string) string {
	out, err := exec.Command("ssh", "-G", alias).Output()
	if err != nil {
		return alias
	}
	for line := range strings.SplitSeq(string(out), "\n") {
		if addr, ok := strings.CutPrefix(line, "hostname "); ok {
			return addr
		}
	}
	return alias
}

// validateSSHHost checks that the target looks like an SSH config Host alias
// (not a bare IP or FQDN), and prints a helpful snippet if not.
func validateSSHHost(target string) error {
	// If it contains @, a dot, or a colon, it's not a plain Host alias.
	if strings.ContainsAny(target, "@.:") {
		ui.Errf(`outpost requires an SSH config Host entry.

Add this to ~/.ssh/config:

    Host %s
        HostName %s
        User <your-user>
        IdentityFile ~/.ssh/id_ed25519

Then run: outpost server setup %s
`, suggestAlias(target), target, suggestAlias(target))
		return fmt.Errorf("use an SSH config Host alias, not %q", target)
	}

	// Verify the host exists in SSH config by checking connectivity.
	out, err := exec.Command("ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", target, "true").CombinedOutput()
	if err != nil {
		hint := strings.TrimSpace(string(out))
		if hint == "" {
			hint = err.Error()
		}
		return fmt.Errorf("SSH to %q failed: %s\n\nCheck your ~/.ssh/config entry", target, hint)
	}

	return nil
}

// suggestAlias extracts a short alias from an IP or hostname.
func suggestAlias(target string) string {
	// Strip user@ prefix.
	if i := strings.Index(target, "@"); i >= 0 {
		target = target[i+1:]
	}
	// Strip port.
	if host, _, err := net.SplitHostPort(target); err == nil {
		target = host
	}
	// Use first segment of hostname.
	if i := strings.Index(target, "."); i > 0 {
		return target[:i]
	}
	return "myserver"
}

// sshControlPath returns a deterministic socket path for ControlMaster.
func sshControlPath(target string) string {
	return filepath.Join(os.TempDir(), "outpost-ssh-"+target+".sock")
}

// startSSHControlMaster opens a persistent SSH connection that all subsequent
// ssh/scp calls multiplex over. Returns a cleanup function.
func startSSHControlMaster(target string) (func(), error) {
	sockPath := sshControlPath(target)

	cmd := exec.Command("ssh",
		"-o", "ControlMaster=yes",
		"-o", "ControlPath="+sockPath,
		"-o", "ControlPersist=60",
		"-N", target,
	)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start SSH control master: %w", err)
	}

	// Wait briefly for the socket to appear.
	for range 20 {
		if _, err := os.Stat(sockPath); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	cleanup := func() {
		_ = exec.Command("ssh", "-o", "ControlPath="+sockPath, "-O", "exit", target).Run()
		_ = cmd.Wait()
	}
	return cleanup, nil
}

func sshRun(target, cmd string) (string, error) {
	sockPath := sshControlPath(target)
	args := []string{"-o", "ControlPath=" + sockPath, target, cmd}
	out, err := exec.Command("ssh", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ssh %s: %w: %s", target, err, out)
	}
	return string(out), nil
}

func scpFile(localPath, target, remotePath string) error {
	sockPath := sshControlPath(target)
	args := []string{"-o", "ControlPath=" + sockPath, localPath, target + ":" + remotePath}
	out, err := exec.Command("scp", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("scp: %w: %s", err, out)
	}
	return nil
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
