package cli

type CommandLineOpts struct {
	Version VersionCommand `command:"version"`

	Init InitCommand `command:"init"`

	C      CreateCommand `command:"c"`
	Create CreateCommand `command:"create"`

	F    FindCommand `command:"f"`
	Find FindCommand `command:"find"`

	Preview        PreviewCommand        `command:"preview"`
	EnumerateFiles EnumerateFilesCommand `command:"enumerate-files"`

	Open OpenCommand `command:"open"`

	S    SyncCommand `command:"s"`
	Sync SyncCommand `command:"sync"`
}

var Opts CommandLineOpts
