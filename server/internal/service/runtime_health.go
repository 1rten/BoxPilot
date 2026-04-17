package service

import (
	"context"
	"net"
	"os"
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

	// runtimeHealthWaitStep determines the poll interval while waiting for
	// sing-box to bind HTTP/SOCKS inbounds after restart.
	runtimeHealthWaitStep = 300 * time.Millisecond

	// defaultRuntimeHealthMaxWait is used when BOXPILOT_RUNTIME_LISTENER_READY_MAX_MS
	// is unset. sing-box may download large rule-sets before binding ports (120 × 300ms = 36s).
	defaultRuntimeHealthMaxWait = 120 * 300 * time.Millisecond
)

func runtimeHealthMaxWait(overrideMs int) time.Duration {
	if overrideMs >= 5000 && overrideMs <= 300000 {
		return time.Duration(overrideMs) * time.Millisecond
	}
	// 120 × 300ms = 36s by default; override for slow disks (e.g. BOXPILOT_RUNTIME_LISTENER_READY_MAX_MS=120000).
	const minMs = 5000
	const maxMs = 300000
	s := strings.TrimSpace(os.Getenv("BOXPILOT_RUNTIME_LISTENER_READY_MAX_MS"))
	if s == "" {
		return defaultRuntimeHealthMaxWait
	}
	ms, err := strconv.Atoi(s)
	if err != nil || ms < minMs {
		return defaultRuntimeHealthMaxWait
	}
	if ms > maxMs {
		ms = maxMs
	}
	return time.Duration(ms) * time.Millisecond
}

func runtimeHealthWaitSteps(overrideMs int) int {
	maxWait := runtimeHealthMaxWait(overrideMs)
	n := int(maxWait / runtimeHealthWaitStep)
	if n < 10 {
		return 10
	}
	return n
}

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
	return errorx.New(errorx.RTRestartFailed, "runtime listener startup timeout").WithDetails(map[string]any{
		"listener_errors": strings.Join(h.ListenerErrors, "; "),
	})
}

func WaitForRuntimeReady(ctx context.Context, httpProxy, socksProxy generator.ProxyInbound, overrideMs int) error {
	var lastHealth RuntimeHealth
	waitMax := runtimeHealthMaxWait(overrideMs)
	startedAt := time.Now()
	steps := runtimeHealthWaitSteps(overrideMs)
	for attempt := 0; attempt < steps; attempt++ {
		lastHealth = ObserveRuntimeHealth(ctx, httpProxy, socksProxy)
		if err := lastHealth.ListenerError(); err == nil {
			return nil
		}
		if attempt == steps-1 {
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
	if len(lastHealth.ListenerErrors) > 0 {
		waitedMs := int(time.Since(startedAt).Milliseconds())
		if waitedMs < 0 {
			waitedMs = 0
		}
		return errorx.New(errorx.RTRestartFailed, "runtime listener startup timeout").WithDetails(map[string]any{
			"listener_errors": strings.Join(lastHealth.ListenerErrors, "; "),
			"wait_max_ms":     int(waitMax.Milliseconds()),
			"waited_ms":       waitedMs,
			"probe_step_ms":   int(runtimeHealthWaitStep.Milliseconds()),
			"probe_attempts":  steps,
		})
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
