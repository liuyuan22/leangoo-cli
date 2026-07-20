package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/deepglint/leangoo-cli/internal/api"
	"github.com/deepglint/leangoo-cli/internal/auth"
	"github.com/deepglint/leangoo-cli/internal/client"
	"github.com/deepglint/leangoo-cli/internal/output"
	"github.com/deepglint/leangoo-cli/internal/parse"
	"github.com/deepglint/leangoo-cli/internal/session"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func authCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "登录 / 退出 / 状态",
	}
	cmd.AddCommand(authLoginCmd())
	cmd.AddCommand(authSendCodeCmd())
	cmd.AddCommand(authLogoutCmd())
	cmd.AddCommand(authStatusCmd())
	return cmd
}

func authLoginCmd() *cobra.Command {
	var phone, password, code, country string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "登录（交互式；也可用 --phone/--password/--code）",
		Long: `交互式登录流程：
  1. 输入手机号/账号
  2. 选择登录方式（密码 / 短信验证码）
  3. 输入密码或验证码

也可直接传参：leangoo auth login --phone ... --password ...`,
		Run: func(cmd *cobra.Command, args []string) {
			var err error
			if phone == "" {
				phone, err = promptLine("手机号/账号")
				if err != nil {
					output.Fatal(err)
				}
			}
			if phone == "" {
				output.Fatal(fmt.Errorf("手机号不能为空"))
			}

			c, err := client.New()
			if err != nil {
				output.Fatal(err)
			}

			useCode := code != ""
			usePassword := password != ""
			if !useCode && !usePassword {
				mode, err := promptLoginMode()
				if err != nil {
					output.Fatal(err)
				}
				useCode = mode == "code"
				usePassword = mode == "password"
			}

			var result *auth.LoginResult
			if useCode {
				if code == "" {
					fmt.Fprintln(os.Stderr, "正在发送验证码…")
					if err := auth.SendLoginCode(c, country, phone); err != nil {
						output.Fatal(err)
					}
					fmt.Fprintln(os.Stderr, "验证码已发送，请查收短信。")
					code, err = promptLine("验证码")
					if err != nil {
						output.Fatal(err)
					}
				}
				if code == "" {
					output.Fatal(fmt.Errorf("验证码不能为空"))
				}
				result, err = auth.LoginWithCode(c, country, phone, code)
			} else {
				if password == "" {
					password, err = promptPassword("密码")
					if err != nil {
						output.Fatal(err)
					}
				}
				if password == "" {
					output.Fatal(fmt.Errorf("密码不能为空"))
				}
				result, err = auth.LoginWithPassword(c, phone, password)
			}
			if err != nil {
				output.Fatal(err)
			}

			c2, err := client.NewFromSession()
			if err == nil {
				ents, cur, err := api.RefreshEnterprises(c2)
				if err == nil {
					_ = output.Print(map[string]any{
						"ok":          true,
						"home_url":    result.HomeURL,
						"account":     phone,
						"current_ent": cur,
						"ents":        ents,
					})
					return
				}
			}
			_ = output.Print(map[string]any{
				"ok":       true,
				"home_url": result.HomeURL,
				"account":  phone,
			})
		},
	}
	cmd.Flags().StringVar(&phone, "phone", "", "手机号或邮箱账号（省略则交互输入）")
	cmd.Flags().StringVar(&password, "password", "", "密码（省略则交互输入）")
	cmd.Flags().StringVar(&code, "code", "", "短信验证码（省略则交互选择并输入）")
	cmd.Flags().StringVar(&country, "country", "86", "国家码，默认 86")
	return cmd
}

