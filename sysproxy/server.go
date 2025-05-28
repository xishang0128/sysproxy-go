//go:build darwin

package sysproxy

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
)

var unixServer *http.Server

func Start() error {
	if err := startServer("/tmp/sparkle-helper.sock", StartUnix); err != nil {
		return err
	}
	return nil
}

func startServer(addr string, startFunc func(string) error) error {
	if unixServer != nil {
		_ = unixServer.Close()
		unixServer = nil
	}

	if len(addr) > 0 {
		dir := filepath.Dir(addr)
		if err := ensureDirExists(dir); err != nil {
			return err
		}

		if err := syscall.Unlink(addr); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("unlink error: %w", err)
		}

		if err := startFunc(addr); err != nil {
			return err
		}
	}
	return nil
}

func ensureDirExists(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("directory creation error: %w", err)
		}
	}
	return nil
}

func StartUnix(addr string) error {
	l, err := net.Listen("unix", addr)
	if err != nil {
		return fmt.Errorf("unix listen error: %w", err)
	}
	_ = os.Chmod(addr, 0o666)
	log.Printf("unix listening at: %s", l.Addr().String())

	server := &http.Server{
		Handler: router(),
	}
	unixServer = server
	return server.Serve(l)
}
