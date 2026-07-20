package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

var JSON bool

func Print(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func PrintText(format string, args ...any) {
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}

func Fatal(err error) {
	fmt.Fprintf(os.Stderr, "错误: %v\n", err)
	msg := err.Error()
	if strings.Contains(msg, "未登录") || strings.Contains(msg, "会话无效") || strings.Contains(msg, "重新登录") {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "尚未登录领歌。请在本机终端执行（交互登录，勿在 Agent 里盲填密码）：")
		fmt.Fprintln(os.Stderr, "  leangoo auth login")
		fmt.Fprintln(os.Stderr, "登录成功后再让 Agent 重试。")
	}
	os.Exit(1)
}
