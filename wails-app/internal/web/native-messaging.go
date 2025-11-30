package web

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"os"
	"reflect"
	"strings"
	"time"
	"wails-app/internal/data"
)

const (
	// pollInterval is the interval at which the web blocklist is polled for changes.
	pollInterval = 500 * time.Millisecond
)

// WebMetadataPayload is the payload for the log_web_metadata message from the extension.
type WebMetadataPayload struct {
	Domain  string `json:"domain"`
	Title   string `json:"title"`
	IconURL string `json:"iconUrl"`
}

// Request is a message received from the browser extension.
type Request struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// Response is a message sent to the browser extension.
type Response struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Run starts the native messaging host, which listens for messages from the browser extension.
func Run() {
	log := data.GetLogger()

	// Start a goroutine to poll the web blocklist and push updates to the extension.
	go pollWebBlocklist()

	// The main loop reads messages from stdin, which is connected to the browser extension.
	// CRITICAL: We must check if Stdin is actually a pipe (connected to Chrome).
	// If we run this when launched by user (double-click), Stdin is invalid and we get infinite loop errors.
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		log.Println("Stdin is a terminal, not a pipe. Skipping native messaging host.")
		return
	}

	for {
		// The native messaging protocol prefixes each message with its length in bytes.
		var length uint32
		if err := binary.Read(os.Stdin, binary.LittleEndian, &length); err != nil {
			if err == io.EOF {
				log.Println("EOF received, exiting native messaging host.")
				break // Exit loop on EOF
			}
			// If we get an error here, it likely means Stdin is closed or invalid.
			// We should exit the loop to avoid spamming logs.
			log.Printf("Error reading message length: %v. Exiting native messaging loop.", err)
			break
		}

		msg := make([]byte, length)
		if _, err := io.ReadFull(os.Stdin, msg); err != nil {
			log.Printf("Error reading message body: %v", err)
			continue
		}

		var req Request
		if err := json.Unmarshal(msg, &req); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			continue
		}

		// Handle the message based on its type.
		switch req.Type {
		case "ping":
			var payload string
			if err := json.Unmarshal(req.Payload, &payload); err != nil {
				log.Printf("Error unmarshalling ping payload: %v", err)
				continue
			}
			resp := Response{
				Type:    "echo",
				Payload: payload,
			}
			sendMessage(resp)
		case "log_url":
			var url string
			if err := json.Unmarshal(req.Payload, &url); err != nil {
				log.Printf("Error unmarshalling log_url payload: %v", err)
				continue
			}
			// Ignore logging the app's own GUI.
			if strings.HasPrefix(url, "http://127.0.0.1:58141") {
				continue
			}

			// Log the URL directly to the database
			data.EnqueueWrite("INSERT INTO web_events (url, timestamp) VALUES (?, ?)", url, time.Now().Unix())

		case "log_web_metadata":
			var payload WebMetadataPayload
			if err := json.Unmarshal(req.Payload, &payload); err != nil {
				log.Printf("Error unmarshalling log_web_metadata payload: %v", err)
				continue
			}
			
			// Log metadata directly to the database
			data.EnqueueWrite("INSERT OR REPLACE INTO web_metadata (domain, title, icon_url, timestamp) VALUES (?, ?, ?, ?)", 
				payload.Domain, payload.Title, payload.IconURL, time.Now().Unix())

		case "get_web_blocklist":
			list, err := data.LoadWebBlocklist()
			if err != nil {
				log.Printf("Error loading web blocklist: %v", err)
				continue
			}
			resp := Response{
				Type:    "web_blocklist",
				Payload: list,
			}
			sendMessage(resp)
		case "add_to_web_blocklist":
			var domain string
			if err := json.Unmarshal(req.Payload, &domain); err != nil {
				log.Printf("Error unmarshalling add_to_web_blocklist payload: %v", err)
				continue
			}
			if _, err := data.AddWebsiteToBlocklist(domain); err != nil {
				log.Printf("Error adding to web blocklist: %v", err)
			}
		default:
			// Optionally handle unknown message types.
		}
	}
}

// pollWebBlocklist periodically checks for changes in the web blocklist and sends updates to the extension.
func pollWebBlocklist() {
	log := data.GetLogger()
	var lastBlocklist []string
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for range ticker.C {
		// Load blocklist directly from data package
		list, err := data.LoadWebBlocklist()
		if err != nil {
			log.Printf("Failed to get web blocklist: %v", err)
			continue
		}

		// Only send an update if the blocklist has changed.
		if !reflect.DeepEqual(list, lastBlocklist) {
			lastBlocklist = list
			resp := Response{
				Type:    "web_blocklist",
				Payload: list,
			}
			sendMessage(resp)
		}
	}
}

// sendMessage sends a message to the browser extension.
func sendMessage(resp Response) {
	log := data.GetLogger()
	b, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error marshalling response: %v", err)
		return
	}

	// The native messaging protocol requires that the message length be sent first.
	if err := binary.Write(os.Stdout, binary.LittleEndian, uint32(len(b))); err != nil {
		log.Printf("Error writing message length: %v", err)
		return
	}

	// Then, the message body is sent.
	if _, err := os.Stdout.Write(b); err != nil {
		log.Printf("Error writing message body: %v", err)
		return
	}
}
