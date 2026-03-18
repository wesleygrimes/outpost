package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClientConfig_RoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	original := &ClientConfig{
		Server: "outpost.example.com:7600",
		Token:  "abc123",
		CACert: "/path/to/ca.pem",
	}

	if err := original.SaveTo(path); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	// Verify YAML contains expected fields.
	content := string(data)
	for _, want := range []string{"server:", "token:", "ca_cert:"} {
		if !contains(content, want) {
			t.Errorf("YAML missing %q:\n%s", want, content)
		}
	}

	// Verify permissions are restrictive.
	info, _ := os.Stat(path)
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("permissions = %o, want 600", perm)
	}
}

func TestClientConfig_SaveTo_CreatesDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "config.yaml")

	cfg := &ClientConfig{Server: "host:7600", Token: "tok"}
	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}

func TestClientConfig_CACertOptional(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	original := &ClientConfig{
		Server: "host:7600",
		Token:  "tok",
	}
	if err := original.SaveTo(path); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}

	// ca_cert should be omitted from YAML when empty.
	data, _ := os.ReadFile(path)
	if contains(string(data), "ca_cert") {
		t.Errorf("empty ca_cert should be omitted from YAML:\n%s", data)
	}
}

func TestMigrateOldDotfiles(t *testing.T) {
	// Use a temp HOME to avoid touching real files.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	// Write old-style dotfiles.
	os.WriteFile(filepath.Join(home, ".outpost-url"), []byte("outpost.example.com:7600\n"), 0o600)
	os.WriteFile(filepath.Join(home, ".outpost-token"), []byte("  mytoken  \n"), 0o600)
	os.WriteFile(filepath.Join(home, ".outpost-ca.pem"), []byte("-----BEGIN CERTIFICATE-----\nfake\n-----END CERTIFICATE-----\n"), 0o600)

	migrated, err := MigrateOldDotfiles()
	if err != nil {
		t.Fatalf("MigrateOldDotfiles: %v", err)
	}
	if !migrated {
		t.Fatal("expected migration to occur")
	}

	// Verify new config was written.
	configPath := filepath.Join(home, ".config", "outpost", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("new config not created: %v", err)
	}

	content := string(data)
	if !contains(content, "outpost.example.com:7600") {
		t.Errorf("server not migrated:\n%s", content)
	}
	if !contains(content, "mytoken") {
		t.Errorf("token not migrated (should be trimmed):\n%s", content)
	}

	// Verify CA cert was copied.
	caPath := filepath.Join(home, ".config", "outpost", "ca.pem")
	if _, err := os.Stat(caPath); err != nil {
		t.Errorf("CA cert not copied: %v", err)
	}

	// Verify old files were cleaned up.
	for _, name := range []string{".outpost-url", ".outpost-token", ".outpost-ca.pem"} {
		if _, err := os.Stat(filepath.Join(home, name)); err == nil {
			t.Errorf("old file %s should have been removed", name)
		}
	}
}

func TestMigrateOldDotfiles_NothingToMigrate(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	migrated, err := MigrateOldDotfiles()
	if err != nil {
		t.Fatalf("MigrateOldDotfiles: %v", err)
	}
	if migrated {
		t.Fatal("expected no migration when dotfiles don't exist")
	}
}

func TestMigrateOldDotfiles_PartialFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Only URL, no token.
	os.WriteFile(filepath.Join(home, ".outpost-url"), []byte("host:7600"), 0o600)

	migrated, err := MigrateOldDotfiles()
	if err != nil {
		t.Fatalf("MigrateOldDotfiles: %v", err)
	}
	if migrated {
		t.Fatal("should not migrate with only partial dotfiles")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := range len(s) - len(substr) + 1 {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
