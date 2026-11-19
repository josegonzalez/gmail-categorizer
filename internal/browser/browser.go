// Package browser opens URLs in the user's default web browser.
package browser

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Open opens url in the default browser without waiting for it to exit.
func Open(url string) error {
	name, args, err := command(runtime.GOOS, url)
	if err != nil {
		return err
	}
	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("opening browser: %w", err)
	}
	// Reap the launcher process in the background.
	go func() { _ = cmd.Wait() }()
	return nil
}

// command returns the program and arguments that open url on goos.
func command(goos, url string) (string, []string, error) {
	switch goos {
	case "darwin":
		return "open", []string{url}, nil
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}, nil
	case "linux", "freebsd", "openbsd", "netbsd", "dragonfly":
		return "xdg-open", []string{url}, nil
	default:
		return "", nil, fmt.Errorf("opening a browser is not supported on %s", goos)
	}
}
