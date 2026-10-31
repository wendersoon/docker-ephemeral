package plugin

import (
	"net"
	"path/filepath"
	"testing"
	"time"
)

func TestWatchSocket_CallsBackWhenClosed(t *testing.T) {
	addr := filepath.Join(t.TempDir(), "cli.sock")
	ln, err := net.Listen("unix", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	closed := make(chan struct{})
	if err := watchSocket(addr, func() { close(closed) }); err != nil {
		t.Fatal(err)
	}

	conn, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("termination callback not called")
	}
}

func TestWatchSocket_NoSocket(t *testing.T) {
	if err := watchSocket("", func() { t.Error("unexpected callback") }); err != nil {
		t.Fatal(err)
	}
}
