//go:build windows

package desktop

import (
	"context"
	"syscall"
	"unsafe"
)

var (
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	user32                       = syscall.NewLazyDLL("user32.dll")
	shell32                      = syscall.NewLazyDLL("shell32.dll")
	procGetModuleHandleW         = kernel32.NewProc("GetModuleHandleW")
	procGetConsoleWindow         = kernel32.NewProc("GetConsoleWindow")
	procShowWindow               = user32.NewProc("ShowWindow")
	procRegisterClassExW         = user32.NewProc("RegisterClassExW")
	procCreateWindowExW          = user32.NewProc("CreateWindowExW")
	procDefWindowProcW           = user32.NewProc("DefWindowProcW")
	procGetMessageW              = user32.NewProc("GetMessageW")
	procTranslateMessage         = user32.NewProc("TranslateMessage")
	procDispatchMessageW         = user32.NewProc("DispatchMessageW")
	procPostQuitMessage          = user32.NewProc("PostQuitMessage")
	procPostMessageW             = user32.NewProc("PostMessageW")
	procDestroyWindow            = user32.NewProc("DestroyWindow")
	procLoadCursorW              = user32.NewProc("LoadCursorW")
	procCreatePopupMenu          = user32.NewProc("CreatePopupMenu")
	procAppendMenuW              = user32.NewProc("AppendMenuW")
	procTrackPopupMenu           = user32.NewProc("TrackPopupMenu")
	procDestroyMenu              = user32.NewProc("DestroyMenu")
	procSetForeground            = user32.NewProc("SetForegroundWindow")
	procGetCursorPos             = user32.NewProc("GetCursorPos")
	procMessageBoxW              = user32.NewProc("MessageBoxW")
	procShellNotifyIconW         = shell32.NewProc("Shell_NotifyIconW")
	procCreateIconFromResourceEx = user32.NewProc("CreateIconFromResourceEx")
	procLoadIconW                = user32.NewProc("LoadIconW")
	procDestroyIcon              = user32.NewProc("DestroyIcon")
)

const (
	wmDestroy     = 0x0002
	wmClose       = 0x0010
	wmCommand     = 0x0111
	wmContextMenu = 0x007B
	wmApp         = 0x8000
	wmTrayIcon    = wmApp + 1
	wmQuitTray    = wmApp + 2
	wmLButtonUp   = 0x0202
	wmRButtonUp   = 0x0205
	wmLButtonDbl  = 0x0203
	wmNull        = 0x0000

	nimAdd     = 0
	nimModify  = 1
	nimDelete  = 2
	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004

	mfString    = 0x0000
	mfSeparator = 0x0800
	mfChecked   = 0x0008
	mfUnchecked = 0x0000

	tpmRightButton = 0x0002
	tpmBottomAlign = 0x0020
	tpmRightAlign  = 0x0008

	idiApplication = 32512
	swHide         = 0
	mbOK           = 0x00000000
	mbIconInfo     = 0x00000040
	mbTopmost      = 0x00040000
	cwUseDefault   = 0x80000000
	wsExAppWindow  = 0x00040000
	wsPopup        = 0x80000000
	lrDefaultColor = 0x0000
	idReady        = 1001
	idTech         = 1002
	idStatus       = 1003
	idAutostart    = 1004
	idQuit         = 1005
	idUpdate       = 1006
	idReleases     = 1007
	iconVer        = 0x00030000
)

type wndClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type point struct {
	x, y int32
}

type notifyIconData struct {
	cbSize           uint32
	hWnd             uintptr
	uID              uint32
	uFlags           uint32
	uCallbackMessage uint32
	hIcon            uintptr
	szTip            [128]uint16
}

func HideConsole() {
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd != 0 {
		_, _, _ = procShowWindow.Call(hwnd, uintptr(swHide))
	}
}

