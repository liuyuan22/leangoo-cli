package output

import (
	"encoding/json"
	"fmt"
	"os"
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
	os.Exit(1)
}
