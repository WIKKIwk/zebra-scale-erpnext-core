package mobileapi

import (
	bridgestate "bridge/state"
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoginAndProfile(t *testing.T) {
	t.Parallel()

	server := New(Config{
		ServerName:      "gscale-dev",
		BridgeStateFile: t.TempDir() + "/bridge_state.json",
		LoginPhone:      "998900000000",
		LoginCode:       "1234",
		Profile: SessionProfile{
			Role:        "admin",
			DisplayName: "Polygon Operator",
			LegalName:   "Polygon Operator",
			Ref:         "dev-operator",
			Phone:       "998900000000",
		},
	})

	body := bytes.NewBufferString(`{"phone":"998900000000","code":"1234"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/mobile/auth/login", body)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d", rec.Code)
	}

	var loginResp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	token, _ := loginResp["token"].(string)
	if token == "" {
		t.Fatal("token is empty")
	}

	profileReq := httptest.NewRequest(http.MethodGet, "/v1/mobile/profile", nil)
	profileReq.Header.Set("Authorization", "Bearer "+token)
	profileRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(profileRec, profileReq)

	if profileRec.Code != http.StatusOK {
		t.Fatalf("profile status = %d", profileRec.Code)
	}
	if !bytes.Contains(profileRec.Body.Bytes(), []byte(`"role":"admin"`)) {
		t.Fatalf("profile body = %s", profileRec.Body.String())
	}
}

func TestHandshakeReturnsServerIdentity(t *testing.T) {
	t.Parallel()

	server := New(Config{
		ServerName:      "gscale-dev",
		BridgeStateFile: t.TempDir() + "/bridge_state.json",
		Profile: SessionProfile{
			Role:        "admin",
			DisplayName: "Polygon Operator",
			LegalName:   "Polygon Operator",
			Ref:         "dev-operator",
			Phone:       "998900000000",
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/mobile/handshake", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("handshake status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"server_name":"gscale-dev"`)) {
		t.Fatalf("handshake body = %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"app":"gscale-zebra"`)) {
		t.Fatalf("handshake body = %s", rec.Body.String())
	}
}

func TestMonitorStateReturnsBridgeSnapshot(t *testing.T) {
	t.Parallel()

	stateFile := t.TempDir() + "/bridge_state.json"
	store := bridgestate.New(stateFile)
	weight := 1.25
	stable := true
	if err := store.Update(func(snapshot *bridgestate.Snapshot) {
		snapshot.Scale = bridgestate.ScaleSnapshot{
			Source:    "polygon",
			Port:      "polygon://scale",
			Weight:    &weight,
			Unit:      "kg",
			Stable:    &stable,
			UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
		}
	}); err != nil {
		t.Fatalf("seed bridge: %v", err)
	}

	server := New(Config{
		ServerName:      "gscale-dev",
		BridgeStateFile: stateFile,
		LoginPhone:      "998900000000",
		LoginCode:       "1234",
		Profile: SessionProfile{
			Role:        "admin",
			DisplayName: "Polygon Operator",
			LegalName:   "Polygon Operator",
			Ref:         "dev-operator",
			Phone:       "998900000000",
		},
	})

	loginReq := httptest.NewRequest(http.MethodPost, "/v1/mobile/auth/login", bytes.NewBufferString(`{"phone":"998900000000","code":"1234"}`))
	loginRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(loginRec, loginReq)

	var loginResp map[string]any
	if err := json.Unmarshal(loginRec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	token, _ := loginResp["token"].(string)

	req := httptest.NewRequest(http.MethodGet, "/v1/mobile/monitor/state", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("monitor status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"source":"polygon"`)) {
		t.Fatalf("monitor body = %s", rec.Body.String())
	}
}

func TestMonitorStreamReturnsInitialSnapshotEvent(t *testing.T) {
	t.Parallel()

	stateFile := t.TempDir() + "/bridge_state.json"
	store := bridgestate.New(stateFile)
	weight := 2.5
	stable := true
	if err := store.Update(func(snapshot *bridgestate.Snapshot) {
		snapshot.Scale = bridgestate.ScaleSnapshot{
			Source: "polygon",
			Weight: &weight,
			Unit:   "kg",
			Stable: &stable,
		}
	}); err != nil {
		t.Fatalf("seed bridge: %v", err)
	}

	server := New(Config{
		ServerName:      "gscale-dev",
		BridgeStateFile: stateFile,
		Profile: SessionProfile{
			Role:        "admin",
			DisplayName: "Polygon Operator",
			LegalName:   "Polygon Operator",
			Ref:         "dev-operator",
			Phone:       "998900000000",
		},
	})

	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/v1/mobile/monitor/stream")
	if err != nil {
		t.Fatalf("stream request: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content-type = %q", got)
	}

	reader := bufio.NewReader(resp.Body)
	foundEvent := false
	foundScale := false
	for i := 0; i < 8; i++ {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read stream: %v", err)
		}
		if bytes.Contains([]byte(line), []byte("event: snapshot")) {
			foundEvent = true
		}
		if bytes.Contains([]byte(line), []byte(`"weight":2.5`)) {
			foundScale = true
		}
		if foundEvent && foundScale {
			return
		}
	}

	t.Fatalf("stream did not include initial snapshot event")
}
