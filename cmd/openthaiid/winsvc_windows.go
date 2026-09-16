//go:build windows

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/night-sornram/openthaiid/internal/desktop"
	"golang.org/x/sys/windows/svc"
)

func maybeRunWindowsService() bool {
	is, err := svc.IsWindowsService()
	if err != nil || !is {
		return false
	}
	if err := svc.Run(desktop.WindowsServiceName, &windowsHelperService{}); err != nil {
		fmt.Fprintf(os.Stderr, "service: %v\n", err)
		os.Exit(1)
	}
	return true
}

type windowsHelperService struct{}

func (m *windowsHelperService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	changes <- svc.Status{State: svc.StartPending}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		runHelperCtx(ctx, []string{"serve"})
	}()
	changes <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				cancel()
				<-done
				return false, 0
			default:
				changes <- c.CurrentStatus
			}
		case <-done:
			return false, 0
		}
	}
}
