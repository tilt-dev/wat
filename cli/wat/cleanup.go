package wat

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var cleanupMu = &sync.Mutex{}
var cleanupFuncs []*func()
var cleanupCh chan os.Signal

// Create a cleanup script that runs even if the process gets a SIGINT/SIGTERM
//
// Intended to be used like:
// tearDown := createCleanup(func() { ...critical cleanup... })
// defer tearDown()
//
// The critical cleanup will get executed in tearDown()
func createCleanup(f func()) (tearDown func()) {
	cleanupMu.Lock()
	defer cleanupMu.Unlock()

	fptr := &f

	cleanupFuncs = append(cleanupFuncs, fptr)
	if cleanupCh == nil {
		cleanupCh = make(chan os.Signal, 1)
		signal.Notify(cleanupCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			_, ok := <-cleanupCh
			if !ok {
				// The channel closed normally
				return
			}

			cleanupMu.Lock()
			defer cleanupMu.Unlock()

			for _, fp := range cleanupFuncs {
				f := *fp
				f()
			}
			os.Exit(1)
		}()
	}

	return func() {
		f()

		cleanupMu.Lock()
		defer cleanupMu.Unlock()

		for i, fp := range cleanupFuncs {
			if fp == fptr {
				cleanupFuncs = append(cleanupFuncs[:i], cleanupFuncs[i+1:]...)
				break
			}
		}

		if len(cleanupFuncs) == 0 {
			signal.Stop(cleanupCh)
			close(cleanupCh)
			cleanupCh = nil
		}
	}
}
