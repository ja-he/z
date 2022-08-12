package cli

type CommandLineOpts struct {
	Version VersionCommand `command:"version"`
	Init    InitCommand    `command:"init"`
	Create  CreateCommand  `command:"create"`
}

var Opts CommandLineOpts
