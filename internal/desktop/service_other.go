//go:build !windows

package desktop

func installWindowsService(string) error { return nil }

func uninstallWindowsService() error { return nil }

func windowsServiceInstalled() (bool, string, error) {
	return false, "", nil
}

// StartWindowsService is a no-op off Windows.
func StartWindowsService() error { return nil }

// StopWindowsService is a no-op off Windows.
func StopWindowsService() error { return nil }

// QueryWindowsService is unused off Windows.
func QueryWindowsService() (string, error) {
	return "Windows service: not applicable", nil
}
