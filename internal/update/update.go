package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const DefaultRepo = "liuyuan22/leangoo-cli"

type Options struct {
	Repo     string // owner/name
	Version  string // empty or "latest" → latest release; else tag like v1.1.0
	Check    bool   // only report, do not install
	Force    bool   // install even if same version
	NoSkills bool
	Current  string // current binary version (no v prefix)
	BinPath  string // path to replace; empty → os.Executable()
	HTTP     *http.Client
}

type Result struct {
	OK           bool     `json:"ok"`
	Action       string   `json:"action"` // check | up_to_date | updated
	Current      string   `json:"current"`
	Latest       string   `json:"latest"`
	Asset        string   `json:"asset,omitempty"`
	BinPath      string   `json:"bin_path,omitempty"`
	Skills       []string `json:"skills,omitempty"`
	DownloadURL  string   `json:"download_url,omitempty"`
	Message      string   `json:"message"`
}

type release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func Run(opts Options) (*Result, error) {
	if opts.Repo == "" {
		opts.Repo = DefaultRepo
	}
	if opts.HTTP == nil {
		opts.HTTP = &http.Client{Timeout: 120 * time.Second}
	}
	binPath := opts.BinPath
	if binPath == "" {
		exe, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("定位当前二进制失败: %w", err)
		}
		binPath, err = filepath.EvalSymlinks(exe)
		if err != nil {
			binPath = exe
		}
	}

	assetName, err := assetNameFor(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, err
	}

	rel, err := fetchRelease(opts.HTTP, opts.Repo, opts.Version)
	if err != nil {
		return nil, err
	}
	latest := strings.TrimPrefix(rel.TagName, "v")
	current := strings.TrimPrefix(opts.Current, "v")

	var downloadURL string
	for _, a := range rel.Assets {
		if a.Name == assetName {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		return nil, fmt.Errorf("Release %s 中未找到 %s", rel.TagName, assetName)
	}

	res := &Result{
		OK:          true,
		Current:     current,
		Latest:      latest,
		Asset:       assetName,
		BinPath:     binPath,
		DownloadURL: downloadURL,
	}

	if opts.Check {
		res.Action = "check"
		if current == latest && current != "" && current != "dev" {
			res.Message = fmt.Sprintf("已是最新版本 %s", latest)
		} else {
			res.Message = fmt.Sprintf("可更新: %s → %s", displayVer(current), latest)
		}
		return res, nil
	}

	if !opts.Force && current == latest && current != "" && current != "dev" {
		res.Action = "up_to_date"
		res.Message = fmt.Sprintf("已是最新版本 %s", latest)
		return res, nil
	}

	tmpdir, err := os.MkdirTemp("", "leangoo-update-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpdir)

	archivePath := filepath.Join(tmpdir, assetName)
	if err := downloadFile(opts.HTTP, downloadURL, archivePath); err != nil {
		return nil, fmt.Errorf("下载失败: %w", err)
	}

	extractDir := filepath.Join(tmpdir, "out")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return nil, err
	}
	if strings.HasSuffix(assetName, ".zip") {
		if err := unzip(archivePath, extractDir); err != nil {
			return nil, err
		}
	} else {
		if err := untargz(archivePath, extractDir); err != nil {
			return nil, err
		}
	}

	newBin, err := findFile(extractDir, "leangoo", "leangoo.exe")
	if err != nil {
		return nil, err
	}
	if err := replaceBinary(newBin, binPath); err != nil {
		return nil, fmt.Errorf("替换二进制失败 (%s): %w", binPath, err)
	}

	var skillsInstalled []string
	if !opts.NoSkills {
		skillsSrc, serr := findDir(extractDir, "skills")
		if serr == nil {
			skillsInstalled, _ = installSkills(skillsSrc)
		}
	}

	res.Action = "updated"
	res.Skills = skillsInstalled
	res.Message = fmt.Sprintf("已更新: %s → %s", displayVer(current), latest)
	_ = saveCheckCache(checkCache{CheckedAt: time.Now(), Latest: latest, Current: latest})
	return res, nil
}

func displayVer(v string) string {
	if v == "" {
		return "(unknown)"
	}
	return v
}

func assetNameFor(goos, goarch string) (string, error) {
	var osLabel, archLabel string
	switch goos {
	case "darwin":
		osLabel = "Darwin"
	case "linux":
		osLabel = "Linux"
	case "windows":
		osLabel = "Windows"
	default:
		return "", fmt.Errorf("不支持的操作系统: %s", goos)
	}
	switch goarch {
	case "amd64":
		archLabel = "x86_64"
	case "arm64":
		archLabel = "arm64"
	default:
		return "", fmt.Errorf("不支持的架构: %s", goarch)
	}
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("leangoo_%s_%s%s", osLabel, archLabel, ext), nil
}

