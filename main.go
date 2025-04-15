package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	corev2 "github.com/sensu/core/v2"
	"github.com/sensu/sensu-plugin-sdk/sensu"
)

// Config represents the handler plugin config.
type HandlerConfig struct {
	sensu.PluginConfig
	webhook   string
	dashboard string
}

var (
	config = HandlerConfig{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-google-chat-handler",
			Short:    "Sensu Google Chat Handler",
			Keyspace: "sensu.io/plugins/google-chat/config",
		},
	}

	options = []sensu.ConfigOption{
		&sensu.PluginConfigOption[string]{
			Path:      "webhook",
			Env:       "GOOGLE_CHAT_WEBHOOK",
			Argument:  "webhook",
			Shorthand: "w",
			Default:   "",
			Usage:     "The webhook URL to post the message to",
			Value:     &config.webhook,
		},
		&sensu.PluginConfigOption[string]{
			Path:      "dashboard",
			Env:       "SENSU_DASHBOARD",
			Argument:  "dashboard",
			Shorthand: "d",
			Default:   "",
			Usage:     "URL prefix to dashboard with namespace",
			Value:     &config.dashboard,
		},
	}
)

// statusStrings maps Sensu status codes to human-readable strings
var statusStrings = []string{"RESOLVED", "WARNING", "ALERT"}

// ThreadMessage represents the Google Chat message structure
type ThreadMessage struct {
	Text   string       `json:"text"`
	Thread ThreadConfig `json:"thread"`
}

// ThreadConfig represents the thread configuration for Google Chat
type ThreadConfig struct {
	ThreadKey string `json:"threadKey"`
}

func main() {
	handler := sensu.NewHandler(&config.PluginConfig, options, checkArgs, executeHandler)
	handler.Execute()
}

func checkArgs(_ *corev2.Event) error {
	if len(config.webhook) == 0 {
		return fmt.Errorf("--webhook or GOOGLE_CHAT_WEBHOOK environment variable is required")
	}
	if len(config.dashboard) == 0 {
		return fmt.Errorf("--dashboard or SENSU_DASHBOARD environment variable is required")
	}
	return nil
}

func executeHandler(event *corev2.Event) error {
	namespace := event.Entity.Namespace
	entity := event.Entity.System.Hostname
	if entity == "" {
		entity = event.Entity.Name
	}

	status := event.Check.Status
	checkName := event.Check.Name

	statusStr := "UNKNOWN"
	if int(status) >= 0 && int(status) < len(statusStrings) {
		statusStr = statusStrings[status]
	}

	// Construct the dashboard URL by joining elements and ensuring no double slashes
	parts := []string{strings.TrimRight(config.dashboard, "/"), namespace, "events", entity, checkName}
	eventURL := ""
	for i, part := range parts {
		if i == 0 {
			eventURL = part
		} else {
			eventURL = path.Join(eventURL, part)
		}
	}

	// Format the message text with fixed width status and a link to the event
	messageText := fmt.Sprintf("*`%-8s`* <%s|%s/%s>", statusStr, eventURL, entity, checkName)

	// Create the message payload
	message := ThreadMessage{
		Text: messageText,
		Thread: ThreadConfig{
			ThreadKey: entity,
		},
	}

	// Convert the message to JSON
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	// Send the message to Google Chat
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Post(config.webhook, "application/json", bytes.NewBuffer(messageBytes))
	if err != nil {
		return fmt.Errorf("failed to send message to Google Chat: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("error sending message to Google Chat, got status %d", resp.StatusCode)
	}

	return nil
}