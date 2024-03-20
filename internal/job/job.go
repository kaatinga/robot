package job

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/go-github/v60/github"
	"github.com/kaatinga/robot/internal/pretty"
	"github.com/kaatinga/robot/internal/tool"
	"golang.org/x/oauth2"
)

type Job struct {
	User         string
	PRBranchName string
	baseBranch   string
	toMerge      bool

	// repo related fields
	branchCreated bool
	filesToUpdate map[string][]byte

	prURLs []string

	counter uint16
}

func (j *Job) skipRepo(ctx context.Context, repo *github.Repository, loopPrinter pretty.ScopePrinter) bool {
	if repo.GetFork() {
		loopPrinter.Skipped("Fork")
		return true
	}

	if repo.GetArchived() {
		loopPrinter.Skipped("Archived")
		return true
	}

	// Check for go.mod file in the repository's root
	_, _, resp, err := client.Repositories.GetContents(ctx, j.User, repo.GetName(), "go.mod", &github.RepositoryContentGetOptions{})
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			loopPrinter.Skipped("Repository is not a Golang package/project")
		} else {
			loopPrinter.Error("Error getting contents: %v", err)
		}
		return true
	}

	return false
}

func (j *Job) PRURLs() []string {
	return j.prURLs
}

func (j *Job) next() {
	j.branchCreated = false
	j.baseBranch = "main"
}

func NewJob(user, prBranchName string, toMerge bool) (*Job, error) {
	filesToUpdate, err := loadTemplates()
	if err != nil {
		return nil, fmt.Errorf("Error loading templates: %v\n", err)
	}

	if len(filesToUpdate) == 0 {
		return nil, errors.New("no templates found")
	}

	initClientOnce.Do(func() {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: tool.GetOptions().GitHubToken},
		)

		ctx := context.Background()
		tc := oauth2.NewClient(ctx, ts)
		client = github.NewClient(tc)
	})

	return &Job{
		User:          user,
		filesToUpdate: filesToUpdate,
		PRBranchName:  prBranchName,
		baseBranch:    "main",
		toMerge:       toMerge,
	}, nil
}

func (j *Job) Counter() uint16 {
	return j.counter
}

func (j *Job) FetchAllGoRepos(ctx context.Context, repoJob func(context.Context, *github.Repository) (resultAction, error)) error {
	scopePrinter := pretty.NewScopePrinter("")
	scopePrinter.Info("Fetching all Go repositories for user '%s'", j.User)
	// List all repositories for the authenticated user
	opt := &github.RepositoryListByUserOptions{
		Type:        "owner",
		Sort:        "Updated",
		ListOptions: github.ListOptions{PerPage: 30},
	}

	for {
		repos, listRepos, err := client.Repositories.ListByUser(ctx, j.User, opt)
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
			j.next() // Reset the branchCreated and other flags
			loopPrinter := pretty.NewScopePrinter("-")
			if j.skipRepo(ctx, repo, loopPrinter) {
				continue
			}

			loopPrinter.Info("Golang package/project detected")

			result, err := repoJob(ctx, repo)
			if err = j.finalizePR(ctx, err, result, repo); err != nil {
				return err
			}

			//if j.counter >= 1 {
			//	scopePrinter.Info("Limit of 1 PRs reached")
			//	break
			//}
		}

		if listRepos.NextPage == 0 {
			break
		}
		opt.Page = listRepos.NextPage
	}

	return nil
}

func (j *Job) UpdateWorkflow(ctx context.Context, repo *github.Repository) (result resultAction, err error) {
	printer := pretty.NewScopePrinter("---")

	// Get the current contents of .github/workflows
	var contents []*github.RepositoryContent
	_, contents, _, err = client.Repositories.GetContents(ctx, j.User, repo.GetName(), ".github/workflows", &github.RepositoryContentGetOptions{})
	if err != nil {
		var errorResponse *github.ErrorResponse
		if errors.As(err, &errorResponse) && errorResponse.Response.StatusCode == 404 {
			printer.Skipped("No .github/workflows directory found.")
			err = nil
		} else {
			err = fmt.Errorf("Error getting contents: %v\n", err)
		}
		return
	}

	printer.Info("Found %d files in .github/workflows", len(contents))

	var filesToCreate = make(map[string]struct{})
	for filePath := range j.filesToUpdate {
		filesToCreate[filePath] = struct{}{}
	}
	for _, content := range contents {
		printer.Info("Processing file '%s'", content.GetName())
		// Check if the current file is one of the files to update
		if newContent, found := j.filesToUpdate[content.GetName()]; found {
			delete(filesToCreate, content.GetName())

			var updateResult resultAction
			updateResult, err = j.createBranchAndDo(ctx, repo.GetName(), content.GetPath(), newContent, updateAction)
			if err != nil {
				err = fmt.Errorf("unable to update '%s': %v", content.GetName(), err)
				return
			}
			result.add(updateResult)
		} else {
			// Delete the file since it's not one of the files to keep
			var deleteResult resultAction
			deleteResult, err = j.createBranchAndDo(ctx, repo.GetName(), content.GetPath(), nil, deleteAction)
			if err != nil {
				err = fmt.Errorf("unable to delete '%s': %v", content.GetName(), err)
				return
			}

			result.add(deleteResult)
		}
	}

	for filePath, fileContent := range j.filesToUpdate {
		if _, found := filesToCreate[filePath]; !found {
			continue
		}

		var createResult resultAction
		createResult, err = j.createBranchAndDo(ctx, repo.GetName(), ".github/workflows/"+filePath, fileContent, createAction)
		if err != nil {
			err = fmt.Errorf("unable to create '%s': %v", filePath, err)
			return
		}

		result.add(createResult)
	}

	return
}

