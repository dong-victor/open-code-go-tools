//go:build windows

package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

var (
	hotkeyStop  chan struct{}
	hotkeyMutex sync.Mutex
)

func startGlobalHotkey(modifiers, key string, callback func()) error {
	hotkeyMutex.Lock()
	defer hotkeyMutex.Unlock()

	if hotkeyStop != nil {
		close(hotkeyStop)
	}

	mod, err := parseModifiers(modifiers)
	if err != nil {
		return err
	}
	vk, err := parseVirtualKey(key)
	if err != nil {
		return err
	}

	hotkeyStop = make(chan struct{})
	go windowsHotkeyLoop(mod, vk, callback, hotkeyStop)
	return nil
}

func stopGlobalHotkey() {
	hotkeyMutex.Lock()
	defer hotkeyMutex.Unlock()
	if hotkeyStop != nil {
		close(hotkeyStop)
		hotkeyStop = nil
	}
}

func parseModifiers(s string) (uint32, error) {
	var mod uint32
	parts := strings.Split(s, "+")
	for _, p := range parts {
		switch strings.TrimSpace(p) {
		case "Ctrl":
			mod |= 0x0002 // MOD_CONTROL
		case "Alt":
			mod |= 0x0001 // MOD_ALT
		case "Shift":
			mod |= 0x0004 // MOD_SHIFT
		case "Win":
			mod |= 0x0008 // MOD_WIN
		default:
			return 0, fmt.Errorf("unknown modifier: %q", p)
		}
	}
	if mod == 0 {
		return 0, fmt.Errorf("no modifiers specified")
	}
	return mod, nil
}

func parseVirtualKey(s string) (uint32, error) {
	s = strings.TrimSpace(s)
	if len(s) == 1 {
		ch := s[0]
		if ch >= 'A' && ch <= 'Z' {
			return uint32(ch), nil
		}
		if ch >= '0' && ch <= '9' {
			return uint32(ch), nil
		}
	}
	var fNum int
	if _, err := fmt.Sscanf(s, "F%d", &fNum); err == nil && fNum >= 1 && fNum <= 24 {
		return uint32(0x70 + fNum - 1), nil // VK_F1 = 0x70
	}
	return 0, fmt.Errorf("invalid hotkey key: %q", s)
}

// MSG structure for PeekMessage/GetMessage
type winMsg struct {
	Hwnd    uintptr
	Message uint32
	_       uintptr
	_       uintptr
	_       uint32
	_       uint32
	_       uint32
}

func windowsHotkeyLoop(mod, vk uint32, callback func(), stop chan struct{}) {
	// CRITICAL: RegisterHotKey(NULL,...) posts WM_HOTKEY to the calling thread's
	// message queue. LockOSThread ensures the registration and message pump stay
	// on the same OS thread — Go goroutines normally migrate between threads.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	user32 := syscall.NewLazyDLL("user32.dll")
	registerHotKey := user32.NewProc("RegisterHotKey")
	peekMessage := user32.NewProc("PeekMessageW")
	unregisterHotKey := user32.NewProc("UnregisterHotKey")

	const id uintptr = 0xdead
	const pmRemove = 1

	ret, _, _ := registerHotKey.Call(0, id, uintptr(mod), uintptr(vk))
	if ret == 0 {
		log.Printf("[hotkey] RegisterHotKey failed for mod=0x%x vk=0x%x", mod, vk)
		return
	}
	log.Printf("[hotkey] registered: mod=0x%x vk=0x%x", mod, vk)

	defer func() {
		unregisterHotKey.Call(0, id)
		log.Printf("[hotkey] unregistered")
	}()

	for {
		// Check stop signal
		select {
		case <-stop:
			return
		default:
		}

		var msg winMsg
		got, _, _ := peekMessage.Call(
			uintptr(unsafe.Pointer(&msg)),
			0, 0, 0,
			pmRemove,
		)
		if got != 0 && msg.Message == 0x0312 { // WM_HOTKEY
			callback()
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func setAutoStart(enabled bool) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return fmt.Errorf("open Run key: %w", err)
	}
	defer key.Close()

	if enabled {
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("get executable path: %w", err)
		}
		return key.SetStringValue("ocgt", exePath)
	}
	if err := key.DeleteValue("ocgt"); err != nil && err != registry.ErrNotExist {
		return err
	}
	return nil
}

// hideWindowNative bypasses the Wails runtime and uses the Win32 API directly
// to find and hide our main window. This is more reliable during early startup
// when the Wails dispatch queue may not be processing yet.
func hideWindowNative() {
	user32 := syscall.NewLazyDLL("user32.dll")
	findWindow := user32.NewProc("FindWindowW")
	showWindow := user32.NewProc("ShowWindow")

	const swHide = 0

	title, _ := syscall.UTF16PtrFromString("ocgt Control Panel / 控制面板")
	hwnd, _, _ := findWindow.Call(0, uintptr(unsafe.Pointer(title)))
	if hwnd != 0 {
		showWindow.Call(hwnd, swHide)
	}
}
