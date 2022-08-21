package main

import (
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

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	configData, readErr := os.ReadFile(path.Join(
		os.Getenv("HOME"),
		".config/z.yml",
	))
	if readErr != nil {
		log.Warn().Err(readErr).Msg("could not read config file, assuming no config")
	} else {
		config := cfg.Cfg{}
		err := yaml.Unmarshal(configData, &config)
		if err != nil {
			log.Fatal().Err(err).Msg("could not parse config")
		} else {
			cfg.GlobalCfg = config
			for id, k := range cfg.GlobalCfg.Ks {
				cfg.GlobalCfg.Ks[id] = cfg.K{
					Path: os.ExpandEnv(k.Path),
					URL:  os.ExpandEnv(k.URL),
				}
			}
		}
	}

	parser := flags.NewParser(&cli.Opts, flags.Default)
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

	_, err := parser.Parse()
	if flags.WroteHelp(err) {
		os.Exit(0)
	} else if err != nil {
		log.Fatal().Err(err).Msg("some error occurred")
	}
}
