package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"z/internal/cfg"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type InitCommand struct{}

func (_ *InitCommand) Execute(_ []string) error {
	// Check if config file exists, create boilerplate if not
	configPath := path.Join(os.Getenv("HOME"), ".config/z.yml")
	if _, err := os.Stat(configPath); errors.Is(err, fs.ErrNotExist) {
		log.Info().Str("path", configPath).Msg("config file not found, creating boilerplate config")

		if err := createBoilerplateConfig(configPath); err != nil {
			return fmt.Errorf("failed to create boilerplate config: %w", err)
		}

		log.Info().Str("path", configPath).Msg("created boilerplate config")

		// Reload the config
		configData, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read newly created config: %w", err)
		}

		config := cfg.Cfg{}
		if err := yaml.Unmarshal(configData, &config); err != nil {
			return fmt.Errorf("failed to parse newly created config: %w", err)
		}

		// Expand environment variables in paths
		for id, k := range config.Ks {
			config.Ks[id] = cfg.K{
				Path: os.ExpandEnv(k.Path),
				URL:  os.ExpandEnv(k.URL),
			}
		}

		cfg.GlobalCfg = config
	}

	log.Info().Msg("initializing Ks")

	config := cfg.GlobalCfg
	log.Debug().Interface("config", config).Msg("using global config")

	if len(config.Ks) == 0 {
		log.Warn().Msg("no Ks configured")
		return nil
	}

	for id, k := range config.Ks {
		if k.URL != "" {
			// Remote K - clone from URL
			if _, err := os.Stat(k.Path); err == nil {
				log.Info().Str("K", id).Str("path", k.Path).Msg("K already exists, skipping clone")
				continue
			}

			log.Info().Str("K", id).Str("url", k.URL).Msg("cloning remote K")
			clone := exec.Command("git", "clone", k.URL, k.Path)
			clone.Stdout, clone.Stderr, clone.Stdin = os.Stdout, os.Stderr, os.Stdin
			err := clone.Run()
			if err != nil {
				log.Error().
					Err(err).
					Strs("args", clone.Args).
					Str("K", id).
					Msg("error executing clone command")
			}
		} else {
			// Local K - initialize as git repo
			if _, err := os.Stat(k.Path); err == nil {
				log.Info().Str("K", id).Str("path", k.Path).Msg("K directory already exists, checking git status")

				// Check if it's already a git repo
				gitDir := path.Join(k.Path, ".git")
				if _, err := os.Stat(gitDir); err == nil {
					log.Info().Str("K", id).Msg("already a git repository")
					continue
				}
			} else {
				// Create the directory
				log.Info().Str("K", id).Str("path", k.Path).Msg("creating K directory")
				if err := os.MkdirAll(k.Path, 0755); err != nil {
					log.Error().Err(err).Str("K", id).Msg("failed to create K directory")
					continue
				}
			}

			// Initialize as git repo
			log.Info().Str("K", id).Str("path", k.Path).Msg("initializing local K as git repository")
			gitInit := exec.Command("git", "init")
			gitInit.Dir = k.Path
			gitInit.Stdout, gitInit.Stderr, gitInit.Stdin = os.Stdout, os.Stderr, os.Stdin
			if err := gitInit.Run(); err != nil {
				log.Error().Err(err).Str("K", id).Msg("failed to initialize git repository")
				continue
			}

			log.Info().Str("K", id).Msg("initialized local K successfully")
		}
	}

	return nil
}

func createBoilerplateConfig(configPath string) error {
	// Ensure the config directory exists
	configDir := path.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create boilerplate config with a local 'misc' K
	miscPath := path.Join(os.Getenv("HOME"), "notes", "misc")

	config := cfg.Cfg{
		Ks: map[string]cfg.K{
			"misc": {
				Path: miscPath,
				URL:  "", // Local-only K
			},
		},
		Blueprints: map[string]cfg.Blueprint{
			"note": {
				Subdir: "{{.Name}}",
				Templates: map[string]string{
					"note.md": "# {{.Name}}\n\nCreated: {{.Today}}\n\n",
				},
				Open:    "nvim note.md",
				View:    "",
				Post:    []string{},
				Sources: []string{"note.md"},
				Objects: []string{},
			},
		},
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	// Add a helpful comment at the top
	commentedYAML := `# z configuration file
#
# Ks are knowledge bases - directories containing your notes
# Each K can be:
#   - Remote: Has a 'url' field, will be cloned and synced via git
#   - Local: No 'url' field, managed locally only
#
# Example remote K:
#   work:
#     path: ~/notes/work
#     url: git@github.com:user/work-notes.git
#
# Blueprints are templates for creating new notes

` + string(yamlData)

	// Write to file
	if err := os.WriteFile(configPath, []byte(commentedYAML), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
