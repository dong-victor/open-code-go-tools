//go:build !windows && !darwin

package main

import "log"

func startGlobalHotkey(modifiers, key string, callback func()) error {
	log.Printf("[hotkey] global hotkey not supported on this platform")
	return nil
}

func stopGlobalHotkey() {}

func setAutoStart(enabled bool) error {
	log.Printf("[autostart] auto-start not supported on this platform")
	return nil
}

func hideWindowNative() {}
