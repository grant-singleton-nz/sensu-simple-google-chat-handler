package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corev2 "github.com/sensu/core/v2"
)

func TestCheckArgs(t *testing.T) {
	testCases := []struct {
		name       string
		webhook    string
		dashboard  string
		shouldFail bool
	}{
		{
			name:       "Valid arguments",
			webhook:    "https://chat.googleapis.com/webhook",
			dashboard:  "https://sensu.example.com",
			shouldFail: false,
		},
		{
			name:       "Missing webhook",
			webhook:    "",
			dashboard:  "https://sensu.example.com",
			shouldFail: true,
		},
		{
			name:       "Missing dashboard",
			webhook:    "https://chat.googleapis.com/webhook",
			dashboard:  "",
			shouldFail: true,
		},
		{
			name:       "HTTP webhook rejected",
			webhook:    "http://chat.googleapis.com/webhook",
			dashboard:  "https://sensu.example.com",
			shouldFail: true,
		},
		{
			name:       "Non-URL webhook rejected",
			webhook:    "not-a-url",
			dashboard:  "https://sensu.example.com",
			shouldFail: true,
		},
		{
			name:       "File scheme webhook rejected",
			webhook:    "file:///etc/passwd",
			dashboard:  "https://sensu.example.com",
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up test configuration
			config.webhook = tc.webhook
			config.dashboard = tc.dashboard

			// Call the function being tested
			err := checkArgs(nil)

			// Verify the result
			if tc.shouldFail && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tc.shouldFail && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestExecuteHandler(t *testing.T) {
	// Create a test server to receive the webhook request
	var receivedRequest []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		receivedRequest = body
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Set up the config
	config.webhook = ts.URL
	config.dashboard = "https://sensu.example.com"

	// Create a sample Sensu event
	event := &corev2.Event{
		Entity: &corev2.Entity{
			ObjectMeta: corev2.ObjectMeta{
				Name:      "test-entity",
				Namespace: "default",
			},
			System: corev2.System{
				Hostname: "test-host",
			},
		},
		Check: &corev2.Check{
			ObjectMeta: corev2.ObjectMeta{
				Name: "test-check",
			},
			Status: 1, // WARNING status
		},
	}

	// Call the function being tested
	err := executeHandler(event)
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	// Verify the request sent to the webhook
	var message ThreadMessage
	err = json.Unmarshal(receivedRequest, &message)
	if err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	// Verify the message content
	if !strings.Contains(message.Text, "WARNING") {
		t.Errorf("Expected message to contain WARNING status, got: %s", message.Text)
	}

	if !strings.Contains(message.Text, config.dashboard) {
		t.Errorf("Expected message to contain %s, got: %s", config.dashboard, message.Text)
	}

	if !strings.Contains(message.Text, "test-host/test-check") {
		t.Errorf("Expected message to contain entity/check name, got: %s", message.Text)
	}

	// Verify the thread
	if message.Thread.ThreadKey != "test-host" {
		t.Errorf("Expected thread key to be test-host, got: %s", message.Thread.ThreadKey)
	}
}

func TestExecuteHandlerErrorResponse(t *testing.T) {
	// Create a test server that returns an error with a body
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid request"}`))
	}))
	defer ts.Close()

	config.webhook = ts.URL
	config.dashboard = "https://sensu.example.com"

	event := &corev2.Event{
		Entity: &corev2.Entity{
			ObjectMeta: corev2.ObjectMeta{
				Name:      "test-entity",
				Namespace: "default",
			},
			System: corev2.System{
				Hostname: "test-host",
			},
		},
		Check: &corev2.Check{
			ObjectMeta: corev2.ObjectMeta{
				Name: "test-check",
			},
			Status: 1,
		},
	}

	err := executeHandler(event)
	if err == nil {
		t.Fatal("Expected error for 400 response but got none")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("Expected error to contain status code 400, got: %v", err)
	}
	if !strings.Contains(err.Error(), "invalid request") {
		t.Errorf("Expected error to contain response body, got: %v", err)
	}
}

func TestExecuteHandlerEntityFallback(t *testing.T) {
	// Verify that when hostname is empty, entity name is used instead
	var receivedRequest []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		receivedRequest = body
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	config.webhook = ts.URL
	config.dashboard = "https://sensu.example.com"

	event := &corev2.Event{
		Entity: &corev2.Entity{
			ObjectMeta: corev2.ObjectMeta{
				Name:      "entity-name-fallback",
				Namespace: "default",
			},
			System: corev2.System{
				Hostname: "", // empty hostname to trigger fallback
			},
		},
		Check: &corev2.Check{
			ObjectMeta: corev2.ObjectMeta{
				Name: "test-check",
			},
			Status: 0,
		},
	}

	err := executeHandler(event)
	if err != nil {
		t.Fatalf("Expected no error but got: %v", err)
	}

	var message ThreadMessage
	err = json.Unmarshal(receivedRequest, &message)
	if err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	if !strings.Contains(message.Text, "entity-name-fallback") {
		t.Errorf("Expected message to use entity name as fallback, got: %s", message.Text)
	}

	if message.Thread.ThreadKey != "entity-name-fallback" {
		t.Errorf("Expected thread key to use entity name as fallback, got: %s", message.Thread.ThreadKey)
	}
}

func TestStatusString(t *testing.T) {
	testCases := []struct {
		status     uint32
		expectText string
	}{
		{0, "RESOLVED"},
		{1, "WARNING"},
		{2, "ALERT"},
		{3, "UNKNOWN"},
		{999, "UNKNOWN"},
	}

	for _, tc := range testCases {
		t.Run(tc.expectText, func(t *testing.T) {
			// Create a sample Sensu event with the test status
			event := &corev2.Event{
				Entity: &corev2.Entity{
					ObjectMeta: corev2.ObjectMeta{
						Name:      "test-entity",
						Namespace: "default",
					},
					System: corev2.System{
						Hostname: "test-host",
					},
				},
				Check: &corev2.Check{
					ObjectMeta: corev2.ObjectMeta{
						Name: "test-check",
					},
					Status: tc.status,
				},
			}

			// Set up a test server to capture the webhook payload
			var receivedRequest []byte
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("Failed to read request body: %v", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				receivedRequest = body
				w.WriteHeader(http.StatusOK)
			}))
			defer ts.Close()

			// Configure the config
			config.webhook = ts.URL
			config.dashboard = "https://sensu.example.com"

			// Execute the handler
			err := executeHandler(event)
			if err != nil {
				t.Fatalf("Failed to execute handler: %v", err)
			}

			// Verify the status text in the message
			var message ThreadMessage
			err = json.Unmarshal(receivedRequest, &message)
			if err != nil {
				t.Fatalf("Failed to unmarshal request: %v", err)
			}

			if !strings.Contains(message.Text, tc.expectText) {
				t.Errorf("Expected message to contain %s status, got: %s", tc.expectText, message.Text)
			}
		})
	}
}
