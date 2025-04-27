package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sysproxy/sysproxy"

	"github.com/spf13/cobra"
)

var (
	server string
	bypass string
	pacUrl string
)

var cmd = &cobra.Command{
	Use:   "sysproxy",
	Short: "系统代理设置工具",
}

var sysCmd = &cobra.Command{
	Use:   "sys",
	Short: "设置系统代理",
	Run: func(cmd *cobra.Command, args []string) {
		err := sysproxy.SetProxy(server, bypass)
		if err != nil {
			fmt.Println("设置代理失败：", err)
			return
		}
		fmt.Println("代理设置成功")
	},
}

var pacCmd = &cobra.Command{
	Use:   "pac",
	Short: "设置 PAC 代理",
	Run: func(cmd *cobra.Command, args []string) {
		err := sysproxy.SetPac(pacUrl)
		if err != nil {
			fmt.Println("设置 PAC 代理失败：", err)
			return
		}
		fmt.Println("PAC 代理设置成功")
	},
}

var unsetCmd = &cobra.Command{
	Use:   "unset",
	Short: "取消代理设置",
	Run: func(cmd *cobra.Command, args []string) {
		err := sysproxy.DisableProxy()
		if err != nil {
			fmt.Println("取消代理设置失败：", err)
			return
		}
		fmt.Println("代理设置已取消")
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看当前代理设置",
	Run: func(cmd *cobra.Command, args []string) {
		status, err := sysproxy.QueryProxySettings()
		if err != nil {
			fmt.Println("查询代理设置失败：", err)
			return
		}
		statusJSON, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			fmt.Println("格式化 JSON 失败：", err)
			return
		}
		fmt.Println(string(statusJSON))
	},
}

func init() {
	cmd.AddCommand(sysCmd)
	cmd.AddCommand(pacCmd)
	cmd.AddCommand(unsetCmd)
	cmd.AddCommand(statusCmd)

	sysCmd.Flags().StringVarP(&server, "server", "s", "", "代理服务器地址")
	sysCmd.Flags().StringVarP(&bypass, "bypass", "b", "", "绕过地址")

	pacCmd.Flags().StringVarP(&pacUrl, "pacurl", "p", "", "pac 地址")
}

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
