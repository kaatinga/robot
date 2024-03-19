package main

import (
	"context"
	"fmt"
	"github.com/kaatinga/robot/internal/color"
	"github.com/kaatinga/robot/internal/job"
	"github.com/kaatinga/robot/internal/pretty"
	"log"
	"os"
	"time"

	"github.com/kaatinga/robot/internal/tool"
)

const branchSafeTimeFormat = `2006-01-02T150405Z0700`

func main() {
	if err := tool.Init(); err != nil {
		log.Fatal(err)
	}

	jobBranch := "robot-works-" + time.Now().Format(branchSafeTimeFormat)
	printer := pretty.NewScopePrinter("")
	j, err := job.NewJob("kaatinga", jobBranch)
	if err != nil {
		printer.Error("Failure: %v", err)
		os.Exit(1)
	}
	err = j.FetchAllGoRepos(context.Background(), j.UpdateWorkflow)
	println()
	fmt.Println(color.Faint + "------- Finished -------" + color.Reset)
	if err != nil {
		printer.Error("%v", err)
		os.Exit(1)
	}

	printer.OK("%d Pull Requests created in Go repositories", j.Counter())
}
