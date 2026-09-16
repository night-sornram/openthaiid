//go:build !windows && !(darwin && cgo)

package desktop

import "context"

func runNativeTray(context.Context, Host) error {
	return ErrNoTray
}

// HideConsole is a no-op off Windows.
func HideConsole() {}

func showAlert(title, body string) {
	_ = title
	_ = body
}
