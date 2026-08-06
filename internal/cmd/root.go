package cmd

import (
	"fmt"
	"os"

	"github.com/deepglint/leangoo-cli/internal/output"
	"github.com/deepglint/leangoo-cli/internal/update"
	"github.com/spf13/cobra"
)

// Version is injected by GoReleaser ldflags.
var Version = "dev"

var noUpdateCheck bool

var rootCmd = &cobra.Command{
	Use:   "leangoo",
	Short: "Leangoo (领歌) 在线版 CLI",
	Long:  "通过网页接口操作 Leangoo：登录、企业、项目、Sprint、Story。",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if noUpdateCheck || skipUpdateCheck(cmd) {
			return
		}
		update.NotifyIfOutdated(Version)
	},
}

func skipUpdateCheck(cmd *cobra.Command) bool {
	// Walk to the leaf command name.
	name := cmd.Name()
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "update", "version", "completion", "help":
			return true
		}
	}
	// `leangoo help ...` / unknown
	if name == "leangoo" {
		return true
	}
	return false
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&output.JSON, "json", false, "以 JSON 输出（默认即 JSON）")
	rootCmd.PersistentFlags().BoolVar(&noUpdateCheck, "no-update-check", false, "跳过启动时的版本更新检查")
	rootCmd.AddCommand(authCmd())
	rootCmd.AddCommand(entCmd())
	rootCmd.AddCommand(projectCmd())
	rootCmd.AddCommand(sprintCmd())
	rootCmd.AddCommand(storyCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(laneCmd())
	rootCmd.AddCommand(recycleCmd())
	rootCmd.AddCommand(updateCmd())
	rootCmd.AddCommand(versionCmd())
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "显示版本",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("leangoo", Version)
		},
	}
}
