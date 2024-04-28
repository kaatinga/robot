package job

import (
	"context"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"

	"github.com/kaatinga/robot/internal/tool"
)

func Init() {
	initClientOnce.Do(func() {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: tool.GetOptions().GitHubToken},
		)

		ctx := context.Background()
		tc := oauth2.NewClient(ctx, ts)
		client = github.NewClient(tc)
	})
}
