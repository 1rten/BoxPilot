package service

import (
	"context"
	"net"
	"strconv"
	"strings"
	"testing"

	"boxpilot/server/internal/generator"
	"boxpilot/server/internal/util/errorx"
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
	if !strings.Contains(strings.ToLower(got), "startup timeout") {
		t.Fatalf("expected timeout message, got %q", got)
	}
	appErr, ok := err.(*errorx.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T", err)
	}
	listenerDetails, _ := appErr.Details["listener_errors"].(string)
	if !(strings.Contains(strings.ToLower(listenerDetails), "http") && strings.Contains(strings.ToLower(listenerDetails), "socks")) {
		t.Fatalf("expected both listener names in details, got %q", listenerDetails)
	}
}
