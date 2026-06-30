//go:build darwin

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

func startGlobalHotkey(modifiers, key string, callback func()) error {
	// macOS global hotkey registration requires Carbon APIs which need cgo.
	// For now, log and skip — tray menu is the primary access method on macOS.
	log.Printf("[hotkey] macOS global hotkey not yet implemented (modifiers=%s key=%s)", modifiers, key)
	return nil
}

func stopGlobalHotkey() {}

const launchAgentPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.ocgt.panel</string>
    <key>Program</key>
    <string>{{.ExePath}}</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
</dict>
</plist>`

func setAutoStart(enabled bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}
	launchAgentsDir := filepath.Join(home, "Library", "LaunchAgents")
	plistPath := filepath.Join(launchAgentsDir, "com.ocgt.panel.plist")

	if !enabled {
		if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove plist: %w", err)
		}
		// Unload if running
		exec.Command("launchctl", "unload", plistPath).Run()
		return nil
	}

	if err := os.MkdirAll(launchAgentsDir, 0700); err != nil {
		return fmt.Errorf("mkdir LaunchAgents: %w", err)
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("executable path: %w", err)
	}

	f, err := os.Create(plistPath)
	if err != nil {
		return fmt.Errorf("create plist: %w", err)
	}
	defer f.Close()

	tmpl := template.Must(template.New("plist").Parse(launchAgentPlist))
	if err := tmpl.Execute(f, map[string]string{"ExePath": exePath}); err != nil {
		return err
	}

	// Load the agent
	exec.Command("launchctl", "unload", plistPath).Run()
	return exec.Command("launchctl", "load", plistPath).Run()
}
