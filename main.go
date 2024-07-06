package main

import (
	"context"
	"log"
	"os"

	"github.com/kaatinga/robot/internal/job"
	"github.com/kaatinga/robot/internal/pretty"
	"github.com/kaatinga/robot/internal/tool"
)

func main() {
	if err := tool.Init(); err != nil {
		log.Fatal(err)
	}

	job.Init()

	printer := pretty.NewScopePrinter("")
	user := "kaatinga"

	job1, err := job.NewUpdateWorkflowJob(user, true)
	if err != nil {
		printer.Error("Failure: %v", err)
		os.Exit(1)
	}
	err = job.FetchAllGoRepos(context.Background(), job1, job1.UpdateWorkflow)
	if err != nil {
		printer.Error("%v", err)
		os.Exit(1)
	}

	// job2 := job.NewDeleteOldRobotBranchesJob(user)
	// if err := job.FetchAllGoRepos(context.Background(), job2, job2.DeleteLeftRobotBranches); err != nil {
	// 	printer.Error("%v", err)
	// 	os.Exit(1)
	// }
}
