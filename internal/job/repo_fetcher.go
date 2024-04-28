package job

import (
	"context"
	"fmt"

	"github.com/google/go-github/v60/github"
	"github.com/kaatinga/robot/internal/color"
	"github.com/kaatinga/robot/internal/pretty"
)

func FetchAllGoRepos(ctx context.Context, j Job, repoJob func(context.Context, *github.Repository) error) error {
	scopePrinter := pretty.NewScopePrinter("")
	scopePrinter.Info("Fetching all Go repositories for user '%s'", j.User())

	// List all repositories for the authenticated user
	opt := &github.RepositoryListByAuthenticatedUserOptions{
		Sort:        "Updated",
		ListOptions: github.ListOptions{PerPage: 30},
	}

	for {
		repos, listRepos, err := client.Repositories.ListByAuthenticatedUser(ctx, opt)
		if err != nil {
			return fmt.Errorf("Error listing repositories: %v\n", err)
		}
		if listRepos.Rate.Remaining < 300 {
			scopePrinter.Info("Rate limit reached. Remaining Quota: %d", listRepos.Rate.Remaining)
			break
		} else {
			scopePrinter.Info("Remaining Quota: %d", listRepos.Rate.Remaining)
		}

		for _, repo := range repos {
			scopePrinter.Info("Processing repository '%s'", repo.GetName())
			j.Next() // Reset the branchCreated and other flags
			loopPrinter := pretty.NewScopePrinter("-")
			if skipRepo(ctx, repo, loopPrinter, j.User()) {
				continue
			}

			loopPrinter.Info("Golang package/project detected")

			if err := repoJob(ctx, repo); err != nil {
				return err
			}
		}

		if listRepos.NextPage == 0 {
			break
		}
		opt.Page = listRepos.NextPage
	}

	println()
	fmt.Println(color.Faint + "------- updateWorkflowFilesJob Finished -------" + color.Reset)
	if j.Counter() == 0 {
		scopePrinter.Info("No Pull Requests created in Go repositories by this job")
		return nil
	}

	scopePrinter.OK("%d Pull Requests created in Go repositories", j.Counter())
	scopePrinter.AddPrefix("--")
	for _, pr := range j.PRURLs() {
		scopePrinter.Info(pr)
	}

	return nil
}

func skipRepo(ctx context.Context, repo *github.Repository, loopPrinter pretty.ScopePrinter, user string) bool {
	if repo.GetFork() {
		loopPrinter.Skipped("Fork")
		return true
	}

	if repo.GetArchived() {
		loopPrinter.Skipped("Archived")
		return true
	}

	if repo.GetOwner().GetType() != "User" {
		loopPrinter.Skipped("Not a user repository: " + repo.GetOwner().GetType())
		return true
	}

	// Check for go.mod file in the repository's root
	_, _, resp, err := client.Repositories.GetContents(ctx, user, repo.GetName(), "go.mod", &github.RepositoryContentGetOptions{})
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			loopPrinter.Skipped("go.mod is not in the root directory")
		} else {
			loopPrinter.Error("Error getting contents: %v", err)
		}
		return true
	}

	return false
}
