package main

var AppVersion = "1.4.0"

const (
	AppName   = "怀旧天龙CC科技"
	AppAuthor = "怀旧天龙CC科技"

	UpdateRepoOwner = "tlcc-tech"
	UpdateRepoName  = "tlbb-game-screen-monitor"
)

type AppInfo struct {
	Name    string `json:"name"`
	Author  string `json:"author"`
	Version string `json:"version"`
	RepoURL string `json:"repoUrl"`
}
