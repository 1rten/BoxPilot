package service

import (
	"context"
	"net"
	"strconv"
	"strings"
	"testing"

	"boxpilot/server/internal/generator"
)

func TestObserveRuntimeHealth_AcceptsWildcardListenerViaLoopback(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	_, rawPort, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("split addr: %v", err)
	}
	port, err := strconv.Atoi(rawPort)
	if err != nil {
		t.Fatalf("atoi port: %v", err)
	}

	health := ObserveRuntimeHealth(context.Background(),
		generator.ProxyInbound{Type: "http", ListenAddress: "0.0.0.0", Port: port, Enabled: true},
		generator.ProxyInbound{},
	)
	if err := health.ListenerError(); err != nil {
		t.Fatalf("expected healthy listener, got %v", err)
	}
}

func TestObserveRuntimeHealth_ReportsUnreachableEnabledListeners(t *testing.T) {
	health := ObserveRuntimeHealth(context.Background(),
		generator.ProxyInbound{Type: "http", ListenAddress: "127.0.0.1", Port: 1, Enabled: true},
		generator.ProxyInbound{Type: "socks", ListenAddress: "127.0.0.1", Port: 2, Enabled: true},
	)

	err := health.ListenerError()
	if err == nil {
		t.Fatal("expected listener health error")
	}
	got := err.Error()
	if got == "" {
		t.Fatal("expected non-empty error message")
	}
	if !(strings.Contains(strings.ToLower(got), "http") && strings.Contains(strings.ToLower(got), "socks")) {
		t.Fatalf("expected both listener names in error, got %q", got)
	}
}
