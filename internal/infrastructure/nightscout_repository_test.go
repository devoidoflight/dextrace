package infrastructure

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/brkss/dextrace/internal/domain"
)

func TestGetLastRecord_UsesSecretHeaderAndParsesArray(t *testing.T) {
	secret := "mysecret"
	h := sha1.New()
	h.Write([]byte(secret))
	expected := hex.EncodeToString(h.Sum(nil))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/entries.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("api-secret") != expected {
			t.Fatalf("expected api-secret header %q, got %q", expected, r.Header.Get("api-secret"))
		}
		_ = json.NewEncoder(w).Encode([]domain.NightscoutEntry{{DateString: "2026-01-01T10:00:00Z", Date: 1}})
	}))
	defer ts.Close()

	repo := NewNightscoutRepository(ts.URL, secret)
	record, err := repo.GetLastRecord()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record == nil {
		t.Fatal("expected record, got nil")
	}
}

func TestGetLastRecord_ReturnsHelpfulErrorForObjectResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status":401,"message":"unauthorized"}`))
	}))
	defer ts.Close()

	repo := NewNightscoutRepository(ts.URL, "")
	_, err := repo.GetLastRecord()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPushData_NoNewEntriesReturnsNil(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode([]domain.NightscoutEntry{{DateString: now}})
		case http.MethodPost:
			t.Fatal("did not expect POST when no new entries")
		}
	}))
	defer ts.Close()

	repo := NewNightscoutRepository(ts.URL, "")
	err := repo.PushData([]domain.GetDataResponse{{Timestamp: now, Value: 100}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
