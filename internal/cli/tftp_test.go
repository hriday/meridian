package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveTFTPFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "grabs") // doesn't exist yet
	path, err := saveTFTPFile(dir, "router-config.txt", []byte("hostname R1\n"))
	if err != nil {
		t.Fatalf("saveTFTPFile: %v", err)
	}
	if filepath.Base(path) != "router-config.txt" {
		t.Errorf("unexpected path %q", path)
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b) != "hostname R1\n" {
		t.Errorf("content not written: %q err=%v", b, err)
	}
}

func TestSaveTFTPFileSanitizesName(t *testing.T) {
	dir := t.TempDir()
	// A path-traversal name must collapse to its base inside dir.
	path, err := saveTFTPFile(dir, "../../etc/passwd", []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(path) != dir {
		t.Errorf("file escaped the output dir: %q (dir %q)", path, dir)
	}
	if filepath.Base(path) != "passwd" {
		t.Errorf("name not reduced to base: %q", path)
	}
}

func TestRunTFTPRejectsPositional(t *testing.T) {
	var o, e bytes.Buffer
	if code := runTFTP([]string{"stray"}, &o, &e); code != 2 {
		t.Errorf("tftp with a positional: got exit %d, want 2", code)
	}
}
