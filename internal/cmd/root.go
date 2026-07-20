package cmd

import (
	"fmt"
	"os"

	"github.com/deepglint/leangoo-cli/internal/output"
	"github.com/spf13/cobra"
)

// Version is injected by GoReleaser ldflags.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "leangoo",
	Short: "Leangoo (领歌) 在线版 CLI",
	Long:  "通过网页接口操作 Leangoo：登录、企业、项目、Sprint、Story。",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&output.JSON, "json", false, "以 JSON 输出（默认即 JSON）")
	rootCmd.AddCommand(authCmd())
	rootCmd.AddCommand(entCmd())
	rootCmd.AddCommand(projectCmd())
	rootCmd.AddCommand(sprintCmd())
	rootCmd.AddCommand(storyCmd())
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
