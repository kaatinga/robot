package tool

import (
	"fmt"
	"github.com/kaatinga/settings"
)

type Options struct {
	//GitHubAPIKey string `env:"GITHUB_API_KEY" required:"true"`
	//OpenAIKey    string `env:"OPENAI_API_KEY" required:"true"`
	GitHubToken string `env:"GITHUB_TOKEN" required:"true"`
}

var toolSettings = &Options{}

func Init() error {
	// Load environment variables
	if err := settings.LoadSettings(toolSettings); err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	return nil
}

func GetOptions() *Options {
	return toolSettings
}
