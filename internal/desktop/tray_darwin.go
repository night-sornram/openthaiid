//go:build darwin && cgo

package desktop

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>
#include "tray_darwin.h"
*/
import "C"

import (
	"context"
	"sync"
	"unsafe"
)

var (
	darwinMu   sync.Mutex
	darwinHost Host
)

func runNativeTray(ctx context.Context, host Host) error {
	darwinMu.Lock()
	darwinHost = host
	darwinMu.Unlock()

	go func() {
		<-ctx.Done()
		C.openthaiidDarwinTrayQuit()
	}()

	png := IconPNG()
	var ptr *C.uchar
	if len(png) > 0 {
		ptr = (*C.uchar)(unsafe.Pointer(&png[0]))
	}
	C.openthaiidDarwinTrayRun(ptr, C.int(len(png)))
	return nil
}

//export openthaiidTrayAction
func openthaiidTrayAction(item C.int) {
	darwinMu.Lock()
	h := darwinHost
	darwinMu.Unlock()
	switch int(item) {
	case 1:
		h.openReady()
	case 2:
		h.openTechnician()
	case 3:
		title := C.CString("OpenThaiID status")
		body := C.CString(h.Status())
		C.openthaiidDarwinAlert(title, body)
		C.free(unsafe.Pointer(title))
		C.free(unsafe.Pointer(body))
	case 4:
		h.toggleAutostart()
	case 5:
		h.Quit()
		C.openthaiidDarwinTrayQuit()
	}
}

//export openthaiidTrayAutostartOn
func openthaiidTrayAutostartOn() C.int {
	darwinMu.Lock()
	h := darwinHost
	darwinMu.Unlock()
	if h.AutostartOn != nil && h.AutostartOn() {
		return 1
	}
	return 0
}
