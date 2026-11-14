package sipping

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

// fakeSIPServer reads the OPTIONS request then writes a canned 200 response.
func fakeSIPServer(server net.Conn, resp string) {
	defer server.Close()
	br := bufio.NewReader(server)
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return
		}
		if strings.TrimRight(line, "\r\n") == "" {
			break // end of request headers
		}
	}
	server.Write([]byte(resp))
}

func dialTo(client net.Conn) DialFunc {
	return func(ctx context.Context, network, addr string) (net.Conn, error) { return client, nil }
}

func TestProbe200(t *testing.T) {
	resp := "SIP/2.0 200 OK\r\n" +
		"Via: SIP/2.0/UDP 10.0.0.1:5060\r\n" +
		"Server: Asterisk PBX 18.5.0\r\n" +
		"Allow: INVITE, ACK, CANCEL, OPTIONS, BYE, REFER, SUBSCRIBE, NOTIFY\r\n" +
		"Supported: replaces, timer\r\n" +
		"Accept: application/sdp\r\n" +
		"Content-Length: 0\r\n\r\n"
	client, server := net.Pipe()
	go fakeSIPServer(server, resp)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	r, err := Probe(ctx, dialTo(client), "pbx.example.com", "5060", "udp")
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if r.StatusCode != 200 || r.Reason != "OK" {
		t.Errorf("status = %d %q, want 200 OK", r.StatusCode, r.Reason)
	}
	if r.Server != "Asterisk PBX 18.5.0" {
		t.Errorf("server = %q", r.Server)
	}
	if len(r.Allow) != 8 || r.Allow[3] != "OPTIONS" {
		t.Errorf("allow parsed wrong: %v", r.Allow)
	}
	if len(r.Supported) != 2 {
		t.Errorf("supported = %v", r.Supported)
	}
	if r.Transport != "udp" || r.Target != "pbx.example.com:5060" {
		t.Errorf("meta wrong: %s %s", r.Transport, r.Target)
	}
}

func TestProbe401(t *testing.T) {
	resp := "SIP/2.0 401 Unauthorized\r\nServer: FreeSWITCH\r\nContent-Length: 0\r\n\r\n"
	client, server := net.Pipe()
	go fakeSIPServer(server, resp)
	r, err := Probe(context.Background(), dialTo(client), "sip.example.com", "5060", "udp")
	if err != nil {
		t.Fatal(err)
	}
	// A 401/200/etc. all prove the server is alive and speaking SIP.
	if r.StatusCode != 401 {
		t.Errorf("status = %d, want 401", r.StatusCode)
	}
}

func TestProbeNonSIP(t *testing.T) {
	client, server := net.Pipe()
	go fakeSIPServer(server, "HTTP/1.1 200 OK\r\n\r\n")
	if _, err := Probe(context.Background(), dialTo(client), "h", "5060", "udp"); err == nil {
		t.Error("a non-SIP response should error")
	}
}
