package main

import (
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"z/internal/cli"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	parser := flags.NewParser(&cli.Opts, flags.Default)
	parser.SubcommandsOptional = false

	_, err := parser.Parse()
	if flags.WroteHelp(err) {
		os.Exit(0)
	} else if err != nil {
		log.Fatal().Err(err).Msg("some flag parsing error occurred")
	}
}
