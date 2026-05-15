package internal

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds optional values from ~/.daywrap.yml.
// All fields are optional; missing file is not an error.
type Config struct {
	Jira struct {
		BaseURL string `yaml:"base_url"`
	} `yaml:"jira"`
	Repos         []string `yaml:"repos"`
	TicketPattern string   `yaml:"ticket_pattern"`
}

// DefaultTicketPattern matches common ticket formats when none is configured.
const DefaultTicketPattern = `(ENG|PROJ|PLAT)-\d+`

// LoadConfig reads ~/.daywrap.yml. A missing file returns an empty Config.
func LoadConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return &Config{}, nil
	}
	data, err := os.ReadFile(filepath.Join(home, ".daywrap.yml"))
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
