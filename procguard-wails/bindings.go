package main

import (
	"context"
	"procguard-wails/api"
)

// App struct
type App struct {
	ctx context.Context
	*api.Server
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}
