package handlers

import (
	"fmt"
	"net"
	"testing"
	"time"
)

func TestProbeNode_Hysteria2UDP_LocalEcho(t *testing.T) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()

	addr := pc.LocalAddr().(*net.UDPAddr)

	go func() {
		buf := make([]byte, 2048)
		for {
			_ = pc.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
			n, raddr, err := pc.ReadFrom(buf)
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					continue
				}
				return
			}
			if n > 0 && raddr != nil {
				_, _ = pc.WriteTo(buf[:n], raddr)
			}
		}
	}()

	time.Sleep(20 * time.Millisecond)

	raw := fmt.Sprintf(`{"type":"hysteria2","server":"127.0.0.1","server_port":%d,"tls":{"enabled":true}}`, addr.Port)
	lat, status, errMsg := probeNode(raw, "hysteria2", "ping", 2*time.Second)
	if status != "ok" || errMsg != "" {
		t.Fatalf("expected ok, got status=%q err=%q", status, errMsg)
	}
	if lat < 0 {
		t.Fatalf("expected non-negative latency, got %d", lat)
	}
}

func TestProbeNode_NonHysteria2_StillTCP(t *testing.T) {
	raw := `{"type":"vmess","server":"127.0.0.1","server_port":59999,"uuid":"00000000-0000-0000-0000-000000000000"}`
	_, status, errMsg := probeNode(raw, "vmess", "ping", 500*time.Millisecond)
	if status != "error" || errMsg == "" {
		t.Fatalf("expected tcp error, got status=%q err=%q", status, errMsg)
	}
}