func promptLine(label string) (string, error) {
	fmt.Fprintf(os.Stderr, "%s: ", label)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func promptPassword(label string) (string, error) {
	fmt.Fprintf(os.Stderr, "%s: ", label)
	pw, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(pw), nil
}

func promptLoginMode() (string, error) {
	fmt.Fprintln(os.Stderr, "请选择登录方式:")
	fmt.Fprintln(os.Stderr, "  1) 密码登录")
	fmt.Fprintln(os.Stderr, "  2) 短信验证码登录")
	for {
		choice, err := promptLine("请输入 1 或 2")
		if err != nil {
			return "", err
		}
		switch strings.TrimSpace(choice) {
		case "1", "password", "密码":
			return "password", nil
		case "2", "code", "sms", "验证码":
			return "code", nil
		default:
			fmt.Fprintln(os.Stderr, "无效选项，请重新输入。")
		}
	}
}

func authSendCodeCmd() *cobra.Command {
	var phone, country string
	cmd := &cobra.Command{
		Use:   "send-code",
		Short: "发送登录短信验证码",
		Run: func(cmd *cobra.Command, args []string) {
			if phone == "" {
				output.Fatal(fmt.Errorf("请指定 --phone"))
			}
			c, err := client.New()
			if err != nil {
				output.Fatal(err)
			}
			if err := auth.SendLoginCode(c, country, phone); err != nil {
				output.Fatal(err)
			}
			_ = output.Print(map[string]any{"ok": true, "message": "验证码已发送"})
		},
	}
	cmd.Flags().StringVar(&phone, "phone", "", "手机号")
	cmd.Flags().StringVar(&country, "country", "86", "国家码")
	return cmd
}

func authLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "退出登录并清除本地会话",
		Run: func(cmd *cobra.Command, args []string) {
			c, err := client.NewFromSession()
			if err != nil {
				_ = session.Clear()
				_ = output.Print(map[string]any{"ok": true})
				return
			}
			if err := auth.Logout(c); err != nil {
				output.Fatal(err)
			}
			_ = output.Print(map[string]any{"ok": true})
		},
	}
}

func authStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "查看登录状态",
		Run: func(cmd *cobra.Command, args []string) {
			s, err := session.Load()
			if err != nil {
				_ = output.Print(map[string]any{"logged_in": false, "error": err.Error()})
				return
			}
			_ = output.Print(map[string]any{
				"logged_in":   true,
				"account":     s.Account,
				"home_url":    s.HomeURL,
				"current_ent": s.CurrentEnt,
				"ents":        s.Ents,
				"updated_at":  s.UpdatedAt,
			})
		},
	}
}

func entCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ent",
		Short: "企业",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "列出可切换的企业",
		Run: func(cmd *cobra.Command, args []string) {
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			ents, cur, err := api.RefreshEnterprises(c)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(map[string]any{"current": cur, "ents": ents})
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "use [id|name]",
		Short: "切换当前企业（-1 或 团队版）",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			ent, err := api.UseEnterprise(c, args[0])
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(map[string]any{"ok": true, "current_ent": ent})
		},
	})
	return cmd
}

func projectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "项目",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "列出当前企业下的项目",
		Run: func(cmd *cobra.Command, args []string) {
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			list, err := api.ListProjects(c)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(map[string]any{
				"ent":      c.Session.CurrentEnt,
				"projects": list,
			})
		},
	})
	return cmd
}

func sprintCmd() *cobra.Command {
	var projectID string
	cmd := &cobra.Command{
		Use:     "sprint",
		Aliases: []string{"board"},
		Short:   "Sprint（看板）",
	}
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "列出项目下的 Sprint/看板",
		Run: func(cmd *cobra.Command, args []string) {
			if projectID == "" {
				output.Fatal(fmt.Errorf("请指定 --project <project_id>"))
			}
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			msg, err := api.ListBoards(c, projectID)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(msg)
		},
	}
	listCmd.Flags().StringVar(&projectID, "project", "", "项目 ID")
	cmd.AddCommand(listCmd)

	getCmd := &cobra.Command{
		Use:   "get [board_uuid|board_url]",
		Short: "获取 Sprint 结构（泳道/列）；可直接传 lg.team 看板链接",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			ref, err := parse.ParseBoardRef(args[0])
			if err != nil {
				output.Fatal(err)
			}
			ctx, err := api.LoadBoardContext(c, ref.BoardUUID)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(map[string]any{
				"board_uuid": ctx.BoardUUID,
				"board_id":   ctx.BoardID,
				"structure":  ctx.Structure,
				"lists":      jsonRawOrNil(ctx.Lists),
				"lanes":      jsonRawOrNil(ctx.Lanes),
			})
		},
	}
	cmd.AddCommand(getCmd)
	return cmd
}

