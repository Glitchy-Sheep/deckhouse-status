package motd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	scriptName    = "99-deckhouse-status"
	updateMotdDir = "/etc/update-motd.d"
	profileDir    = "/etc/profile.d"
)

// Install creates a login script that runs deckhouse-status on every SSH session.
// Auto-detects the best method: update-motd.d (Ubuntu/Debian) or profile.d (universal).
func Install() {
	requireRoot("install-motd")

	binPath, err := detectBinaryPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot detect binary path: %v\n", err)
		os.Exit(1)
	}

	scriptPath, content := chooseMethod(binPath)

	if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot write %s: %v\n", scriptPath, err)
		os.Exit(1)
	}

	fmt.Printf("✅ Installed login script: %s\n", scriptPath)
	fmt.Printf("   Binary: %s\n", binPath)
	fmt.Printf("   Deckhouse status will be shown on every login.\n")
	fmt.Printf("\n   To edit:   sudo deckhouse-status edit-motd\n")
	fmt.Printf("   To remove: sudo deckhouse-status uninstall-motd\n")
}

// Uninstall removes the login script from both possible locations.
func Uninstall() {
	requireRoot("uninstall-motd")

	removed := false

	for _, dir := range []string{updateMotdDir, profileDir} {
		path := scriptPath(dir)
		if _, err := os.Stat(path); err == nil {
			if err := os.Remove(path); err != nil {
				fmt.Fprintf(os.Stderr, "Error: cannot remove %s: %v\n", path, err)
				os.Exit(1)
			}
			fmt.Printf("✅ Removed: %s\n", path)
			removed = true
		}
	}

	if !removed {
		fmt.Println("ℹ️  Nothing to remove — MOTD script is not installed.")
	}
}

// Edit opens the installed MOTD script in an editor so the user can customize flags.
func Edit() {
	requireRoot("edit-motd")

	path := findInstalled()
	if path == "" {
		fmt.Fprintln(os.Stderr, "Error: MOTD script is not installed.")
		fmt.Fprintln(os.Stderr, "Run 'sudo deckhouse-status install-motd' first.")
		os.Exit(1)
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// requireRoot re-executes the current command under sudo if not running as root.
func requireRoot(subcmd string) {
	if os.Geteuid() == 0 {
		return
	}

	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot detect binary path: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Root privileges required. Re-running with sudo...\n")
	args := append([]string{exe, subcmd}, os.Args[2:]...)
	cmd := exec.Command("sudo", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
	os.Exit(0)
}

func findInstalled() string {
	for _, dir := range []string{updateMotdDir, profileDir} {
		path := scriptPath(dir)
		if _, err := os.Stat(path); err == nil {
			return path
		}
		// profile.d variant has .sh suffix
		if _, err := os.Stat(path + ".sh"); err == nil {
			return path + ".sh"
		}
	}
	return ""
}

func detectBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

func chooseMethod(binPath string) (path, content string) {
	if isDir(updateMotdDir) {
		return scriptPath(updateMotdDir), motdScript(binPath)
	}
	return scriptPath(profileDir) + ".sh", profileScript(binPath)
}

func scriptPath(dir string) string {
	return filepath.Join(dir, scriptName)
}

func motdScript(binPath string) string {
	return fmt.Sprintf(`#!/bin/bash
# Installed by: deckhouse-status install-motd
sudo %s 2>/dev/null || true
`, binPath)
}

func profileScript(binPath string) string {
	return fmt.Sprintf(`#!/bin/bash
# Installed by: deckhouse-status install-motd
# Only run in interactive shells
[[ $- == *i* ]] || return 0
sudo %s 2>/dev/null || true
`, binPath)
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
