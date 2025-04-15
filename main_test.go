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
		body, _ := io.ReadAll(r.Body)
		receivedRequest = body
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Set up the config config
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

	if !strings.Contains(message.Text, "test-host/test-check") {
		t.Errorf("Expected message to contain entity/check name, got: %s", message.Text)
	}

	// Verify the thread
	if message.Thread.ThreadKey != "test-host" {
		t.Errorf("Expected thread key to be test-host, got: %s", message.Thread.ThreadKey)
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
				body, _ := io.ReadAll(r.Body)
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