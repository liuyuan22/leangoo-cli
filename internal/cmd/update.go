package cmd

import (
	"fmt"

	"github.com/deepglint/leangoo-cli/internal/output"
	"github.com/deepglint/leangoo-cli/internal/update"
	"github.com/spf13/cobra"
)

func updateCmd() *cobra.Command {
	var (
		version  string
		check    bool
		force    bool
		noSkills bool
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "从 GitHub Releases 更新 leangoo（覆盖当前二进制，默认同步 Skills）",
		Long: `从 https://github.com/liuyuan22/leangoo-cli/releases 下载对应平台产物，
替换当前正在使用的 leangoo 二进制，并默认把 Skills 同步到
~/.cursor/skills 与 ~/.claude/skills。

示例:
  leangoo update              # 升到最新
  leangoo update --check      # 只检查是否有新版本
  leangoo update --version v1.1.0
  leangoo update --no-skills  # 不更新 Skills
`,
		Run: func(cmd *cobra.Command, args []string) {
			res, err := update.Run(update.Options{
				Version:  version,
				Check:    check,
				Force:    force,
				NoSkills: noSkills,
				Current:  Version,
			})
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(res)
			if res.Action == "updated" {
				fmt.Fprintf(cmd.ErrOrStderr(), "提示: 新版本已写入 %s，可再执行 leangoo version 确认。\n", res.BinPath)
			}
		},
	}
	cmd.Flags().StringVar(&version, "version", "", "目标版本 tag（默认 latest，如 v1.1.0）")
	cmd.Flags().BoolVar(&check, "check", false, "仅检查是否有更新，不下载安装")
	cmd.Flags().BoolVar(&force, "force", false, "即使版本相同也重新下载安装")
	cmd.Flags().BoolVar(&noSkills, "no-skills", false, "不更新 Agent Skills")
	return cmd
}
