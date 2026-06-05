package core

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

	handshake := websocketHandshake(parsed.Host)
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
	if ready["protocol_version"] != "mergeos.event.v1" || ready["kind"] != "connection" {
		t.Fatalf("ready event missing protocol metadata: %#v", ready)
	}

	var snapshot map[string]interface{}
	if err := json.Unmarshal(readWebSocketTextFrame(t, reader), &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot["type"] != "live_feed_snapshot" {
		t.Fatalf("unexpected snapshot event: %#v", snapshot)
	}
	if snapshot["protocol_version"] != "mergeos.event.v1" || snapshot["kind"] != "snapshot" {
		t.Fatalf("snapshot event missing protocol metadata: %#v", snapshot)
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
	events, ok := snapshot["events"].(map[string]interface{})
	if !ok {
		t.Fatalf("snapshot missing protocol events: %#v", snapshot)
	}
	eventRows, ok := events["events"].([]interface{})
	if !ok || len(eventRows) == 0 {
		t.Fatalf("snapshot missing protocol event rows: %#v", snapshot)
	}
}

func TestWebSocketBroadcastsSanitizedTaskAcceptedFeed(t *testing.T) {
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
	workerAuth, err := store.AuthenticateGitHub(GitHubAuthProfile{
		ID:        "3001",
		Username:  "realtime-worker",
		Name:      "Realtime Worker",
		Email:     "realtime-worker@example.com",
		AvatarURL: "https://avatars.githubusercontent.com/u/3001",
	}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	clientAuth, err := store.Register(RegisterRequest{
		Name:        "Realtime Claim Client",
		CompanyName: "Realtime Claim Co",
		Email:       "realtime-claim-client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), clientAuth.User.ID, CreateProjectRequest{
		Title:            "Realtime claim feed",
		ClientName:       "Realtime Claim Client",
		CompanyName:      "Realtime Claim Co",
		ClientEmail:      "realtime-claim-client@example.com",
		Brief:            "Create a task accepted event without leaking private customer data.",
		BudgetCents:      150000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	var humanTask *Task
	for _, task := range project.Tasks {
		if task.RequiredWorkerKind == WorkerHuman {
			humanTask = task
			break
		}
	}
	if humanTask == nil {
		t.Fatal("project did not create a human task")
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

	handshake := websocketHandshake(parsed.Host)
	if _, err := conn.Write([]byte(handshake)); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(conn)
	if status, err := reader.ReadString('\n'); err != nil || !strings.Contains(status, "101 Switching Protocols") {
		t.Fatalf("websocket status = %q, err = %v", status, err)
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
	_ = readWebSocketTextFrame(t, reader)
	_ = readWebSocketTextFrame(t, reader)

	claimID := marketplaceBountyID(project.ID, humanTask.IssueNumber)
	req, err := http.NewRequest(http.MethodPost, httpServer.URL+"/api/tasks/"+claimID+"/accept", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+workerAuth.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("claim status = %d, body = %s", resp.StatusCode, string(body))
	}

	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	eventBytes := readWebSocketTextFrame(t, reader)
	for _, value := range []string{"realtime-claim-client@example.com", defaultDevPaymentCode, tempDir} {
		if strings.Contains(string(eventBytes), value) {
			t.Fatalf("task accepted websocket leaked private value %q: %s", value, string(eventBytes))
		}
	}
	var event map[string]interface{}
	if err := json.Unmarshal(eventBytes, &event); err != nil {
		t.Fatal(err)
	}
	if event["type"] != "task_accepted" {
		t.Fatalf("unexpected websocket event: %#v", event)
	}
	if event["protocol_type"] != "task.claimed" {
		t.Fatalf("task accepted websocket missing protocol type: %#v", event)
	}
	protocolEvent, ok := event["event"].(map[string]interface{})
	if !ok || protocolEvent["type"] != "task.claimed" || protocolEvent["protocol_version"] != "mergeos.event.v1" {
		t.Fatalf("task accepted websocket missing protocol event: %#v", event)
	}
	feed, ok := event["feed"].(map[string]interface{})
	if !ok {
		t.Fatalf("task accepted event missing feed: %#v", event)
	}
	stats, ok := feed["stats"].(map[string]interface{})
	acceptedCount, countOK := stats["accepted_task_count"].(float64)
	if !ok || !countOK || int(acceptedCount) != 1 {
		t.Fatalf("task accepted feed missing accepted count: %#v", event)
	}
}

func TestWebSocketBroadcastsProposalProtocolEvents(t *testing.T) {
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
	workerAuth, err := store.AuthenticateGitHub(GitHubAuthProfile{
		ID:        "3101",
		Username:  "proposal-realtime-worker",
		Name:      "Proposal Realtime Worker",
		Email:     "proposal-realtime-worker@example.com",
		AvatarURL: "https://avatars.githubusercontent.com/u/3101",
	}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	clientAuth, err := store.Register(RegisterRequest{
		Name:        "Realtime Proposal Client",
		CompanyName: "Realtime Proposal Co",
		Email:       "realtime-proposal-client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), clientAuth.User.ID, CreateProjectRequest{
		Title:            "Realtime proposal feed",
		ClientName:       "Realtime Proposal Client",
		CompanyName:      "Realtime Proposal Co",
		ClientEmail:      "realtime-proposal-client@example.com",
		Brief:            "Create proposal websocket coverage without leaking private proposal text.",
		BudgetCents:      150000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	var humanTask *Task
	for _, task := range project.Tasks {
		if task.RequiredWorkerKind == WorkerHuman {
			humanTask = task
			break
		}
	}
	if humanTask == nil {
		t.Fatal("project did not create a human task")
	}
	publicTaskID := marketplaceBountyID(project.ID, humanTask.IssueNumber)

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

	if _, err := conn.Write([]byte(websocketHandshake(parsed.Host))); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(conn)
	if status, err := reader.ReadString('\n'); err != nil || !strings.Contains(status, "101 Switching Protocols") {
		t.Fatalf("websocket status = %q, err = %v", status, err)
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
	_ = readWebSocketTextFrame(t, reader)
	_ = readWebSocketTextFrame(t, reader)

	privateCover := "I can deliver this with private staging notes and acceptance evidence."
	body := fmt.Sprintf(`{"task_id":%q,"cover_letter":%q,"bid_cents":12345,"estimated_hours":8,"availability":"This week"}`, publicTaskID, privateCover)
	req, err := http.NewRequest(http.MethodPost, httpServer.URL+"/api/proposals", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+workerAuth.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("proposal status = %d, body = %s", resp.StatusCode, string(responseBody))
	}
	var created CreateProposalResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	submittedBytes, submitted := readWebSocketEventOfType(t, reader, "proposal_created", 4)
	for _, value := range []string{"realtime-proposal-client@example.com", defaultDevPaymentCode, tempDir, privateCover, humanTask.ID} {
		if strings.Contains(string(submittedBytes), value) {
			t.Fatalf("proposal websocket leaked private value %q: %s", value, string(submittedBytes))
		}
	}
	if submitted["protocol_type"] != "proposal.submitted" {
		t.Fatalf("proposal created websocket missing proposal protocol type: %#v", submitted)
	}
	event, ok := submitted["event"].(map[string]interface{})
	if !ok || event["type"] != "proposal.submitted" || event["task_id"] != publicTaskID {
		t.Fatalf("proposal created websocket missing protocol event: %#v", submitted)
	}
	feed, ok := submitted["feed"].(map[string]interface{})
	stats, statsOK := feed["stats"].(map[string]interface{})
	proposalCount, countOK := stats["proposal_count"].(float64)
	if !ok || !statsOK || !countOK || int(proposalCount) != 1 {
		t.Fatalf("proposal created websocket missing live proposal count: %#v", submitted)
	}

	decisionReq, err := http.NewRequest(http.MethodPost, httpServer.URL+"/api/proposals/"+created.Proposal.ID+"/decision", strings.NewReader(`{"decision":"accepted"}`))
	if err != nil {
		t.Fatal(err)
	}
	decisionReq.Header.Set("Authorization", "Bearer "+clientAuth.Token)
	decisionReq.Header.Set("Content-Type", "application/json")
	decisionResp, err := http.DefaultClient.Do(decisionReq)
	if err != nil {
		t.Fatal(err)
	}
	defer decisionResp.Body.Close()
	if decisionResp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(decisionResp.Body)
		t.Fatalf("proposal decision status = %d, body = %s", decisionResp.StatusCode, string(responseBody))
	}

	decidedBytes, decided := readWebSocketEventOfType(t, reader, "proposal_decided", 6)
	if strings.Contains(string(decidedBytes), privateCover) || strings.Contains(string(decidedBytes), humanTask.ID) {
		t.Fatalf("proposal decision websocket leaked private data: %s", string(decidedBytes))
	}
	if decided["protocol_type"] != "proposal.accepted" {
		t.Fatalf("proposal decision websocket missing proposal protocol type: %#v", decided)
	}
	decisionEvent, ok := decided["event"].(map[string]interface{})
	if !ok || decisionEvent["type"] != "proposal.accepted" || decisionEvent["task_id"] != publicTaskID {
		t.Fatalf("proposal decision websocket missing protocol event: %#v", decided)
	}
}

func TestWebSocketBroadcastsAdminManualCreditLedgerEvent(t *testing.T) {
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
		AdminAutoPromote:  true,
	}
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	adminAuth, err := store.Register(RegisterRequest{
		Name:     "Realtime Admin",
		Email:    "realtime-admin@example.com",
		Password: "password123",
	})
	if err != nil {
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

	if _, err := conn.Write([]byte(websocketHandshake(parsed.Host))); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(conn)
	if status, err := reader.ReadString('\n'); err != nil || !strings.Contains(status, "101 Switching Protocols") {
		t.Fatalf("websocket status = %q, err = %v", status, err)
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
	_ = readWebSocketTextFrame(t, reader)
	_ = readWebSocketTextFrame(t, reader)

	body := strings.NewReader(`{"worker_id":"github:realtime-reviewer","reward_mrg":50,"bounty_type":"future-small","pr_url":"https://github.com/mergeos-bounties/mergeos/pull/777","pr_title":"Realtime ledger proof"}`)
	req, err := http.NewRequest(http.MethodPost, httpServer.URL+"/api/admin/ledger/credits", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+adminAuth.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("manual credit status = %d, body = %s", resp.StatusCode, string(responseBody))
	}

	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	eventBytes := readWebSocketTextFrame(t, reader)
	for _, value := range []string{"realtime-admin@example.com", defaultDevPaymentCode, tempDir} {
		if strings.Contains(string(eventBytes), value) {
			t.Fatalf("manual credit websocket leaked private value %q: %s", value, string(eventBytes))
		}
	}
	var event map[string]interface{}
	if err := json.Unmarshal(eventBytes, &event); err != nil {
		t.Fatal(err)
	}
	if event["type"] != "ledger_manual_credit" {
		t.Fatalf("unexpected websocket event: %#v", event)
	}
	if event["protocol_type"] != "ledger.recorded" {
		t.Fatalf("manual credit websocket missing protocol type: %#v", event)
	}
	protocolEvent, ok := event["event"].(map[string]interface{})
	if !ok || protocolEvent["type"] != "ledger.recorded" || protocolEvent["protocol_version"] != "mergeos.event.v1" {
		t.Fatalf("manual credit websocket missing protocol event: %#v", event)
	}
	payload, ok := protocolEvent["payload"].(map[string]interface{})
	if !ok || payload["ledger_sequence"] == nil || payload["entry_hash"] == "" {
		t.Fatalf("manual credit websocket missing ledger proof payload: %#v", event)
	}
}

func TestWebSocketBroadcastsSanitizedAdminOpsUpdateOnDispute(t *testing.T) {
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
	clientAuth, err := store.Register(RegisterRequest{
		Name:        "Realtime Dispute Client",
		CompanyName: "Realtime Dispute Co",
		Email:       "realtime-dispute-client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), clientAuth.User.ID, CreateProjectRequest{
		Title:            "Realtime dispute queue",
		ClientName:       "Realtime Dispute Client",
		CompanyName:      "Realtime Dispute Co",
		ClientEmail:      "realtime-dispute-client@example.com",
		Phone:            "+1 555 0177",
		Brief:            "Create dispute websocket coverage without leaking private data.",
		BudgetCents:      150000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
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

	if _, err := conn.Write([]byte(websocketHandshake(parsed.Host))); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(conn)
	if status, err := reader.ReadString('\n'); err != nil || !strings.Contains(status, "101 Switching Protocols") {
		t.Fatalf("websocket status = %q, err = %v", status, err)
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
	_ = readWebSocketTextFrame(t, reader)
	_ = readWebSocketTextFrame(t, reader)

	body := strings.NewReader(`{"project_id":"` + project.ID + `","severity":"critical","subject":"Escalate payout evidence","body":"Please review the private acceptance evidence."}`)
	req, err := http.NewRequest(http.MethodPost, httpServer.URL+"/api/disputes", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+clientAuth.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("dispute status = %d, body = %s", resp.StatusCode, string(responseBody))
	}

	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	eventBytes := readWebSocketTextFrame(t, reader)
	for _, value := range []string{"realtime-dispute-client@example.com", "+1 555 0177", defaultDevPaymentCode, tempDir, "private acceptance evidence"} {
		if strings.Contains(string(eventBytes), value) {
			t.Fatalf("admin ops websocket leaked private value %q: %s", value, string(eventBytes))
		}
	}
	var event map[string]interface{}
	if err := json.Unmarshal(eventBytes, &event); err != nil {
		t.Fatal(err)
	}
	if event["type"] != "admin_ops_updated" || event["kind"] != "admin_ops_signal" || event["protocol_version"] != "mergeos.event.v1" {
		t.Fatalf("unexpected admin ops websocket event: %#v", event)
	}
	if event["feed"] != nil || event["event"] != nil {
		t.Fatalf("admin ops websocket event should not include public feed or protocol event details: %#v", event)
	}
}

func websocketHandshake(host string) string {
	key := "dGhlIHNhbXBs" + "ZSBub25jZQ=="
	return fmt.Sprintf("GET /api/ws HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n", host, key)
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

func readWebSocketEventOfType(t *testing.T, reader *bufio.Reader, eventType string, maxFrames int) ([]byte, map[string]interface{}) {
	t.Helper()
	for i := 0; i < maxFrames; i++ {
		eventBytes := readWebSocketTextFrame(t, reader)
		var event map[string]interface{}
		if err := json.Unmarshal(eventBytes, &event); err != nil {
			t.Fatalf("invalid websocket JSON frame %q: %v", string(eventBytes), err)
		}
		if event["type"] == eventType {
			return eventBytes, event
		}
	}
	t.Fatalf("websocket did not receive event type %q within %d frames", eventType, maxFrames)
	return nil, nil
}
