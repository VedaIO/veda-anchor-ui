package api

import (
	"encoding/json"
	"io"
	"net/http"
	"procguard/internal/data"
	"slices"
	"time"
)

// handleGetWebBlocklist returns the list of blocked websites with their details.
func (s *Server) handleGetWebBlocklist(w http.ResponseWriter, r *http.Request) {
	list, err := data.GetBlockedWebsitesWithDetails(s.db)
	if err != nil {
		http.Error(w, "Failed to load web blocklist with details", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(list); err != nil {
		s.Logger.Printf("Error encoding response: %v", err)
	}
}

// handleAddWebBlocklist adds a domain to the web blocklist.
// It expects a JSON request with a `domain` field.
func (s *Server) handleAddWebBlocklist(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if _, err := data.AddWebsiteToBlocklist(req.Domain); err != nil {
		http.Error(w, "Failed to add to web blocklist", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]bool{"ok": true}); err != nil {
		s.Logger.Printf("Error encoding response: %v", err)
	}
}

// handleRemoveWebBlocklist removes a domain from the web blocklist.
// It expects a JSON request with a `domain` field.
func (s *Server) handleRemoveWebBlocklist(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if _, err := data.RemoveWebsiteFromBlocklist(req.Domain); err != nil {
		http.Error(w, "Failed to remove from web blocklist", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]bool{"ok": true}); err != nil {
		s.Logger.Printf("Error encoding response: %v", err)
	}
}

// handleClearWebBlocklist removes all domains from the web blocklist.
func (s *Server) handleClearWebBlocklist(w http.ResponseWriter, r *http.Request) {
	if err := data.ClearWebBlocklist(); err != nil {
		http.Error(w, "Failed to clear web blocklist", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]bool{"ok": true}); err != nil {
		s.Logger.Printf("Error encoding response: %v", err)
	}
}

// handleSaveWebBlocklist saves the current web blocklist to a file for export.
func (s *Server) handleSaveWebBlocklist(w http.ResponseWriter, r *http.Request) {
	list, err := data.LoadWebBlocklist()
	if err != nil {
		http.Error(w, "Failed to get web blocklist", http.StatusInternalServerError)
		return
	}

	header := map[string]interface{}{
		"exported_at": time.Now().Format(time.RFC3339),
		"blocked":     list,
	}

	b, err := json.MarshalIndent(header, "", "  ")
	if err != nil {
		http.Error(w, "Failed to marshal web blocklist", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=procguard_web_blocklist.json")
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(b); err != nil {
		s.Logger.Printf("Error writing response: %v", err)
	}
}

// handleLoadWebBlocklist loads a web blocklist from an uploaded file and merges it with the existing blocklist.
func (s *Server) handleLoadWebBlocklist(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file from form", http.StatusBadRequest)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			s.Logger.Printf("Error closing file: %v", err)
		}
	}()

	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read uploaded file", http.StatusInternalServerError)
		return
	}

	var newEntries []string
	var savedList struct {
		Blocked []string `json:"blocked"`
	}

	err = json.Unmarshal(content, &newEntries)
	if err != nil {
		err2 := json.Unmarshal(content, &savedList)
		if err2 != nil {
			http.Error(w, "Invalid JSON format in uploaded file", http.StatusBadRequest)
			return
		}
		newEntries = savedList.Blocked
	}

	existingList, err := data.LoadWebBlocklist()
	if err != nil {
		http.Error(w, "Failed to load existing web blocklist", http.StatusInternalServerError)
		return
	}

	for _, entry := range newEntries {
		if !slices.Contains(existingList, entry) {
			existingList = append(existingList, entry)
		}
	}

	if err := data.SaveWebBlocklist(existingList); err != nil {
		http.Error(w, "Failed to save merged web blocklist", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]bool{"ok": true}); err != nil {
		s.Logger.Printf("Error encoding response: %v", err)
	}
}

// handleGetWebLogs retrieves web logs from the database within a given time range.
func (s *Server) handleGetWebLogs(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	sinceStr := r.URL.Query().Get("since")
	untilStr := r.URL.Query().Get("until")

	entries, err := data.GetWebLogs(s.db, query, sinceStr, untilStr)
	if err != nil {
		http.Error(w, "Failed to query web logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(entries); err != nil {
		s.Logger.Printf("Error encoding response: %v", err)
	}
}
