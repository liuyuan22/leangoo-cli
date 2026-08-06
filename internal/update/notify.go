package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/deepglint/leangoo-cli/internal/config"
)

const (
	checkCacheFile = "version_check.json"
	checkInterval  = 24 * time.Hour
	checkTimeout   = 2 * time.Second
)

type checkCache struct {
	CheckedAt time.Time `json:"checked_at"`
	Latest    string    `json:"latest"`
	Current   string    `json:"current,omitempty"`
}

// NotifyIfOutdated prints a stderr tip when a newer GitHub release exists.
// Uses a 24h local cache so most invocations do no network I/O.
// Never returns an error that should abort the command.
func NotifyIfOutdated(current string) {
	if os.Getenv("LEANGOO_NO_UPDATE_CHECK") == "1" {
		return
	}
	current = strings.TrimPrefix(strings.TrimSpace(current), "v")
	// Skip local/dev builds — only tip users of released binaries.
	if current == "" || current == "dev" {
		return
	}

	cache, _ := loadCheckCache()
	needFetch := cache == nil || time.Since(cache.CheckedAt) > checkInterval || cache.Latest == ""

	latest := ""
	if cache != nil {
		latest = strings.TrimPrefix(cache.Latest, "v")
	}

	if needFetch {
		fetched, err := fetchLatestVersionQuick()
		now := time.Now()
		if err == nil && fetched != "" {
			latest = strings.TrimPrefix(fetched, "v")
			_ = saveCheckCache(checkCache{CheckedAt: now, Latest: latest, Current: current})
		} else if cache != nil {
			// Keep old latest; bump checked_at lightly so we don't hammer on offline failures.
			cache.CheckedAt = now
			_ = saveCheckCache(*cache)
		} else {
			return
		}
	}

	if latest == "" || !isOutdated(current, latest) {
		return
	}
	fmt.Fprintf(os.Stderr, "提示: 有新版本 leangoo %s（当前 %s）。运行: leangoo update\n", latest, displayVer(current))
}

func isOutdated(current, latest string) bool {
	if latest == "" {
		return false
	}
	if current == "dev" || strings.Contains(current, "-dev") || strings.HasSuffix(current, "-dirty") {
		return current != latest
	}
	return current != latest
}

func fetchLatestVersionQuick() (string, error) {
	client := &http.Client{Timeout: checkTimeout}
	rel, err := fetchRelease(client, DefaultRepo, "latest")
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(rel.TagName, "v"), nil
}

func checkCachePath() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, checkCacheFile), nil
}

func loadCheckCache() (*checkCache, error) {
	path, err := checkCachePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c checkCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func saveCheckCache(c checkCache) error {
	path, err := checkCachePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
