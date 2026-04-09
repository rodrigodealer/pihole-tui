package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdGuardName(t *testing.T) {
	c := NewAdGuardClient("http://localhost", "admin", "pass")
	if c.Name() != "adguard" {
		t.Errorf("expected adguard, got %s", c.Name())
	}
}

func TestAdGuardGetSummary(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/control/stats":
			json.NewEncoder(w).Encode(map[string]any{
				"num_dns_queries":       5000,
				"num_blocked_filtering": 500,
			})
		case "/control/status":
			json.NewEncoder(w).Encode(map[string]any{
				"protection_enabled": true,
				"running":            true,
			})
		}
	}))
	defer srv.Close()

	c := NewAdGuardClient(srv.URL, "admin", "pass")
	s, err := c.GetSummary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Queries != 5000 {
		t.Errorf("expected 5000 queries, got %d", s.Queries)
	}
	if s.Blocked != 500 {
		t.Errorf("expected 500 blocked, got %d", s.Blocked)
	}
	if s.Status != "enabled" {
		t.Errorf("expected enabled, got %s", s.Status)
	}
	if s.BlockedPct != 10.0 {
		t.Errorf("expected 10%%, got %.1f%%", s.BlockedPct)
	}
}

func TestAdGuardGetTopDomains(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"top_queried_domains": []map[string]int{
				{"google.com": 100},
				{"github.com": 50},
			},
		})
	}))
	defer srv.Close()

	c := NewAdGuardClient(srv.URL, "admin", "pass")
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
}

func TestAdGuardGetDNSRecords(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]string{
			{"domain": "nas.local", "answer": "192.168.0.219"},
			{"domain": "gitea.local", "answer": "192.168.0.71"},
		})
	}))
	defer srv.Close()

	c := NewAdGuardClient(srv.URL, "admin", "pass")
	records, err := c.GetDNSRecords()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].Domain != "nas.local" {
		t.Errorf("expected nas.local, got %s", records[0].Domain)
	}
	if records[0].IP != "192.168.0.219" {
		t.Errorf("expected 192.168.0.219, got %s", records[0].IP)
	}
}

func TestAdGuardEnable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/control/dns_config" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewAdGuardClient(srv.URL, "admin", "pass")
	if err := c.Enable(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdGuardAddDNSRecord(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/control/rewrite/add" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewAdGuardClient(srv.URL, "admin", "pass")
	if err := c.AddDNSRecord("192.168.0.100", "test.local"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
