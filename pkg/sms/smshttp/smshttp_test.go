package smshttp_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/sms"
	"github.com/Shmeeshmee/salus/pkg/sms/smshttp"
)

var _ sms.Sender = (*smshttp.Sender)(nil)

const number = "+38760000001"

func TestSender_PostsTheMessageWithTheTokenAndReadsTheID(t *testing.T) {
	t.Parallel()
	var got map[string]string
	var auth string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Error(err)
		}
		_, _ = w.Write([]byte(`{"id":"p-1"}`))
	}))
	t.Cleanup(srv.Close)
	s, err := smshttp.New(srv.Client(), srv.URL, "synthetic-token")
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.Send(t.Context(), sms.Message{To: number, Body: "Reminder", Reference: "d1"})
	if err != nil || id != "p-1" {
		t.Fatalf("Send() = %q, %v", id, err)
	}
	if auth != "Bearer synthetic-token" || got["to"] != number || got["text"] != "Reminder" || got["reference"] != "d1" {
		t.Errorf("sent %v with %q", got, auth)
	}
}

func TestSender_MapsTheAnswersWithoutTheirContent(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		status   int
		body     string
		rejected bool
	}{
		{"a refused number", http.StatusBadRequest, `{"error":"` + number + `"}`, true},
		{"a provider failure", http.StatusBadGateway, `down`, false},
		{"no id", http.StatusOK, `{}`, false},
	} {
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tt.status)
			_, _ = w.Write([]byte(tt.body))
		}))
		s, err := smshttp.New(srv.Client(), srv.URL, "synthetic-token")
		if err != nil {
			t.Fatal(err)
		}
		_, err = s.Send(t.Context(), sms.Message{To: number, Body: "Reminder"})
		srv.Close()
		if err == nil || errors.Is(err, sms.ErrRejected) != tt.rejected || strings.Contains(err.Error(), number) {
			t.Errorf("%s: Send() = %v, want rejected %t and no number", tt.name, err, tt.rejected)
		}
	}
}

func TestNew_NeedsHTTPSAClientAndAToken(t *testing.T) {
	t.Parallel()
	for name, tt := range map[string]struct {
		client   *http.Client
		endpoint string
		token    string
	}{
		"plain http":  {&http.Client{}, "http://sms.example.test/send", "t"},
		"credentials": {&http.Client{}, "https://u:p@sms.example.test/send", "t"},
		"no token":    {&http.Client{}, "https://sms.example.test/send", ""},
		"no client":   {nil, "https://sms.example.test/send", "t"},
		"not a URL":   {&http.Client{}, "://", "t"},
	} {
		if _, err := smshttp.New(tt.client, tt.endpoint, tt.token); err == nil {
			t.Errorf("%s: New() succeeded", name)
		}
	}
}

func TestSender_RefusesAnInvalidMessageAndAnUnreachableProvider(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.NotFoundHandler())
	s, err := smshttp.New(srv.Client(), srv.URL, "t")
	srv.Close()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Send(t.Context(), sms.Message{To: "0600", Body: "x"}); !errors.Is(err, sms.ErrInvalidMessage) {
		t.Errorf("invalid number: %v", err)
	}
	if _, err := s.Send(t.Context(), sms.Message{To: number, Body: "x"}); err == nil || errors.Is(err, sms.ErrRejected) {
		t.Errorf("closed server: %v", err)
	}
}
