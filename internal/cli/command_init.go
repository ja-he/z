package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"z/internal/cfg"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type InitCommand struct {
	Reinitialize bool `long:"reinitialize" description:"Check and update remote URLs for existing repositories"`
}

// normalizeGitURL removes credentials and normalizes a git URL for comparison
func normalizeGitURL(rawURL string) (string, error) {
	// Handle SSH URLs (git@github.com:user/repo.git)
	sshPattern := regexp.MustCompile(`^([^@]+)@([^:]+):(.+)$`)
	if sshPattern.MatchString(rawURL) {
		return rawURL, nil // SSH URLs don't contain credentials in the URL itself
	}

	// Handle HTTPS URLs that might contain credentials
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		// If it fails to parse, it might be a git protocol URL like git://
		return rawURL, nil
	}

	// Remove credentials from HTTPS URLs
	if parsedURL.Scheme == "https" || parsedURL.Scheme == "http" {
		parsedURL.User = nil
		return parsedURL.String(), nil
	}

	return rawURL, nil
}

// urlsMatch compares two git URLs, ignoring credentials
func urlsMatch(url1, url2 string) bool {
	normalized1, err1 := normalizeGitURL(url1)
	normalized2, err2 := normalizeGitURL(url2)

	if err1 != nil || err2 != nil {
		// If we can't normalize, fall back to direct comparison
		return url1 == url2
	}

	return normalized1 == normalized2
}

// hasUnpushedCommits checks if a repository has commits that haven't been pushed
func hasUnpushedCommits(repoPath string) (bool, error) {
	// Check if there's an upstream branch
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	cmd.Dir = repoPath
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = nil // Suppress error output

	if err := cmd.Run(); err != nil {
		// No upstream branch configured, so nothing to push
		return false, nil
	}

	// Check if local is ahead of remote
	cmd = exec.Command("git", "rev-list", "--count", "@{u}..HEAD")
	cmd.Dir = repoPath
	out.Reset()
	cmd.Stdout = &out
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return false, err
	}

	count := strings.TrimSpace(out.String())
	return count != "0" && count != "", nil
}

// getCurrentRemoteURL gets the current remote URL for the 'origin' remote
func getCurrentRemoteURL(repoPath string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = repoPath
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return strings.TrimSpace(out.String()), nil
}

// updateRemoteURL updates the remote URL for the 'origin' remote
func updateRemoteURL(repoPath, newURL string) error {
	cmd := exec.Command("git", "remote", "set-url", "origin", newURL)
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (c *InitCommand) Execute(_ []string) error {
	// Check if config file exists, create boilerplate if not
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine user home directory: %w", err)
	}
	configPath := path.Join(homeDir, ".config/z.yml")
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
			// Remote K - clone from URL or check/update remote
			if _, err := os.Stat(k.Path); err == nil {
				// K already exists
				if c.Reinitialize {
					// Check if it's a git repo
					gitDir := path.Join(k.Path, ".git")
					if _, err := os.Stat(gitDir); err == nil {
						// It's a git repo, check and update remote URL if needed
						log.Info().Str("K", id).Msg("checking remote URL")

						currentURL, err := getCurrentRemoteURL(k.Path)
						if err != nil {
							log.Warn().Str("K", id).Err(err).Msg("could not get current remote URL, skipping")
							continue
						}

						if !urlsMatch(currentURL, k.URL) {
							log.Warn().
								Str("K", id).
								Str("current", currentURL).
								Str("configured", k.URL).
								Msg("remote URL mismatch detected")

							// Check for unpushed commits
							hasUnpushed, err := hasUnpushedCommits(k.Path)
							if err != nil {
								log.Warn().Str("K", id).Err(err).Msg("could not check for unpushed commits")
							} else if hasUnpushed {
								log.Warn().Str("K", id).Msg("repository has unpushed commits, but updating remote URL anyway")
							}

							// Update the remote URL
							log.Info().Str("K", id).Str("new-url", k.URL).Msg("updating remote URL")
							if err := updateRemoteURL(k.Path, k.URL); err != nil {
								log.Error().Str("K", id).Err(err).Msg("failed to update remote URL")
							} else {
								log.Info().Str("K", id).Msg("remote URL updated successfully")
							}
						} else {
							log.Info().Str("K", id).Msg("remote URL matches, no update needed")
						}
					} else {
						log.Warn().Str("K", id).Msg("directory exists but is not a git repository")
					}
				} else {
					log.Info().Str("K", id).Str("path", k.Path).Msg("K already exists, skipping clone")
				}
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
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine user home directory: %w", err)
	}
	miscPath := path.Join(homeDir, "notes", "misc")

	colorEnabled := true
	config := cfg.Cfg{
		Settings: cfg.Settings{
			Color:          &colorEnabled, // Enable colored output by default
			VerbosityLevel: "info",        // Default log level
		},
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
# Settings are application-wide settings:
#   color: Enable colored output in logs (default: true, set to false to disable)
#   verbosity-level: Log verbosity level - trace, debug, info, warn, error, fatal, panic (default: info)
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
