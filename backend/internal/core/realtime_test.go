package core

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestProjectFundingBroadcastsRealtimeUpdates(t *testing.T) {
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
		Name:        "Realtime Client",
		CompanyName: "Realtime Co",
		Email:       "client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)
	api := httptest.NewServer(server.Routes())
	t.Cleanup(api.Close)

	wsBase := "ws" + strings.TrimPrefix(api.URL, "http")
	publicConn, _, err := websocket.DefaultDialer.Dial(wsBase+"/api/ws/public", nil)
	if err != nil {
		t.Fatalf("dial public websocket: %v", err)
	}
	t.Cleanup(func() { _ = publicConn.Close() })

	dashboardURL := wsBase + "/api/ws/dashboard?token=" + url.QueryEscape(auth.Token)
	dashboardConn, _, err := websocket.DefaultDialer.Dial(dashboardURL, nil)
	if err != nil {
		t.Fatalf("dial dashboard websocket: %v", err)
	}
	t.Cleanup(func() { _ = dashboardConn.Close() })

	body, err := json.Marshal(CreateProjectRequest{
		Title:            "Realtime project",
		ClientName:       auth.User.Name,
		ClientEmail:      auth.User.Email,
		Brief:            "Create a funded realtime project.",
		BudgetCents:      200000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, api.URL+"/api/projects", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", auth.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create project status = %d", resp.StatusCode)
	}

	_ = publicConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, publicPayload, err := publicConn.ReadMessage()
	if err != nil {
		t.Fatalf("read public websocket message: %v", err)
	}
	var publicEvent map[string]any
	if err := json.Unmarshal(publicPayload, &publicEvent); err != nil {
		t.Fatalf("decode public websocket message: %v", err)
	}
	if publicEvent["type"] != "project-funded" {
		t.Fatalf("public event type = %#v", publicEvent["type"])
	}
	marketplace, ok := publicEvent["marketplace"].(map[string]any)
	if !ok {
		t.Fatalf("public marketplace payload missing: %#v", publicEvent)
	}
	projects, ok := marketplace["projects"].([]any)
	if !ok || len(projects) != 1 {
		t.Fatalf("public project list = %#v", marketplace["projects"])
	}
	firstProject, ok := projects[0].(map[string]any)
	if !ok {
		t.Fatalf("public project row = %#v", projects[0])
	}
	if _, ok := firstProject["client_email"]; ok {
		t.Fatalf("public realtime payload exposed client_email: %#v", firstProject)
	}

	_ = dashboardConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, dashboardPayload, err := dashboardConn.ReadMessage()
	if err != nil {
		t.Fatalf("read dashboard websocket message: %v", err)
	}
	var dashboardEvent map[string]any
	if err := json.Unmarshal(dashboardPayload, &dashboardEvent); err != nil {
		t.Fatalf("decode dashboard websocket message: %v", err)
	}
	if dashboardEvent["type"] != "project-funded" {
		t.Fatalf("dashboard event type = %#v", dashboardEvent["type"])
	}
	project, ok := dashboardEvent["project"].(map[string]any)
	if !ok {
		t.Fatalf("dashboard project payload missing: %#v", dashboardEvent)
	}
	if project["client_email"] != auth.User.Email {
		t.Fatalf("dashboard payload missing client_email: %#v", project)
	}
}