func storyCmd() *cobra.Command {
	var sprint, user string
	var tags []string
	cmd := &cobra.Command{
		Use:   "story",
		Short: "Story（卡片/Task）",
	}
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "列出 Sprint 下 Story（可用 --user / --tag 筛选；--sprint 支持看板链接）",
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, err := requireBoardUUID(sprint)
			if err != nil {
				output.Fatal(err)
			}
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			ctx, stories, err := api.ListStoriesOpts(c, boardUUID, api.StoryListOptions{User: user, Tags: tags})
			if err != nil {
				output.Fatal(err)
			}
			out := map[string]any{
				"board_uuid":   ctx.BoardUUID,
				"board_id":     ctx.BoardID,
				"structure":    ctx.Structure,
				"current_user": ctx.CurrentUser,
				"count":        len(stories),
				"stories":      stories,
			}
			if user != "" {
				out["filter_user"] = user
				if _, resolved, err := api.ResolveBoardUser(user, ctx); err == nil {
					out["filter_user_resolved"] = resolved
				}
			}
			if len(tags) > 0 {
				out["filter_tags"] = tags
			}
			_ = output.Print(out)
		},
	}
	listCmd.Flags().StringVar(&sprint, "sprint", "", "看板 UUID 或 lg.team 看板链接")
	listCmd.Flags().StringVar(&user, "user", "", "按成员筛选：me / 用户 id / 昵称 / 邮箱")
	listCmd.Flags().StringArrayVar(&tags, "tag", nil, "按标签筛选（tag_name 子串或 tag_uuid；可多次，AND）")
	cmd.AddCommand(listCmd)

	getCmd := &cobra.Command{
		Use:   "get [task_id|uuid|name|story_url]",
		Short: "获取 Story 详情；可直接传含 sprint+story 的看板链接",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			target := strings.TrimSpace(args[0])
			boardUUID := ""
			taskKey := target

			// Full story URL: /board/go/{board}/{task}
			if ref, err := parse.ParseBoardRef(target); err == nil && ref.TaskUUID != "" {
				boardUUID = ref.BoardUUID
				taskKey = ref.TaskUUID
			} else {
				var err error
				boardUUID, err = requireBoardUUID(sprint)
				if err != nil {
					output.Fatal(fmt.Errorf("请指定 --sprint <board_uuid|url>，或传入含 story 的完整链接"))
				}
			}

			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			detail, err := api.GetStory(c, boardUUID, taskKey)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(detail)
		},
	}
	getCmd.Flags().StringVar(&sprint, "sprint", "", "看板 UUID 或 lg.team 看板链接（传入完整 story 链接时可省略）")
	cmd.AddCommand(getCmd)

	usersCmd := &cobra.Command{
		Use:   "users",
		Short: "列出 Sprint 看板成员（board_data.users）",
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, err := requireBoardUUID(sprint)
			if err != nil {
				output.Fatal(err)
			}
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			ctx, err := api.LoadBoardContext(c, boardUUID)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(map[string]any{
				"board_uuid":   ctx.BoardUUID,
				"current_user": ctx.CurrentUser,
				"count":        len(ctx.Users),
				"users":        ctx.Users,
			})
		},
	}
	usersCmd.Flags().StringVar(&sprint, "sprint", "", "看板 UUID 或 lg.team 看板链接")
	cmd.AddCommand(usersCmd)

	tagsCmd := &cobra.Command{
		Use:   "tags",
		Short: "列出 Sprint 下 Story 出现过的标签（去重）",
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, err := requireBoardUUID(sprint)
			if err != nil {
				output.Fatal(err)
			}
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			_, stories, err := api.ListStories(c, boardUUID)
			if err != nil {
				output.Fatal(err)
			}
			all := api.CollectStoryTags(stories)
			_ = output.Print(map[string]any{
				"board_uuid": boardUUID,
				"count":      len(all),
				"tags":       all,
			})
		},
	}
	tagsCmd.Flags().StringVar(&sprint, "sprint", "", "看板 UUID 或 lg.team 看板链接")
	cmd.AddCommand(tagsCmd)
	return cmd
}

func requireBoardUUID(sprintOrURL string) (string, error) {
	if strings.TrimSpace(sprintOrURL) == "" {
		return "", fmt.Errorf("请指定 --sprint <board_uuid|url>")
	}
	ref, err := parse.ParseBoardRef(sprintOrURL)
	if err != nil {
		return "", err
	}
	return ref.BoardUUID, nil
}

func jsonRawOrNil(b []byte) any {
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	return json.RawMessage(b)
}
