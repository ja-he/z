package cli

type CommandLineOpts struct {
	Version VersionCommand `command:"version" subcommands-optional:"true"`
}

var Opts CommandLineOpts