func runNativeTray(ctx context.Context, host Host) error {
	className, err := syscall.UTF16PtrFromString("OpenThaiIDHelperTray")
	if err != nil {
		return err
	}
	hInstance, _, _ := procGetModuleHandleW.Call(0)
	hCursor, _, _ := procLoadCursorW.Call(0, uintptr(32512)) // IDC_ARROW = 32512... actually IDC_ARROW is 32512, IDI_APPLICATION is also 32512. IDC_ARROW = 32512.

	hIcon := loadTrayIcon()
	ownIcon := hIcon != 0
	if hIcon == 0 {
		hIcon, _, _ = procLoadIconW.Call(0, uintptr(idiApplication))
		ownIcon = false
	}

	var wndProc uintptr
	wndProc = syscall.NewCallback(func(hwnd, message, wParam, lParam uintptr) uintptr {
		switch message {
		case wmTrayIcon:
			switch lParam {
			case wmLButtonUp, wmLButtonDbl:
				host.openReady()
			case wmRButtonUp, wmContextMenu:
				showTrayMenu(hwnd, host)
			}
			return 0
		case wmCommand:
			switch wParam & 0xffff {
			case idReady:
				host.openReady()
			case idTech:
				host.openTechnician()
			case idStatus:
				showStatusDialog(host)
			case idAutostart:
				host.toggleAutostart()
			case idUpdate:
				host.checkUpdate()
			case idReleases:
				host.openReleases()
			case idQuit:
				host.Quit()
				_, _, _ = procPostMessageW.Call(hwnd, wmQuitTray, 0, 0)
			}
			return 0
		case wmQuitTray, wmClose:
			deleteNotifyIcon(hwnd)
			_, _, _ = procDestroyWindow.Call(hwnd)
			return 0
		case wmDestroy:
			_, _, _ = procPostQuitMessage.Call(0)
			return 0
		}
		r, _, _ := procDefWindowProcW.Call(hwnd, message, wParam, lParam)
		return r
	})

	wc := wndClassEx{
		lpfnWndProc:   wndProc,
		hInstance:     hInstance,
		hCursor:       hCursor,
		hIcon:         hIcon,
		lpszClassName: className,
	}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	if atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); atom == 0 {
		return err
	}

	hwndMessage := ^uintptr(2) // HWND_MESSAGE (-3)
	hwnd, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(className)),
		0,
		0, 0, 0, 0,
		hwndMessage, 0, hInstance, 0,
	)
	if hwnd == 0 {
		return err
	}

	if err := addNotifyIcon(hwnd, hIcon); err != nil {
		_, _, _ = procDestroyWindow.Call(hwnd)
		return err
	}

	go func() {
		<-ctx.Done()
		_, _, _ = procPostMessageW.Call(hwnd, wmQuitTray, 0, 0)
	}()

	var m msg
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		_, _, _ = procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		_, _, _ = procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	if ownIcon && hIcon != 0 {
		_, _, _ = procDestroyIcon.Call(hIcon)
	}
	return nil
}

func loadTrayIcon() uintptr {
	ico := IconICO()
	if len(ico) < 22 {
		return 0
	}
	dib := ico[22:]
	h, _, _ := procCreateIconFromResourceEx.Call(
		uintptr(unsafe.Pointer(&dib[0])),
		uintptr(len(dib)),
		1,
		uintptr(iconVer),
		0, 0,
		uintptr(lrDefaultColor),
	)
	return h
}

func addNotifyIcon(hwnd, hIcon uintptr) error {
	var nid notifyIconData
	nid.cbSize = uint32(unsafe.Sizeof(nid))
	nid.hWnd = hwnd
	nid.uID = 1
	nid.uFlags = nifMessage | nifIcon | nifTip
	nid.uCallbackMessage = wmTrayIcon
	nid.hIcon = hIcon
	tip, _ := syscall.UTF16FromString(StaffCopy(StaffLang()).Tooltip)
	copy(nid.szTip[:], tip)
	r, _, err := procShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))
	if r == 0 {
		return err
	}
	return nil
}

func deleteNotifyIcon(hwnd uintptr) {
	var nid notifyIconData
	nid.cbSize = uint32(unsafe.Sizeof(nid))
	nid.hWnd = hwnd
	nid.uID = 1
	_, _, _ = procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
}

func showTrayMenu(hwnd uintptr, host Host) {
	hMenu, _, _ := procCreatePopupMenu.Call()
	if hMenu == 0 {
		return
	}
	defer procDestroyMenu.Call(hMenu)
	c := StaffCopy(StaffLang())
	appendMenu(hMenu, mfString, idReady, c.MenuReady)
	appendMenu(hMenu, mfString, idTech, c.MenuTechnician)
	appendMenu(hMenu, mfString, idStatus, c.MenuStatus)
	appendMenu(hMenu, mfSeparator, 0, "")
	autoFlags := uintptr(mfString)
	if host.AutostartOn() {
		autoFlags |= mfChecked
	}
	appendMenu(hMenu, autoFlags, idAutostart, c.MenuAutostart)
	appendMenu(hMenu, mfSeparator, 0, "")
	appendMenu(hMenu, mfString, idUpdate, c.MenuUpdate)
	appendMenu(hMenu, mfString, idReleases, c.MenuReleases)
	appendMenu(hMenu, mfSeparator, 0, "")
	appendMenu(hMenu, mfString, idQuit, c.MenuQuit)

	var pt point
	_, _, _ = procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	_, _, _ = procSetForeground.Call(hwnd)
	_, _, _ = procTrackPopupMenu.Call(
		hMenu,
		uintptr(tpmRightButton|tpmBottomAlign|tpmRightAlign),
		uintptr(pt.x),
		uintptr(pt.y),
		0,
		hwnd,
		0,
	)
	_, _, _ = procPostMessageW.Call(hwnd, wmNull, 0, 0)
}

func appendMenu(hMenu uintptr, flags uintptr, id uint16, title string) {
	var p *uint16
	if title != "" {
		p, _ = syscall.UTF16PtrFromString(title)
	}
	_, _, _ = procAppendMenuW.Call(hMenu, flags, uintptr(id), uintptr(unsafe.Pointer(p)))
}

func showStatusDialog(host Host) {
	showAlert(StaffCopy(StaffLang()).StatusTitle, host.Status())
}

func showAlert(title, body string) {
	t, _ := syscall.UTF16PtrFromString(title)
	b, _ := syscall.UTF16PtrFromString(body)
	_, _, _ = procMessageBoxW.Call(0, uintptr(unsafe.Pointer(b)), uintptr(unsafe.Pointer(t)), uintptr(mbOK|mbIconInfo|mbTopmost))
}
