package plugin

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
)

// SocketEnv is set by the Docker CLI to the address of a socket the plugin
// should connect to. The CLI closes it to ask the plugin to terminate, instead
// of forwarding signals.
const SocketEnv = "DOCKER_CLI_PLUGIN_SOCKET"

// TerminationContext returns a context canceled when the plugin is asked to
// terminate: by SIGTERM or SIGHUP, or by the Docker CLI closing the plugin
// socket.
//
// SIGINT is ignored: when it comes from the terminal, the child processes in
// our process group receive it too, and the plugin must survive to clean up.
func TerminationContext(parent context.Context) (context.Context, context.CancelFunc, error) {
	ctx, cancel := context.WithCancel(parent)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for {
			select {
			case sig := <-signals:
				if sig != os.Interrupt {
					cancel()
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	stop := func() {
		signal.Stop(signals)
		cancel()
	}
	if err := watchSocket(os.Getenv(SocketEnv), cancel); err != nil {
		stop()
		return nil, nil, err
	}
	return ctx, stop, nil
}

// watchSocket connects to the Docker CLI plugin socket, if any, and calls
// onClose once the CLI closes it.
func watchSocket(addr string, onClose func()) error {
	if addr == "" {
		return nil
	}
	conn, err := net.Dial("unix", addr)
	if err != nil {
		return fmt.Errorf("connect to docker cli plugin socket: %w", err)
	}
	go func() {
		defer conn.Close()
		buf := make([]byte, 1)
		for {
			if _, err := conn.Read(buf); err != nil {
				onClose()
				return
			}
		}
	}()
	return nil
}
