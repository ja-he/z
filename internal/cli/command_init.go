package cli

import (
	"os"
	"os/exec"
	"z/internal/cfg"

	"github.com/rs/zerolog/log"
)

type InitCommand struct{}

func (_ *InitCommand) Execute(_ []string) error {
	log.Info().Msg("initializing Ks")

	config := cfg.GlobalCfg
	log.Debug().Interface("config", config).Msg("using global config")

	for _, k := range config.Ks {
		clone := exec.Command("git", "clone", k.URL, k.Path)
		clone.Stdout, clone.Stderr, clone.Stdin = os.Stdout, os.Stderr, os.Stdin
		err := clone.Run()
		if err != nil {
			log.Error().
				Err(err).
				Strs("args", clone.Args).
				Str("K", k.Name).
				Msg("error executing clone command")
		}
	}

	return nil
}
