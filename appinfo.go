package main

var AppVersion = "1.0.6"

const (
	AppName   = "游戏掉线监控"
	AppAuthor = "怀旧天龙CC科技"

	UpdateRepoOwner = "tlcc-tech"
	UpdateRepoName  = "tlbb-game-screen-monitor"
)

type AppInfo struct {
	Name    string `json:"name"`
	Author  string `json:"author"`
	Version string `json:"version"`
}
