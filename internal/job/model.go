package job

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/go-github/v60/github"
)

var client *github.Client

var initClientOnce sync.Once

type action byte

func (a action) RequiresContent() bool {
	switch a {
	case updateAction, createAction:
		return true
	default:
		return false
	}
}

func (a action) RequiresSHA() bool {
	switch a {
	case updateAction, deleteAction:
		return true
	default:
		return false
	}
}

const (
	updateAction action = iota
	deleteAction
	createAction
)

// loadTemplates loads all templates from templates directory. Returns a map of template name to template content.
func loadTemplates() (map[string][]byte, error) {
	var templates = make(map[string][]byte)
	// list all files in the `templates` directory
	err := filepath.WalkDir("templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		bytes, err := os.ReadFile(path)
		if err != nil {
			if d.IsDir() {
				return nil
			}
			return err
		}

		templates[d.Name()] = bytes
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking the path: %v", err)
	}

	return templates, nil
}

// GenerateRandomString generates a random string of a specified length.
//func GenerateRandomString(n int) ([]byte, error) {
//	bytes := make([]byte, n)
//	if _, err := rand.Read(bytes); err != nil {
//		return nil, err
//	}
//	return []byte(hex.EncodeToString(bytes)), nil
//}
