package session

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/deepglint/leangoo-cli/internal/config"
	"golang.org/x/net/publicsuffix"
)

type Ent struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Sign    string `json:"sign,omitempty"`
	HomeURL string `json:"home_url"`
}

type Session struct {
	Account           string    `json:"account,omitempty"`
	HomeURL           string    `json:"home_url,omitempty"`
	CurrentEnt        *Ent      `json:"current_ent,omitempty"`
	Ents              []Ent     `json:"ents,omitempty"`
	NewLeangooWebURL  string    `json:"new_leangoo_web_url,omitempty"`
	Cookies           []Cookie  `json:"cookies"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Cookie struct {
	Name   string    `json:"name"`
	Value  string    `json:"value"`
	Domain string    `json:"domain"`
	Path   string    `json:"path"`
	Secure bool      `json:"secure"`
	Expires time.Time `json:"expires,omitempty"`
}

var mu sync.Mutex

func Load() (*Session, error) {
	path, err := config.SessionPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("未登录，请先执行: leangoo auth login")
		}
		return nil, err
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if len(s.Cookies) == 0 {
		return nil, fmt.Errorf("会话无效，请重新登录")
	}
	return &s, nil
}

func Save(s *Session) error {
	mu.Lock()
	defer mu.Unlock()
	path, err := config.SessionPath()
	if err != nil {
		return err
	}
	s.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func Clear() error {
	path, err := config.SessionPath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *Session) ApplyToJar(jar http.CookieJar) error {
	byURL := map[string][]*http.Cookie{}
	for _, c := range s.Cookies {
		domain := c.Domain
		if domain == "" {
			domain = "www.lg.team"
		}
		u := &url.URL{Scheme: "https", Host: trimDot(domain), Path: "/"}
		key := u.String()
		hc := &http.Cookie{
			Name:   c.Name,
			Value:  c.Value,
			Path:   c.Path,
			Domain: c.Domain,
			Secure: c.Secure,
		}
		if c.Path == "" {
			hc.Path = "/"
		}
		if !c.Expires.IsZero() {
			hc.Expires = c.Expires
		}
		byURL[key] = append(byURL[key], hc)
	}
	for raw, cookies := range byURL {
		u, err := url.Parse(raw)
		if err != nil {
			continue
		}
		jar.SetCookies(u, cookies)
	}
	return nil
}

func CaptureJar(jar http.CookieJar, urls ...string) []Cookie {
	var out []Cookie
	seen := map[string]struct{}{}
	for _, raw := range urls {
		u, err := url.Parse(raw)
		if err != nil {
			continue
		}
		for _, c := range jar.Cookies(u) {
			domain := c.Domain
			if domain == "" {
				domain = u.Host
			}
			path := c.Path
			if path == "" {
				path = "/"
			}
			key := domain + "|" + path + "|" + c.Name
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, Cookie{
				Name:    c.Name,
				Value:   c.Value,
				Domain:  domain,
				Path:    path,
				Secure:  c.Secure,
				Expires: c.Expires,
			})
		}
	}
	return out
}

func NewJar() (http.CookieJar, error) {
	return cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
}

func trimDot(d string) string {
	if len(d) > 0 && d[0] == '.' {
		return d[1:]
	}
	return d
}

func (s *Session) RequireEnt() (*Ent, error) {
	if s.CurrentEnt == nil {
		return nil, fmt.Errorf("未选择企业，请先执行: leangoo ent list && leangoo ent use <id>")
	}
	return s.CurrentEnt, nil
}
