package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/deepglint/leangoo-cli/internal/config"
	"github.com/deepglint/leangoo-cli/internal/session"
)

type Client struct {
	HTTP    *http.Client
	BaseURL string
	Jar     http.CookieJar
	Session *session.Session
}

type APIResponse struct {
	Succeed   any             `json:"succeed"`
	Message   json.RawMessage `json:"message"`
	ErrorCode int             `json:"error_code"`
}

func (r APIResponse) OK() bool {
	switch v := r.Succeed.(type) {
	case bool:
		return v
	case float64:
		return v != 0
	case string:
		return v == "1" || strings.EqualFold(v, "true")
	default:
		return false
	}
}

func (r APIResponse) MessageString() string {
	if len(r.Message) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(r.Message, &s); err == nil {
		return s
	}
	return string(r.Message)
}

func New() (*Client, error) {
	jar, err := session.NewJar()
	if err != nil {
		return nil, err
	}
	return &Client{
		HTTP: &http.Client{
			Jar:     jar,
			Timeout: 60 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		BaseURL: config.BaseURL,
		Jar:     jar,
	}, nil
}

func NewFromSession() (*Client, error) {
	c, err := New()
	if err != nil {
		return nil, err
	}
	s, err := session.Load()
	if err != nil {
		return nil, err
	}
	if err := s.ApplyToJar(c.Jar); err != nil {
		return nil, err
	}
	c.Session = s
	return c, nil
}

func (c *Client) URL(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(c.BaseURL, "/") + path
}

func (c *Client) PostForm(path string, form url.Values) (*APIResponse, []byte, error) {
	req, err := http.NewRequest(http.MethodPost, c.URL(path), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("User-Agent", "leangoo-cli/0.1")
	return c.doJSON(req)
}

func (c *Client) Get(path string, query url.Values) (*APIResponse, []byte, error) {
	u := c.URL(path)
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("User-Agent", "leangoo-cli/0.1")
	return c.doJSON(req)
}

func (c *Client) GetHTML(path string) (string, error) {
	u := c.URL(path)
	if i := strings.Index(u, "#"); i >= 0 {
		u = u[:i]
	}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("User-Agent", "leangoo-cli/0.1")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	return string(body), nil
}

func (c *Client) doJSON(req *http.Request) (*APIResponse, []byte, error) {
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, body, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	var api APIResponse
	if err := json.Unmarshal(body, &api); err != nil {
		return nil, body, fmt.Errorf("解析 JSON 失败: %w; body=%s", err, truncate(string(body), 200))
	}
	return &api, body, nil
}

func (c *Client) PersistCookies(homeURL string) {
	urls := []string{config.BaseURL + "/", "https://www.lg.team/", "https://lg.team/"}
	if homeURL != "" {
		urls = append(urls, homeURL)
	}
	if c.Session == nil {
		c.Session = &session.Session{}
	}
	c.Session.Cookies = session.CaptureJar(c.Jar, urls...)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
