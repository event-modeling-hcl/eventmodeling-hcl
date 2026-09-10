package serve

import (
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
)

// environment collects every external/OS interaction start makes, so
// production wiring (defaultEnvironment) and tests (a fake environment) can
// each supply their own without start knowing the difference. Nothing in
// start should reach past this struct for the outside world.
type environment struct {
	// readFile reads the model file's contents. Default: os.ReadFile.
	readFile func(path string) ([]byte, error)
	// listen creates the TCP listener the HTTP server serves on.
	// Default: net.Listen.
	listen func(network, address string) (net.Listener, error)
	// openBrowser opens url in the user's default browser. Failures are
	// not reported back to start; the production default already treats
	// them as best-effort. Default: openBrowser.
	openBrowser func(url string)
	// stdout and stderr receive all of start's console output, in place
	// of writing to os.Stdout/os.Stderr directly.
	stdout io.Writer
	stderr io.Writer
	// signals returns the channel that carries a shutdown request (SIGINT
	// or SIGTERM in production) and a stop function to release it. Default
	// wires up the real OS signal channel via signal.Notify.
	signals func() (<-chan os.Signal, func())
}

// defaultEnvironment wires environment to the real OS: real files, a real
// TCP listener, the real browser-opening command, real stdout/stderr, and
// real OS signals. This is what Start uses; tests supply their own
// environment instead.
func defaultEnvironment() environment {
	return environment{
		readFile:    os.ReadFile,
		listen:      net.Listen,
		openBrowser: openBrowser,
		stdout:      os.Stdout,
		stderr:      os.Stderr,
		signals: func() (<-chan os.Signal, func()) {
			ch := make(chan os.Signal, 1)
			signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
			return ch, func() { signal.Stop(ch) }
		},
	}
}
