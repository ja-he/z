package main

import (
	"os"
	"path"

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
	// expand env vars in config data
	configData = []byte(os.ExpandEnv(string(configData)))
	if readErr != nil {
		log.Warn().Err(readErr).Msg("could not read config file, assuming no config")
	} else {
		config := cfg.Cfg{}
		err := yaml.Unmarshal(configData, &config)
		if err != nil {
			log.Fatal().Err(err).Msg("could not parse config")
		} else {
			cfg.GlobalCfg = config
		}
	}

	parser := flags.NewParser(&cli.Opts, flags.Default)
	parser.SubcommandsOptional = false

	_, err := parser.Parse()
	if flags.WroteHelp(err) {
		os.Exit(0)
	} else if err != nil {
		log.Fatal().Err(err).Msg("some flag parsing error occurred")
	}
}
