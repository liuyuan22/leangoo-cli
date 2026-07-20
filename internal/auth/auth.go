package auth

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/deepglint/leangoo-cli/internal/client"
	"github.com/deepglint/leangoo-cli/internal/config"
	"github.com/deepglint/leangoo-cli/internal/session"
)

var smsTokenRe = regexp.MustCompile(`(?i)(?:var\s+)?sms_token\s*=\s*["']([^"']+)["']`)

type LoginResult struct {
	HomeURL string
	Raw     json.RawMessage
}

func DoubleMD5(password string) string {
	if matched, _ := regexp.MatchString(`(?i)^[a-f0-9]{32}$`, password); matched {
		return strings.ToLower(password)
	}
	sum1 := md5.Sum([]byte(password))
	hex1 := hex.EncodeToString(sum1[:])
	sum2 := md5.Sum([]byte(hex1))
	return hex.EncodeToString(sum2[:])
}

func FetchSMSToken(c *client.Client) (string, error) {
	html, err := c.GetHTML("/login")
	if err != nil {
		return "", err
	}
	m := smsTokenRe.FindStringSubmatch(html)
	if len(m) < 2 {
		return "", fmt.Errorf("登录页未找到 sms_token")
	}
	return m[1], nil
}

func SendLoginCode(c *client.Client, countryCode, phone string) error {
	token, err := FetchSMSToken(c)
	if err != nil {
		return err
	}
	cc := strings.TrimPrefix(countryCode, "+")
	form := url.Values{}
	form.Set("type", "login_by_phone")
	form.Set("country_code", cc)
	form.Set("phone_number", phone)
	form.Set("token", token)
	api, _, err := c.PostForm("/sms/send_validate_code", form)
	if err != nil {
		return err
	}
	if !api.OK() {
		return fmt.Errorf("发送验证码失败: %s", api.MessageString())
	}
	return nil
}

func LoginWithPassword(c *client.Client, account, password string) (*LoginResult, error) {
	form := url.Values{}
	form.Set("account", account)
	form.Set("pwd", DoubleMD5(password))
	form.Set("loginRemPwdVal", "true")
	form.Set("isApp", "false")
	api, raw, err := c.PostForm("/login/check_pwd", form)
	if err != nil {
		return nil, err
	}
	if !api.OK() {
		return nil, fmt.Errorf("登录失败(error_code=%d): %s", api.ErrorCode, api.MessageString())
	}
	return finishLogin(c, account, api, raw)
}

func LoginWithCode(c *client.Client, countryCode, phone, code string) (*LoginResult, error) {
	cc := strings.TrimPrefix(countryCode, "+")
	form := url.Values{}
	form.Set("country_code", cc)
	form.Set("phone_number", phone)
	form.Set("verify_code", code)
	api, raw, err := c.PostForm("/login/login_by_phone", form)
	if err != nil {
		return nil, err
	}
	if !api.OK() {
		msg := api.MessageString()
		if api.ErrorCode == 403704 {
			return nil, fmt.Errorf("手机号未注册")
		}
		return nil, fmt.Errorf("登录失败(error_code=%d): %s", api.ErrorCode, msg)
	}
	return finishLogin(c, phone, api, raw)
}

func finishLogin(c *client.Client, account string, api *client.APIResponse, raw []byte) (*LoginResult, error) {
	var msg struct {
		URL string `json:"url"`
	}
	_ = json.Unmarshal(api.Message, &msg)
	home := msg.URL
	if home == "" {
		return nil, fmt.Errorf("登录成功但未返回跳转 URL: %s", string(raw))
	}
	if strings.HasPrefix(home, "/") {
		home = "https://www.lg.team" + home
	}
	// Follow redirect to establish session cookies / ent context.
	if _, err := c.HTTP.Get(home); err != nil {
		return nil, fmt.Errorf("打开工作区失败: %w", err)
	}
	c.PersistCookies(home)
	c.Session.Account = account
	c.Session.HomeURL = home
	if err := session.Save(c.Session); err != nil {
		return nil, err
	}
	return &LoginResult{HomeURL: home, Raw: api.Message}, nil
}

func Logout(c *client.Client) error {
	_, _, _ = c.PostForm("/login/logout", url.Values{})
	return session.Clear()
}

func AbsoluteKanbanURL(pathOrURL string) string {
	if strings.HasPrefix(pathOrURL, "http://") || strings.HasPrefix(pathOrURL, "https://") {
		return pathOrURL
	}
	if strings.HasPrefix(pathOrURL, "/kanban") {
		return "https://www.lg.team" + pathOrURL
	}
	if strings.HasPrefix(pathOrURL, "/") {
		return config.BaseURL + pathOrURL
	}
	return config.BaseURL + "/" + pathOrURL
}
