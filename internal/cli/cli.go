package cli

type CommandLineOpts struct {
	Version VersionCommand `command:"version"`
	Init    InitCommand    `command:"init"`
	Create  CreateCommand  `command:"create"`
	Search  SearchCommand  `command:"search"`
	Preview PreviewCommand `command:"preview"`
	Open    OpenCommand    `command:"open"`
	Sync    SyncCommand    `command:"sync"`
}

var Opts CommandLineOpts
