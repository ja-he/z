package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"

	"z/internal/cfg"
	"z/internal/cli"
)

// parseLogLevel converts a log level string to a zerolog.Level
func parseLogLevel(levelStr string) zerolog.Level {
	switch strings.ToLower(levelStr) {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel // default to info
	}
}

func main() {
	// Initialize logger with colored output by default
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal().Err(err).Msg("could not determine user home directory")
	}
	configPath := path.Join(homeDir, ".config/z.yml")
	configData, readErr := os.ReadFile(configPath)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			log.Warn().Str("path", configPath).Msg("config file not found, assuming no config (use 'z init' to set up)")
		} else {
			log.Warn().Err(readErr).Str("path", configPath).Msg("could not read config file, assuming no config")
		}
	} else {
		config := cfg.Cfg{}
		err := yaml.Unmarshal(configData, &config)
		if err != nil {
			log.Fatal().Err(err).Str("path", configPath).Msg("could not parse config file - check YAML syntax")
		} else {
			cfg.GlobalCfg = config
			for id, k := range cfg.GlobalCfg.Ks {
				cfg.GlobalCfg.Ks[id] = cfg.K{
					Path: os.ExpandEnv(k.Path),
					URL:  os.ExpandEnv(k.URL),
				}
			}

			// Reconfigure logger based on settings

			// Set log level (default: info)
			logLevel := zerolog.InfoLevel
			if cfg.GlobalCfg.Settings.VerbosityLevel != "" {
				logLevel = parseLogLevel(cfg.GlobalCfg.Settings.VerbosityLevel)
			}
			zerolog.SetGlobalLevel(logLevel)

			// Set color output
			// If color is not explicitly set (nil), default to true (colored output)
			// To disable colors, users must explicitly set color: false
			colorEnabled := true // default
			if cfg.GlobalCfg.Settings.Color != nil {
				colorEnabled = *cfg.GlobalCfg.Settings.Color
			}
			if !colorEnabled {
				// Disable colored output
				log.Logger = log.Output(os.Stderr)
			}

			log.Debug().Int("ks", len(cfg.GlobalCfg.Ks)).Int("blueprints", len(cfg.GlobalCfg.Blueprints)).Msg("loaded config")
		}
	}

	parser := flags.NewParser(&cli.Opts, flags.Default)
	parser.Usage = "[OPTIONS] COMMAND [ARGUMENTS]"
	parser.ShortDescription = "A git-centric note management system"
	parser.LongDescription = `z is a simple, opinionated note management tool built around git repositories (Ks).

It provides commands for creating, finding, syncing, and managing notes across multiple knowledge bases.

Configuration is read from ~/.config/z.yml which defines your Ks (knowledge bases) and blueprints.`

	parser.CompletionHandler = func(items []flags.Completion) {
		suggestions := []string{}
		if len(items) > 0 {
			for _, item := range items {
				if len(item.Item) > 1 {
					suggestions = append(suggestions, item.Item)
				}
			}
		} else if len(os.Args) > 2 {

			switch os.Args[1] {

			case "create":
				switch len(os.Args) {
				case 3: // complete K
					for kID := range cfg.GlobalCfg.Ks {
						suggestions = append(suggestions, kID)
					}
				case 5: // complete blueprint
					for bID := range cfg.GlobalCfg.Blueprints {
						suggestions = append(suggestions, bID)
					}
				}

			}

		}
		for _, suggestion := range suggestions {
			if strings.HasPrefix(suggestion, os.Args[len(os.Args)-1]) {
				fmt.Println(suggestion)
			}
		}
		os.Exit(0)
	}
	parser.SubcommandsOptional = false

	_, err = parser.Parse()
	if err != nil {
		if flags.WroteHelp(err) {
			os.Exit(0)
		}

		// Check for specific error types using errors.As for better error handling
		var flagsErr *flags.Error
		if errors.As(err, &flagsErr) {
			switch flagsErr.Type {
			case flags.ErrHelp:
				// Help was requested, exit cleanly
				os.Exit(0)
			case flags.ErrUnknownFlag:
				fmt.Fprintf(os.Stderr, "Error: Unknown flag: %v\n", err)
				fmt.Fprintf(os.Stderr, "Run 'z --help' for usage information\n")
				os.Exit(1)
			case flags.ErrUnknownCommand:
				fmt.Fprintf(os.Stderr, "Error: Unknown command: %v\n", err)
				fmt.Fprintf(os.Stderr, "Run 'z --help' to see available commands\n")
				os.Exit(1)
			case flags.ErrExpectedArgument:
				fmt.Fprintf(os.Stderr, "Error: Expected argument: %v\n", err)
				os.Exit(1)
			case flags.ErrRequired:
				fmt.Fprintf(os.Stderr, "Error: Required option missing: %v\n", err)
				os.Exit(1)
			default:
				// Generic error with better formatting
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				fmt.Fprintf(os.Stderr, "Run 'z --help' for usage information\n")
				os.Exit(1)
			}
		}

		// Non-flags error (e.g., from command execution)
		log.Fatal().Err(err).Msg("command execution failed")
	}
}
