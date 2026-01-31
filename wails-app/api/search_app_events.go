package api

import (
	"strings"
	"wails-app/internal/data/query"
)

// Search handles searches for application events.
func (s *Server) Search(queryStr, since, until string) ([][]string, error) {
	return query.SearchAppEvents(s.db, strings.ToLower(queryStr), since, until)
}
