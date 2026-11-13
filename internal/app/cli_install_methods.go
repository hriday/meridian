package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hriday/meridian/internal/cli"
)

// CLICommands returns the full command-line reference, grouped by area — the
// same catalog that drives `meridian-cli help`, so the GUI reference page and
// the CLI can't drift apart.
func (a *App) CLICommands() []cli.CommandGroup {
	return cli.Catalog()
}

// CLIInfo describes the bundled meridian-cli: whether this build ships it, and
// whether it's been linked onto the user's PATH.
type CLIInfo struct {
	OS        string `json:"os"`
	Available bool   `json:"available"` // the bundled binary was found next to the app
	Path      string `json:"path"`      // the bundled binary's path
	Installed bool   `json:"installed"` // a symlink to it exists in the link dir
	LinkPath  string `json:"linkPath"`  // where it is / would be linked
	OnPath    bool   `json:"onPath"`    // the link dir is on $PATH
	Supported bool   `json:"supported"` // GUI install works on this OS
}

// guiInstallSupported is the set of OSes where the one-click installer works.
func guiInstallSupported() bool { return runtime.GOOS == "darwin" || runtime.GOOS == "linux" }

// CLIInfo reports the state of the bundled command-line tool.
func (a *App) CLIInfo() CLIInfo {
	info := CLIInfo{OS: runtime.GOOS, Supported: guiInstallSupported()}
	exeDir, err := appExeDir()
	if err != nil {
		return info
	}
	bin, ok := cliBinaryIn(exeDir)
	if !ok {
		return info
	}
	info.Available = true
	info.Path = bin

	dir := cliLinkDir()
	info.LinkPath = filepath.Join(dir, "meridian-cli")
	// macOS links (its .app is at a stable path); Linux copies (the AppImage
	// mount is ephemeral), so "installed" means a regular file is present.
	if runtime.GOOS == "darwin" {
		info.Installed = isLinkedTo(info.LinkPath, bin)
	} else {
		info.Installed = isRegularFile(info.LinkPath)
	}
	info.OnPath = dirOnPATH(dir, os.Getenv("PATH"))
	return info
}

// InstallCLI puts the bundled meridian-cli on the user's PATH so it runs as
// `meridian-cli` from any terminal. macOS symlinks the in-bundle binary (the
// .app sits at a stable path); Linux copies it out (an AppImage mounts at an
// ephemeral path, so a symlink would break on exit). Windows returns guidance.
func (a *App) InstallCLI() (CLIInfo, error) {
	if !guiInstallSupported() {
		return a.CLIInfo(), fmt.Errorf("one-click install isn't available on %s yet; add the folder holding meridian-cli to your PATH", runtime.GOOS)
	}
	exeDir, err := appExeDir()
	if err != nil {
		return a.CLIInfo(), err
	}
	bin, ok := cliBinaryIn(exeDir)
	if !ok {
		return a.CLIInfo(), fmt.Errorf("this build doesn't bundle the command-line tool")
	}
	dir := cliLinkDir()
	if runtime.GOOS == "darwin" {
		_, err = linkCLI(bin, dir)
	} else {
		_, err = installCopy(bin, dir)
	}
	if err != nil {
		return a.CLIInfo(), err
	}
	return a.CLIInfo(), nil
}

// installCopy copies bin to <dir>/meridian-cli (executable), via a temp file and
// rename so a concurrent run never sees a half-written binary.
func installCopy(bin, dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("couldn't create %s: %w", dir, err)
	}
	dst := filepath.Join(dir, "meridian-cli")
	in, err := os.Open(bin)
	if err != nil {
		return "", err
	}
	defer in.Close()
	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return "", err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return "", err
	}
	return dst, nil
}

func isRegularFile(p string) bool {
	fi, err := os.Lstat(p)
	return err == nil && fi.Mode().IsRegular()
}

// appExeDir is the directory of the running executable, symlinks resolved. On
// macOS that's meridian.app/Contents/MacOS.
func appExeDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Dir(exe), nil
}

// cliBinaryIn returns the bundled CLI path within dir, and whether it exists.
func cliBinaryIn(dir string) (string, bool) {
	name := "meridian-cli"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	p := filepath.Join(dir, name)
	if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
		return p, true
	}
	return p, false
}

// cliLinkDir is where the symlink goes: /usr/local/bin when it's writable (it's
// almost always on PATH), else ~/.local/bin.
func cliLinkDir() string {
	const usrLocal = "/usr/local/bin"
	if dirWritable(usrLocal) {
		return usrLocal
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return usrLocal
	}
	return filepath.Join(home, ".local", "bin")
}

// linkCLI creates dir if needed and symlinks bin to <dir>/meridian-cli,
// replacing any existing link. Returns the link path.
func linkCLI(bin, dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("couldn't create %s: %w", dir, err)
	}
	link := filepath.Join(dir, "meridian-cli")
	_ = os.Remove(link) // replace a stale or broken link; ignore "not exists"
	if err := os.Symlink(bin, link); err != nil {
		return "", fmt.Errorf("couldn't link %s: %w", link, err)
	}
	return link, nil
}

// isLinkedTo reports whether link is a symlink resolving to bin.
func isLinkedTo(link, bin string) bool {
	target, err := filepath.EvalSymlinks(link)
	if err != nil {
		return false
	}
	want, err := filepath.EvalSymlinks(bin)
	if err != nil {
		want = bin
	}
	return target == want
}

// dirOnPATH reports whether dir is one of the colon-separated entries in pathEnv.
func dirOnPATH(dir, pathEnv string) bool {
	clean := filepath.Clean(dir)
	for _, p := range strings.Split(pathEnv, string(os.PathListSeparator)) {
		if p != "" && filepath.Clean(p) == clean {
			return true
		}
	}
	return false
}

// dirWritable reports whether dir exists and we can create a file in it.
func dirWritable(dir string) bool {
	probe := filepath.Join(dir, ".meridian-write-probe")
	f, err := os.OpenFile(probe, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return false
	}
	f.Close()
	_ = os.Remove(probe)
	return true
}
