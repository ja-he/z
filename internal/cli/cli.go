package cli

type CommandLineOpts struct {
	Version VersionCommand `command:"version" description:"Display version information"`

	Init InitCommand `command:"init" description:"Initialize z: create config if missing, set up all Ks (clone remote, init local)"`

	C      CreateCommand `command:"c" description:"Create a new note or file (short for 'create')"`
	Create CreateCommand `command:"create" description:"Create a new note or file from a blueprint in a K"`

	F    FindCommand `command:"f" description:"Find notes by text or filename (short for 'find')"`
	Find FindCommand `command:"find" description:"Find notes by searching text content or filenames"`

	Preview        PreviewCommand        `command:"preview" description:"Preview a file in the terminal"`
	EnumerateFiles EnumerateFilesCommand `command:"enumerate-files" description:"List all files across all Ks"`

	Open OpenCommand `command:"open" description:"Open a file, directory, or Z-note with the appropriate application"`

	S    SyncCommand `command:"s" description:"Sync all Ks with their git remotes (short for 'sync')"`
	Sync SyncCommand `command:"sync" description:"Synchronize all Ks: commit local changes, pull from remote, and push"`

	M    MakeCommand `command:"m" description:"Run post-processing commands for a Z-note (short for 'make')"`
	Make MakeCommand `command:"make" description:"Execute post-processing commands defined in a Z-note's .z/z.yml"`
}

var Opts CommandLineOpts
