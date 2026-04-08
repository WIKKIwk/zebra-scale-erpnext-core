package mobileapi

import (
	bridgestate "bridge/state"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type SessionProfile struct {
	Role        string `json:"role"`
	DisplayName string `json:"display_name"`
	LegalName   string `json:"legal_name"`
	Ref         string `json:"ref"`
	Phone       string `json:"phone"`
	AvatarURL   string `json:"avatar_url"`
}

type authLoginRequest struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

type Server struct {
	cfg   Config
	store *bridgestate.Store
	http  *http.Client

	mu     sync.Mutex
	tokens map[string]SessionProfile
}

func New(cfg Config) *Server {
	return &Server{
		cfg:    cfg,
		store:  bridgestate.New(cfg.BridgeStateFile),
		http:   &http.Client{Timeout: 1500 * time.Millisecond},
		tokens: make(map[string]SessionProfile),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/v1/mobile/auth/login", s.handleLogin)
	mux.HandleFunc("/v1/mobile/auth/logout", s.handleLogout)
	mux.HandleFunc("/v1/mobile/profile", s.handleProfile)
	mux.HandleFunc("/v1/mobile/monitor/state", s.handleMonitorState)
	return mux
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"service": "mobileapi",
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var req authLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid_json",
		})
		return
	}

	if strings.TrimSpace(req.Phone) != s.cfg.LoginPhone || strings.TrimSpace(req.Code) != s.cfg.LoginCode {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": "invalid_credentials",
		})
		return
	}

	token, err := generateToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "token_generation_failed",
		})
		return
	}

	s.mu.Lock()
	s.tokens[token] = s.cfg.Profile
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"token":   token,
		"profile": s.cfg.Profile,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	token := bearerToken(r.Header.Get("Authorization"))
	if token != "" {
		s.mu.Lock()
		delete(s.tokens, token)
		s.mu.Unlock()
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPut {
		writeMethodNotAllowed(w)
		return
	}

	profile, ok := s.authorize(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
		return
	}

	if r.Method == http.MethodPut {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err == nil {
			if nickname := strings.TrimSpace(asString(payload["nickname"])); nickname != "" {
				profile.DisplayName = nickname
				s.updateAuthorizedProfile(r, profile)
			}
		}
	}

	writeJSON(w, http.StatusOK, profile)
}

func (s *Server) handleMonitorState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	profile, ok := s.authorize(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
		return
	}

	snap, err := s.store.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			snap = bridgestate.Snapshot{}
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"error": err.Error(),
			})
			return
		}
	}

	printer := map[string]any{
		"ok": false,
	}
	if value, err := s.fetchPrinterTrace(); err == nil {
		printer = value
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"profile": profile,
		"state":   snap,
		"printer": printer,
	})
}

func (s *Server) fetchPrinterTrace() (map[string]any, error) {
	base := strings.TrimRight(strings.TrimSpace(s.cfg.PolygonURL), "/")
	if base == "" {
		return nil, fmt.Errorf("polygon url empty")
	}

	resp, err := s.http.Get(base + "/api/v1/dev/printer")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("polygon printer status=%d", resp.StatusCode)
	}

	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Server) authorize(r *http.Request) (SessionProfile, bool) {
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		return SessionProfile{}, false
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	profile, ok := s.tokens[token]
	return profile, ok
}

func (s *Server) updateAuthorizedProfile(r *http.Request, profile SessionProfile) {
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		return
	}
	s.mu.Lock()
	s.tokens[token] = profile
	s.mu.Unlock()
}

func generateToken() (string, error) {
	var raw [24]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return "dev-" + hex.EncodeToString(raw[:]), nil
}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func asString(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
		"error": "method_not_allowed",
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
