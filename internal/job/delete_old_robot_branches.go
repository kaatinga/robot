package job

import (
	"context"
	"github.com/google/go-github/v60/github"
	"github.com/kaatinga/robot/internal/pretty"
	"strings"
)

type deleteOldRobotBranchesJob struct {
	user string
}

func NewDeleteOldRobotBranchesJob(user string) *deleteOldRobotBranchesJob {
	return &deleteOldRobotBranchesJob{
		user: user,
	}
}

func (j *deleteOldRobotBranchesJob) User() string {
	return j.user
}

func (j *deleteOldRobotBranchesJob) Next() {

}

func (j *deleteOldRobotBranchesJob) Counter() uint16 {
	return 0
}

func (j *deleteOldRobotBranchesJob) PRURLs() []string {
	return nil
}

func (j *deleteOldRobotBranchesJob) DeleteLeftRobotBranches(ctx context.Context, repo *github.Repository) error {
	printer := pretty.NewScopePrinter("---")

	// get all branches
	opts := &github.BranchListOptions{ListOptions: github.ListOptions{PerPage: 100}}
	branches, _, err := client.Repositories.ListBranches(ctx, j.User(), repo.GetName(), opts)
	if err != nil {
		return err
	}

	// delete all branches with the prefix "branchPrefix"
	for _, branch := range branches {
		if strings.Contains(branch.GetName(), branchPrefix) {
			_, err := client.Git.DeleteRef(ctx, j.User(), repo.GetName(), "heads/"+branch.GetName())
			if err != nil {
				return err
			}

			printer.OK("Branch '%s' deleted", branch.GetName())
		}
	}

	return nil
}
