package cli

type CommandLineOpts struct {
	Version VersionCommand `command:"version"`
	Init    InitCommand    `command:"init"`
}

var Opts CommandLineOpts