func (j *Job) finalizePR(ctx context.Context, err error, result resultAction, repo *github.Repository) error {
	printer := pretty.NewScopePrinter("-")
	switch {
	case err != nil || (j.branchCreated && !result.Changed()):
		_, delErr := client.Git.DeleteRef(ctx, j.User, repo.GetName(), "refs/heads/"+j.PRBranchName)
		if delErr != nil {
			if err != nil {
				err = fmt.Errorf("error deleting branch '%s': %w: %s", j.PRBranchName, delErr, err)
			} else {
				err = fmt.Errorf("error deleting branch '%s': %v", j.PRBranchName, delErr)
			}
		} else {
			printer.Info("No updates made. Branch '%s' deleted.", j.PRBranchName)
		}
	case result.Changed():
		pr := &github.NewPullRequest{
			Title:               github.String("Update Workflow YAML files"),
			Head:                github.String(j.PRBranchName),
			Base:                github.String(j.baseBranch),
			Body:                github.String("This PR updates workflow files."),
			MaintainerCanModify: github.Bool(true),
		}
		var prResponse *github.PullRequest
		prResponse, _, err = client.PullRequests.Create(ctx, j.User, repo.GetName(), pr)
		if err != nil {
			return fmt.Errorf("error creating pull request: %v", err)
		}

		// print the PR URL
		printer.Info("Pull request created: %s", prResponse.GetHTMLURL())
		j.prURLs = append(j.prURLs, prResponse.GetHTMLURL())
		j.counter++

		if j.toMerge {
			_, _, err = client.PullRequests.Merge(ctx, j.User, repo.GetName(), prResponse.GetNumber(), "Merging PR", nil)
			if err != nil {
				return fmt.Errorf("error merging pull request: %v", err)
			}
		}
	}

	return err
}

// createBranchAndDo creates a branch, updates a file with a random string, and creates a PR.
func (j *Job) createBranchAndDo(ctx context.Context, repo, filePath string, content []byte, action action) (result resultAction, err error) {
	printer := pretty.NewScopePrinter("-----")

	if action.RequiresContent() && len(content) == 0 {
		err = fmt.Errorf("content cannot be empty upon updating a file")
		return
	}

	// Step 1: Get content of the file
	getContentOptions := new(github.RepositoryContentGetOptions)
	if j.branchCreated {
		getContentOptions.Ref = j.PRBranchName
	}

	var file *github.RepositoryContent
	if action.RequiresSHA() {
		file, _, _, err = client.Repositories.GetContents(ctx, j.User, repo, filePath, getContentOptions)
		if err != nil {
			err = fmt.Errorf("error retrieving file: %v", err)
			return
		}
	}

	var oldContent string
	if action == updateAction {
		oldContent, err = file.GetContent()
		if err != nil {
			err = fmt.Errorf("error getting file content: %v", err)
			return
		}

		if string(content) == oldContent {
			printer.Skipped("Content is the same.")
			result.add(resultSkipped)
			return
		}
	}

	if !j.branchCreated {
		if err = j.createBranch(ctx, repo); err != nil {
			return
		}

		printer.OK("Branch '%s' created", j.PRBranchName)
		j.branchCreated = true
	}

	// Step 4: Update Workflow file
	switch action {
	case updateAction:
		var updateResult resultAction
		updateResult, err = j.updateFile(ctx, repo, filePath, content, file, oldContent)
		result.add(updateResult)
	case deleteAction:
		err = j.deleteFile(ctx, repo, filePath, file)
		result.add(resultDeleted)
	case createAction:
		err = j.createFile(ctx, repo, filePath, content)
		result.add(resultCreated)
	default:
		err = fmt.Errorf("unknown action: %v", action)
	}

	for _, r := range result.PrintAll() {
		printer.OK(r)
	}

	// if it is test.yml, add badge to README.md
	//if file.GetName() == "test.yml" {
	//	j.addBadge(ctx, repo, printer)
	//}

	return
}

