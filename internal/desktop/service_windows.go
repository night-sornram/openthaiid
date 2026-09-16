//go:build windows

package desktop

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

func installWindowsService(bin string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect service manager (need Administrator): %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(WindowsServiceName)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists; uninstall first", WindowsServiceName)
	}
	cfg := mgr.Config{
		DisplayName: WindowsDisplayName,
		Description: "Local Thai National ID helper (loopback only). No card data is stored.",
		StartType:   mgr.StartAutomatic,
	}
	s, err = m.CreateService(WindowsServiceName, bin, cfg, "serve")
	if err != nil {
		return err
	}
	defer s.Close()
	_ = eventlog.InstallAsEventCreate(WindowsServiceName, eventlog.Error|eventlog.Warning|eventlog.Info)
	return nil
}

func uninstallWindowsService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect service manager (need Administrator): %w", err)
	}
	defer m.Disconnect()
	s, err := m.OpenService(WindowsServiceName)
	if err != nil {
		return nil
	}
	defer s.Close()
	_, _ = s.Control(svc.Stop)
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		st, err := s.Query()
		if err != nil || st.State == svc.Stopped {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err := s.Delete(); err != nil {
		return err
	}
	_ = eventlog.Remove(WindowsServiceName)
	return nil
}

func windowsServiceInstalled() (bool, string, error) {
	m, err := mgr.Connect()
	if err != nil {
		exe, _ := os.Executable()
		return false, filepath.Join(filepath.Dir(exe), WindowsServiceName), err
	}
	defer m.Disconnect()
	s, err := m.OpenService(WindowsServiceName)
	if err != nil {
		return false, WindowsServiceName, nil
	}
	s.Close()
	return true, WindowsServiceName, nil
}

// StartWindowsService asks SCM to start the helper.
func StartWindowsService() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(WindowsServiceName)
	if err != nil {
		return err
	}
	defer s.Close()
	return s.Start()
}

// StopWindowsService asks SCM to stop the helper.
func StopWindowsService() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(WindowsServiceName)
	if err != nil {
		return err
	}
	defer s.Close()
	_, err = s.Control(svc.Stop)
	return err
}

// QueryWindowsService returns a short status line.
func QueryWindowsService() (string, error) {
	m, err := mgr.Connect()
	if err != nil {
		return "", err
	}
	defer m.Disconnect()
	s, err := m.OpenService(WindowsServiceName)
	if err != nil {
		return "Windows service: not installed", nil
	}
	defer s.Close()
	st, err := s.Query()
	if err != nil {
		return "", err
	}
	state := "unknown"
	switch st.State {
	case svc.Stopped:
		state = "stopped"
	case svc.StartPending:
		state = "start pending"
	case svc.StopPending:
		state = "stop pending"
	case svc.Running:
		state = "running"
	}
	return "Windows service: installed (" + state + ")", nil
}
