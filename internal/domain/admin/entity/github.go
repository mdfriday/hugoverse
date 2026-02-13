package entity

import "github.com/mdfriday/hugoverse/internal/domain/admin/valueobject"

// GitHub GitHub 配置实体
type GitHub struct {
	Conf *valueobject.Config
}

// HookSecret 返回 GitHub Webhook Secret
func (g *GitHub) HookSecret() string {
	if g.Conf == nil {
		return ""
	}
	return g.Conf.GithubHookSecret
}

// GithubToken 返回 GitHub Personal Access Token
func (g *GitHub) GithubToken() string {
	if g.Conf == nil {
		return ""
	}
	return g.Conf.GithubToken
}

// TargetRepository 返回目标仓库全名
func (g *GitHub) TargetRepository() string {
	if g.Conf == nil {
		return ""
	}
	if g.Conf.GithubTargetRepo == "" {
		// 默认仓库
		return "mdfriday/obsidian-friday-plugin"
	}
	return g.Conf.GithubTargetRepo
}

