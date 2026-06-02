package core

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func TestWebSocketSendsReadyAndLiveFeedSnapshot(t *testing.T) {
	tempDir := t.TempDir()
	cfg := Config{
		TokenSymbol:       defaultTokenSymbol,
		StatePath:         filepath.Join(tempDir, "state.json"),
		PlatformFeeBps:    1000,
		DevPaymentEnabled: true,
		DevPaymentCode:    defaultDevPaymentCode,
		GitHubOwner:       defaultGitHubOwner,
		BountyRoot:        filepath.Join(tempDir, "bounties"),
		SMTPFrom:          "noreply@mergeos.local",
	}
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	auth, err := store.Register(RegisterRequest{
		Name:     "Realtime Client",
		Email:    "realtime-client@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "Realtime feed proof",
		ClientName:       "Realtime Client",
		ClientEmail:      "realtime-client@example.com",
		Brief:            "Create public live feed data for websocket snapshot.",
		BudgetCents:      120000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	}); err != nil {
		t.Fatal(err)
	}

	httpServer := httptest.NewServer(NewServer(cfg, store, payments).Routes())
	defer httpServer.Close()
	parsed, err := url.Parse(httpServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := net.Dial("tcp", parsed.Host)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	handshake := fmt.Sprintf("GET /api/ws HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n\r\n", parsed.Host)
	if _, err := conn.Write([]byte(handshake)); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(conn)
	status, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(status, "101 Switching Protocols") {
		t.Fatalf("websocket status = %q", status)
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if line == "\r\n" {
			break
		}
	}

	var ready map[string]interface{}
	if err := json.Unmarshal(readWebSocketTextFrame(t, reader), &ready); err != nil {
		t.Fatal(err)
	}
	if ready["type"] != "connection_ready" || ready["status"] != "ok" || ready["token_symbol"] != defaultTokenSymbol {
		t.Fatalf("unexpected ready event: %#v", ready)
	}

	var snapshot map[string]interface{}
	if err := json.Unmarshal(readWebSocketTextFrame(t, reader), &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot["type"] != "live_feed_snapshot" {
		t.Fatalf("unexpected snapshot event: %#v", snapshot)
	}
	feed, ok := snapshot["feed"].(map[string]interface{})
	if !ok {
		t.Fatalf("snapshot missing feed: %#v", snapshot)
	}
	stats, ok := feed["stats"].(map[string]interface{})
	projectCount, countOK := stats["project_count"].(float64)
	if !ok || !countOK || int(projectCount) != 1 {
		t.Fatalf("snapshot missing live feed stats: %#v", snapshot)
	}
}

func readWebSocketTextFrame(t *testing.T, reader *bufio.Reader) []byte {
	t.Helper()
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		t.Fatal(err)
	}
	opcode := header[0] & 0x0f
	if opcode != 1 {
		t.Fatalf("websocket opcode = %d", opcode)
	}
	length := int64(header[1] & 0x7f)
	switch length {
	case 126:
		extended := make([]byte, 2)
		if _, err := io.ReadFull(reader, extended); err != nil {
			t.Fatal(err)
		}
		length = int64(uint16(extended[0])<<8 | uint16(extended[1]))
	case 127:
		extended := make([]byte, 8)
		if _, err := io.ReadFull(reader, extended); err != nil {
			t.Fatal(err)
		}
		length = 0
		for _, b := range extended {
			length = (length << 8) | int64(b)
		}
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(reader, payload); err != nil {
		t.Fatal(err)
	}
	return payload
}
