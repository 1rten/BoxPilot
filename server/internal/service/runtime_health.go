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
	// runtimeHealthDialTimeout is the per-probe TCP dial timeout.
	// 500 ms gives enough headroom for loopback without masking a truly-dead
	// listener for too long.
	runtimeHealthDialTimeout = 500 * time.Millisecond

	// runtimeHealthWaitStep / runtimeHealthWaitSteps together determine the
	// maximum time we will poll before declaring the runtime unhealthy.
	// sing-box must load ruleset files (several MB) before binding ports, which
	// can take 3-10 s on constrained hardware and occasionally longer under
	// heavy I/O or cold cache conditions. 120 × 300 ms = 36 s reduces false
	// rollback on slow starts while still bounding detection time.
	runtimeHealthWaitStep  = 300 * time.Millisecond
	runtimeHealthWaitSteps = 120
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
