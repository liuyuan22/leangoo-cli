package config

import (
	"os"
	"path/filepath"
)

const (
	BaseURL    = "https://www.lg.team/kanban"
	DirName    = ".leangoo-cli"
	SessionFile = "session.json"
)

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, DirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func SessionPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, SessionFile), nil
}
