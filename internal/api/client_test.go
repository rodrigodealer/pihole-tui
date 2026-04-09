package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient("http://localhost", "secret")
	if c.BaseURL != "http://localhost" {
		t.Errorf("expected base URL http://localhost, got %s", c.BaseURL)
	}
	if c.Password != "secret" {
		t.Errorf("expected password secret, got %s", c.Password)
	}
}

func TestNewClientTrimsTrailingSlash(t *testing.T) {
	c := NewClient("http://localhost/", "secret")
	if c.BaseURL != "http://localhost" {
		t.Errorf("expected trailing slash trimmed, got %s", c.BaseURL)
	}
}

func TestAuthenticate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"session": map[string]string{"sid": "test-session-id"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "password")
	err := c.Authenticate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.SessionID != "test-session-id" {
		t.Errorf("expected session ID test-session-id, got %s", c.SessionID)
	}
}

func TestAuthenticateFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "wrong")
	err := c.Authenticate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetSummary(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/stats/summary" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(Summary{
			Queries:       1000,
			Blocked:       150,
			BlockedPct:    15.0,
			DomainsOnList: 85000,
			Status:        "enabled",
			ClientsEver:   5,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	s, err := c.GetSummary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Queries != 1000 {
		t.Errorf("expected 1000 queries, got %d", s.Queries)
	}
	if s.Blocked != 150 {
		t.Errorf("expected 150 blocked, got %d", s.Blocked)
	}
	if s.Status != "enabled" {
		t.Errorf("expected status enabled, got %s", s.Status)
	}
}

func TestGetTopDomains(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"top_domains": []map[string]any{
				{"domain": "google.com", "count": 100},
				{"domain": "github.com", "count": 50},
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	items, err := c.GetTopDomains(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Domain != "google.com" {
		t.Errorf("expected google.com, got %s", items[0].Domain)
	}
	if items[0].Count != 100 {
		t.Errorf("expected count 100, got %d", items[0].Count)
	}
}

func TestGetTopBlocked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"top_blocked": []map[string]any{
				{"domain": "ads.tracker.com", "count": 200},
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	items, err := c.GetTopBlocked(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Domain != "ads.tracker.com" {
		t.Errorf("expected ads.tracker.com, got %s", items[0].Domain)
	}
}

func TestEnable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/dns/blocking" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	if err := c.Enable(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDisable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	if err := c.Disable(30); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddDenylist(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/domains/deny/exact" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	if err := c.AddDenylist("evil.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddAllowlist(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/domains/allow/exact" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	if err := c.AddAllowlist("good.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetDenylist(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"domains": []map[string]string{
				{"domain": "bad1.com"},
				{"domain": "bad2.com"},
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	domains, err := c.GetDenylist()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(domains))
	}
	if domains[0] != "bad1.com" {
		t.Errorf("expected bad1.com, got %s", domains[0])
	}
}

func TestAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	_, err := c.GetSummary()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSessionIDSentInHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sid := r.Header.Get("X-FTL-SID")
		if sid != "my-session" {
			t.Errorf("expected X-FTL-SID my-session, got %s", sid)
		}
		json.NewEncoder(w).Encode(Summary{Status: "enabled"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	c.SessionID = "my-session"
	_, err := c.GetSummary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