func (j *Job) addBadge(ctx context.Context, repo string, printer pretty.ScopePrinter) {
	const badgeTemplate = `[![Tests](https://github.com/%s/%s/actions/workflows/test.yml/badge.svg?branch=%s)](https://github.com/%[1]s/%[2]s/actions/workflows/test.yml)`
	badge := fmt.Sprintf(badgeTemplate, j.User, "luna", j.baseBranch)
	// Step 5: Read README.md
	readmeFile, _, _, err := client.Repositories.GetContents(ctx, j.User, repo, "README.md", &github.RepositoryContentGetOptions{Ref: j.baseBranch})
	if err != nil {
		printer.Error("error getting README.md: %v", err)
	}

	// Step 6: Update README.md
	var readmeContent string
	readmeContent, err = readmeFile.GetContent()
	if err != nil {
		printer.Error("error getting README.md content: %v", err)
	}

	readmeContent = strings.Replace(readmeContent, badge+"\n", "", -1)
	badge = fmt.Sprintf(badgeTemplate, j.User, repo, j.baseBranch)
	readmeContent = badge + "\n" + readmeContent

	updateResult, err := j.updateFile(ctx, repo, "README.md", []byte(readmeContent), readmeFile, readmeContent)
	if err != nil {
		printer.Error("error updating README.md: %v", err)
	}

	if updateResult.Updated() {
		printer.OK("Added badge to README.md")
	}
}

func (j *Job) createBranch(ctx context.Context, repo string) error {
	// Step 2: Get the latest commit SHA of the base branch
	baseRef, _, err := client.Git.GetRef(ctx, j.User, repo, "refs/heads/"+j.baseBranch)
	if err != nil {
		var errorResponse *github.ErrorResponse
		if errors.As(err, &errorResponse) && errorResponse.Response.StatusCode == 404 && j.baseBranch != "master" {
			j.baseBranch = "master"
			baseRef, _, err = client.Git.GetRef(ctx, j.User, repo, "refs/heads/"+j.baseBranch)
		}

		if err != nil {
			return fmt.Errorf("error getting base branch ref: %w", err)
		}
	}

	// Step 3: Create a new branch from the latest commit of the base branch
	newRef := &github.Reference{
		Ref:    github.String("refs/heads/" + j.PRBranchName),
		Object: &github.GitObject{SHA: baseRef.Object.SHA},
	}
	_, _, err = client.Git.CreateRef(ctx, j.User, repo, newRef)
	if err != nil {
		return fmt.Errorf("error creating new branch: %w", err)
	}

	return nil
}

func (j *Job) updateFile(ctx context.Context, repo string, filePath string, content []byte, file *github.RepositoryContent, oldContent string) (result resultAction, err error) {
	updateOpts := &github.RepositoryContentFileOptions{
		Message: github.String(fmt.Sprintf("Update %s", filePath)),
		Content: content,
		Branch:  github.String(j.PRBranchName),
		SHA:     file.SHA,
	}

	_, _, err = client.Repositories.UpdateFile(ctx, j.User, repo, filePath, updateOpts)
	if err != nil {
		err = fmt.Errorf("error updating file: %v", err)
		return
	}

	// Verify the file was Updated
	// Retrieve the file again to check the new content
	var updatedFileContent *github.RepositoryContent
	updatedFileContent, _, _, err = client.Repositories.GetContents(ctx, j.User, repo, filePath, &github.RepositoryContentGetOptions{Ref: j.PRBranchName})
	if err != nil {
		err = fmt.Errorf("error retrieving Updated file: %v", err)
		return
	}

	// Decode the content from base64
	var decodedContent string
	decodedContent, err = updatedFileContent.GetContent()
	if err != nil {
		err = fmt.Errorf("error decoding Updated file content: %v", err)
		return
	}

	// Check if the Updated content matches the expected content
	if decodedContent != oldContent {
		result.add(resultUpdated)
	}

	return
}

func (j *Job) deleteFile(ctx context.Context, repo string, filePath string, file *github.RepositoryContent) error {
	opts := &github.RepositoryContentFileOptions{
		Message: github.String(fmt.Sprintf("Delete %s", filePath)),
		SHA:     file.SHA,
		Branch:  github.String(j.PRBranchName),
	}

	_, _, err := client.Repositories.DeleteFile(ctx, j.User, repo, filePath, opts)
	if err != nil {
		return fmt.Errorf("error deleting file: %v", err)
	}

	return nil
}

func (j *Job) createFile(ctx context.Context, repo string, filePath string, content []byte) error {
	opts := &github.RepositoryContentFileOptions{
		Message: github.String(fmt.Sprintf("Create %s", filePath)),
		Content: content,
		Branch:  github.String(j.PRBranchName),
	}

	_, _, err := client.Repositories.CreateFile(ctx, j.User, repo, filePath, opts)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}

	return nil
}