func fetchRelease(client *http.Client, repo, version string) (*release, error) {
	rel, err := fetchReleaseAPI(client, repo, version)
	if err == nil {
		return rel, nil
	}
	// API 限流或失败时走 github.com 重定向 + releases/download（无需 API）
	fallback, ferr := fetchReleaseFallback(client, repo, version)
	if ferr != nil {
		return nil, fmt.Errorf("%v；回退也失败: %w", err, ferr)
	}
	return fallback, nil
}

func fetchReleaseAPI(client *http.Client, repo, version string) (*release, error) {
	var url string
	v := strings.TrimSpace(version)
	if v == "" || v == "latest" {
		url = fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	} else {
		if !strings.HasPrefix(v, "v") {
			v = "v" + v
		}
		url = fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", repo, v)
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "leangoo-cli")
	if tok := githubToken(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var rel release
	if err := json.Unmarshal(body, &rel); err != nil {
		return nil, err
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("无法解析 Release tag")
	}
	return &rel, nil
}

func fetchReleaseFallback(client *http.Client, repo, version string) (*release, error) {
	tag := strings.TrimSpace(version)
	if tag == "" || tag == "latest" {
		resolved, err := resolveLatestTag(client, repo)
		if err != nil {
			return nil, err
		}
		tag = resolved
	} else if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	asset, err := assetNameFor(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, err
	}
	return &release{
		TagName: tag,
		Assets: []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		}{
			{
				Name:               asset,
				BrowserDownloadURL: fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, tag, asset),
			},
		},
	}, nil
}

func resolveLatestTag(client *http.Client, repo string) (string, error) {
	// 不跟随重定向，从 Location 解析 tag
	noRedirect := *client
	noRedirect.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	url := fmt.Sprintf("https://github.com/%s/releases/latest", repo)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "leangoo-cli")
	resp, err := noRedirect.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", fmt.Errorf("无法解析 latest Location（HTTP %s）", resp.Status)
	}
	// .../releases/tag/v1.1.0
	const marker = "/releases/tag/"
	i := strings.LastIndex(loc, marker)
	if i < 0 {
		return "", fmt.Errorf("无法从 Location 解析 tag: %s", loc)
	}
	tag := strings.TrimSuffix(loc[i+len(marker):], "/")
	if tag == "" {
		return "", fmt.Errorf("空 tag: %s", loc)
	}
	return tag, nil
}

func githubToken() string {
	for _, k := range []string{"GITHUB_TOKEN", "GH_TOKEN"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func downloadFile(client *http.Client, url, dest string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "leangoo-cli")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", resp.Status)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func untargz(archive, dest string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		target, err := safeJoin(dest, hdr.Name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)|0o200)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
}

func unzip(archive, dest string) error {
	r, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		target, err := safeJoin(dest, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode()|0o200)
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		out.Close()
		rc.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

func safeJoin(base, name string) (string, error) {
	clean := filepath.Clean(name)
	if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return "", fmt.Errorf("非法路径: %s", name)
	}
	target := filepath.Join(base, clean)
	rel, err := filepath.Rel(base, target)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("非法路径: %s", name)
	}
	return target, nil
}

func findFile(root string, names ...string) (string, error) {
	want := map[string]bool{}
	for _, n := range names {
		want[n] = true
	}
	var found string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || found != "" {
			return nil
		}
		if want[info.Name()] {
			found = path
			return io.EOF
		}
		return nil
	})
	if found == "" {
		return "", fmt.Errorf("压缩包中未找到 leangoo 二进制")
	}
	return found, nil
}

func findDir(root, name string) (string, error) {
	var found string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || found != "" {
			return nil
		}
		if info.IsDir() && info.Name() == name {
			found = path
			return io.EOF
		}
		return nil
	})
	if found == "" {
		return "", fmt.Errorf("未找到 %s 目录", name)
	}
	return found, nil
}

func replaceBinary(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	tmp := dest + ".new"
	if err := os.WriteFile(tmp, data, 0o755); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	// On Windows, rename over existing may fail; try remove first.
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(dest)
		if err2 := os.Rename(tmp, dest); err2 != nil {
			_ = os.Remove(tmp)
			return err2
		}
	}
	return nil
}

func installSkills(skillsSrc string) ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	var installed []string
	for _, destRoot := range []string{
		filepath.Join(home, ".cursor", "skills"),
		filepath.Join(home, ".claude", "skills"),
	} {
		for _, name := range []string{"leangoo-shared", "leangoo-sprint", "leangoo-story"} {
			src := filepath.Join(skillsSrc, name)
			if st, err := os.Stat(src); err != nil || !st.IsDir() {
				continue
			}
			dst := filepath.Join(destRoot, name)
			_ = os.RemoveAll(dst)
			if err := copyDir(src, dst); err != nil {
				return installed, err
			}
			installed = append(installed, dst)
		}
	}
	return installed, nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}
