package service

import (
	"context"
	"net"
	"strconv"
	"strings"
	"time"

	"boxpilot/server/internal/generator"
	"boxpilot/server/internal/util/errorx"
)

const (
	runtimeHealthDialTimeout = 250 * time.Millisecond
	runtimeHealthWaitStep    = 150 * time.Millisecond
	runtimeHealthWaitSteps   = 10
)

type RuntimeHealth struct {
	ListenerErrors []string
}

func ObserveRuntimeHealth(ctx context.Context, httpProxy, socksProxy generator.ProxyInbound) RuntimeHealth {
	errors := make([]string, 0, 2)
	for _, proxy := range []generator.ProxyInbound{httpProxy, socksProxy} {
		if !proxy.Enabled {
			continue
		}
		address := listenerProbeAddress(proxy.ListenAddress, proxy.Port)
		if err := probeListener(ctx, address); err != nil {
			errors = append(errors, formatListenerError(proxy.Type, address, err))
		}
	}
	return RuntimeHealth{ListenerErrors: errors}
}

func (h RuntimeHealth) ListenerError() error {
	if len(h.ListenerErrors) == 0 {
		return nil
	}
	return errorx.New(errorx.RTRestartFailed, strings.Join(h.ListenerErrors, "; "))
}

func WaitForRuntimeReady(ctx context.Context, httpProxy, socksProxy generator.ProxyInbound) error {
	var lastErr error
	for attempt := 0; attempt < runtimeHealthWaitSteps; attempt++ {
		health := ObserveRuntimeHealth(ctx, httpProxy, socksProxy)
		if err := health.ListenerError(); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if attempt == runtimeHealthWaitSteps-1 {
			break
		}
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				return ctx.Err()
			}
		case <-time.After(runtimeHealthWaitStep):
		}
	}
	if lastErr != nil {
		return lastErr
	}
	return errorx.New(errorx.RTRestartFailed, "runtime listeners not ready")
}

func listenerProbeAddress(listenAddress string, port int) string {
	host := strings.TrimSpace(listenAddress)
	switch host {
	case "", "0.0.0.0", "::":
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, strconv.Itoa(port))
}

func probeListener(ctx context.Context, address string) error {
	dialer := &net.Dialer{Timeout: runtimeHealthDialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}

func formatListenerError(proxyType, address string, err error) string {
	label := strings.ToUpper(strings.TrimSpace(proxyType))
	if label == "" {
		label = "PROXY"
	}
	return label + " listener unreachable at " + address + ": " + err.Error()
}
