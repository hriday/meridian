package smtpprobe

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// fakeServer plays an SMTP server over one end of a pipe: banner, a multiline
// 250 EHLO reply built from caps, then a 221 on QUIT.
func fakeServer(server net.Conn, greeting string, caps []string) {
	defer server.Close()
	br := bufio.NewReader(server)
	fmt.Fprint(server, "220 "+greeting+" ESMTP\r\n")
	if _, err := br.ReadString('\n'); err != nil { // EHLO
		return
	}
	fmt.Fprintf(server, "250-%s\r\n", greeting)
	for i, c := range caps {
		sep := "-"
		if i == len(caps)-1 {
			sep = " "
		}
		fmt.Fprintf(server, "250%s%s\r\n", sep, c)
	}
	br.ReadString('\n') // QUIT (best-effort)
	fmt.Fprint(server, "221 bye\r\n")
}

func dialerTo(client net.Conn) DialFunc {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return client, nil
	}
}

func TestProbeParsesCapabilities(t *testing.T) {
	client, server := net.Pipe()
	go fakeServer(server, "mail.example.com", []string{"SIZE 35882577", "8BITMIME", "AUTH LOGIN PLAIN", "ENHANCEDSTATUSCODES"})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	res, err := Probe(ctx, dialerTo(client), "mail.example.com", "25", "probe.test", false)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if !strings.Contains(res.Banner, "mail.example.com") {
		t.Errorf("banner = %q", res.Banner)
	}
	if res.Greeting != "mail.example.com" {
		t.Errorf("greeting = %q, want mail.example.com", res.Greeting)
	}
	if !hasCap(res.Capabilities, "SIZE") || !hasCap(res.Capabilities, "8BITMIME") {
		t.Errorf("capabilities missing SIZE/8BITMIME: %v", res.Capabilities)
	}
	if res.StartTLS {
		t.Error("STARTTLS was not advertised, must be false")
	}
	want := []string{"LOGIN", "PLAIN"}
	if strings.Join(res.AuthMechs, ",") != strings.Join(want, ",") {
		t.Errorf("AuthMechs = %v, want %v", res.AuthMechs, want)
	}
}

func TestProbeDetectsStartTLSAdvertised(t *testing.T) {
	client, server := net.Pipe()
	// Advertise STARTTLS but the fake won't actually negotiate TLS; the probe
	// records StartTLS=true and leaves TLS nil when the handshake can't complete.
	go func() {
		defer server.Close()
		br := bufio.NewReader(server)
		fmt.Fprint(server, "220 mx.test ESMTP\r\n")
		br.ReadString('\n') // EHLO
		fmt.Fprint(server, "250-mx.test\r\n250-STARTTLS\r\n250 SIZE 10240000\r\n")
		br.ReadString('\n') // STARTTLS command
		fmt.Fprint(server, "220 go ahead\r\n")
		// then the client starts a TLS handshake we ignore; closing ends it.
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	res, err := Probe(ctx, dialerTo(client), "mx.test", "25", "probe.test", false)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if !res.StartTLS {
		t.Error("STARTTLS was advertised, want StartTLS=true")
	}
}

func TestReadReplyMultiline(t *testing.T) {
	in := "250-first\r\n250-second\r\n250 last\r\n"
	got, err := readReply(bufio.NewReader(strings.NewReader(in)))
	if err != nil {
		t.Fatalf("readReply: %v", err)
	}
	if !strings.Contains(got, "first") || !strings.Contains(got, "last") {
		t.Errorf("readReply dropped lines: %q", got)
	}
	// A single "250 x" reply must terminate immediately.
	got2, _ := readReply(bufio.NewReader(strings.NewReader("220 ready\r\nEXTRA\r\n")))
	if strings.Contains(got2, "EXTRA") {
		t.Errorf("readReply over-read past a terminating reply: %q", got2)
	}
}
