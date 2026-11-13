package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCLIBinaryIn(t *testing.T) {
	dir := t.TempDir()
	if _, ok := cliBinaryIn(dir); ok {
		t.Fatal("empty dir should have no CLI binary")
	}
	name := "meridian-cli"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	p, ok := cliBinaryIn(dir)
	if !ok || filepath.Base(p) != name {
		t.Errorf("expected to find %s, got %q ok=%v", name, p, ok)
	}
}

func TestLinkCLIAndIsLinkedTo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on Windows")
	}
	src := t.TempDir()
	bin := filepath.Join(src, "meridian-cli")
	if err := os.WriteFile(bin, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	linkDir := filepath.Join(t.TempDir(), "bin") // doesn't exist yet
	link, err := linkCLI(bin, linkDir)
	if err != nil {
		t.Fatalf("linkCLI: %v", err)
	}
	if !isLinkedTo(link, bin) {
		t.Errorf("link %s should resolve to %s", link, bin)
	}
	// Re-linking is idempotent (replaces the existing link).
	if _, err := linkCLI(bin, linkDir); err != nil {
		t.Errorf("relink should succeed: %v", err)
	}
	// A link to a different target is not "linked to bin".
	other := filepath.Join(src, "other")
	_ = os.WriteFile(other, []byte("y"), 0o755)
	_ = os.Remove(link)
	_ = os.Symlink(other, link)
	if isLinkedTo(link, bin) {
		t.Error("link to a different target should not report as linked to bin")
	}
}

func TestInstallCopy(t *testing.T) {
	src := t.TempDir()
	bin := filepath.Join(src, "meridian-cli")
	if err := os.WriteFile(bin, []byte("binary-bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "bin") // doesn't exist yet
	dst, err := installCopy(bin, dir)
	if err != nil {
		t.Fatalf("installCopy: %v", err)
	}
	b, err := os.ReadFile(dst)
	if err != nil || string(b) != "binary-bytes" {
		t.Errorf("copied content wrong: %q err=%v", b, err)
	}
	if !isRegularFile(dst) {
		t.Error("destination should be a regular file (a real copy, not a symlink)")
	}
	if runtime.GOOS != "windows" {
		if fi, _ := os.Stat(dst); fi.Mode()&0o100 == 0 {
			t.Error("copied binary should be executable")
		}
	}
	// No stray temp file left behind.
	if _, err := os.Stat(dst + ".tmp"); !os.IsNotExist(err) {
		t.Error("temp file should be gone after rename")
	}
	// Re-copy is idempotent.
	if _, err := installCopy(bin, dir); err != nil {
		t.Errorf("re-copy should succeed: %v", err)
	}
}

func TestDirOnPATH(t *testing.T) {
	sep := string(os.PathListSeparator)
	path := "/usr/bin" + sep + "/usr/local/bin" + sep + "/opt/x/"
	if !dirOnPATH("/usr/local/bin", path) {
		t.Error("/usr/local/bin should be detected on PATH")
	}
	if !dirOnPATH("/opt/x", path) {
		t.Error("trailing-slash entry should still match after Clean")
	}
	if dirOnPATH("/nowhere", path) {
		t.Error("absent dir should not be on PATH")
	}
}

func TestCLIInfoShape(t *testing.T) {
	// In the test binary there's no bundled meridian-cli next to the exe, so the
	// tool reports unavailable — but the OS/Supported fields are always set.
	a := New()
	info := a.CLIInfo()
	if info.OS != runtime.GOOS {
		t.Errorf("OS = %q, want %q", info.OS, runtime.GOOS)
	}
	if info.Supported != (runtime.GOOS == "darwin") {
		t.Errorf("Supported = %v on %s", info.Supported, runtime.GOOS)
	}
}
