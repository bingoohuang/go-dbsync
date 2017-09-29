package main

import (
	"../myutil"
	"github.com/BurntSushi/toml"
	"os"
)

type goIpAllowConfig struct {
	ContextPath         string
	Envs                []string // 环境
	ListenPort          int      // 监听端口
	UpdateFirewallShell string   // 更新防火墙IP脚本
	WxCorpId            string
	WxCorpSecret        string
	WxAgentId           int64
	EncryptKey          string
	CookieName          string
	RedirectUri         string
}

func readIpAllowConfig() goIpAllowConfig {
	fpath := "go-ip-allow.toml"
	if len(os.Args) > 1 {
		fpath = os.Args[1]
	}

	ipAllowConfig := goIpAllowConfig{}
	if _, err := toml.DecodeFile(fpath, &ipAllowConfig); err != nil {
		myutil.CheckErr(err)
	}

	return ipAllowConfig
}

var g_config goIpAllowConfig

func init() {
	g_config = readIpAllowConfig()
}
