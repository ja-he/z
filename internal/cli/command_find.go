package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"z/internal/cfg"

	"github.com/rs/zerolog/log"
)

type FindCommand struct {
	T    FindTextCommand `command:"t"`
	Text FindTextCommand `command:"text"`
	F    FindFileCommand `command:"f"`
	File FindFileCommand `command:"file"`
}

type FindTextCommand struct{}

func (c *FindTextCommand) Execute(args []string) error {
	pathsArg := ""
	sedConvertKToPathPipeline := ""
	sedConvertPathToKCommandStr := ""
	for kID, k := range cfg.GlobalCfg.Ks {
		pathsArg += fmt.Sprintf(` "%s"`, k.Path)
		sedConvertKToPathPipeline += fmt.Sprintf(
			`| sed "s/^%s /%s\//"`,
			kID,
			strings.ReplaceAll(k.Path, `/`, `\/`),
		)
		sedConvertPathToKCommandStr += fmt.Sprintf(
			`| sed "s/^%s\//%s /"`,
			strings.ReplaceAll(k.Path, `/`, `\/`),
			kID,
		)
	}

	rgCommandStr := `rg --line-number --with-filename . --color=never --field-match-separator ' '` + pathsArg + " "
	sourceCommandStr := rgCommandStr + sedConvertPathToKCommandStr
	fzfOptsStr := "--ansi --preview " +
		fmt.Sprintf(
			`'bat --color=always --decorations=never $(echo {1..2} %s) --highlight-line {3}'`,
			sedConvertKToPathPipeline,
		)
	fzfCommandStr := "fzf " + fzfOptsStr

	cmd := exec.Command("bash", "-c", sourceCommandStr+" | "+fzfCommandStr)

	cmd.Stderr, cmd.Stdin = os.Stderr, os.Stdin
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("cant get stdout pipe (%s)", err.Error())
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting bash command (%s)", err.Error())
	}
	selected, err := io.ReadAll(outPipe)
	if err != nil {
		return fmt.Errorf("cant read (%s)", err.Error())
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("error waiting on bash command (%s)", err.Error())
	}

	selected = bytes.TrimRight(selected, "\n")
	selectedLinewise := bytes.Split(selected, []byte{'\n'})

	switch len(selectedLinewise) {
	case 0:
		fmt.Println("nothing selected, exiting...")
		return nil

	case 1:
		tokens := bytes.Split(selectedLinewise[0], []byte{' '})
		if len(tokens) < 4 {
			return fmt.Errorf("expected at least 4 tokens to be returned by fzf (got %d)", len(tokens))
		}
		kID := string(tokens[0])
		file := string(tokens[1])
		zt := func() string {
			k, ok := cfg.GlobalCfg.Ks[kID]
			if !ok {
				return "F"
			}
			fullPath := path.Join(k.Path, file)
			dir, filename := path.Split(fullPath)
			z, err := cfg.ReadZ(dir)
			if err != nil {
				return "F"
			}
			for _, source := range z.Sources {
				if source == filename {
					return "S"
				}
			}
			for _, object := range z.Objects {
				if object == filename {
					return "O"
				}
			}
			return "F"
		}()

		return (&OpenCommand{}).Execute([]string{kID, file, zt})

	default:
		log.Warn().Msg("unable to open multiple files right now")
		return nil
	}
}

type FindFileCommand struct{}

func (c *FindFileCommand) Execute(args []string) error {

	fzfCmd := exec.Command("fzf", "--preview", "z preview {}")
	fzfCmd.Stderr = os.Stderr
	resultsWriter, err := fzfCmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("could not open stdin pipe for fzf (%s)", err.Error())
	}
	stdoutPipe, err := fzfCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("could not open stdout pipe for fzf (%s)", err.Error())
	}
	if err := fzfCmd.Start(); err != nil {
		return fmt.Errorf("could not start fzf (%s)", err.Error())
	}

	enumerationErr := (&EnumerateFilesCommand{
		K:        true,
		FileName: true,
		FileType: true,
		FullPath: false,
	}).enumerateFiles(resultsWriter)
	if enumerationErr != nil {
		return fmt.Errorf("could not enumerate files (%s)", enumerationErr)
	}

	selected, err := io.ReadAll(stdoutPipe)
	if err != nil {
		return fmt.Errorf("could not read fzf output (%s)", err.Error())
	}

	if err := fzfCmd.Wait(); err != nil {
		return fmt.Errorf("could not wait for fzf to complete (%s)", err.Error())
	}

	selected = bytes.TrimRight(selected, "\n")
	selectedLinewise := bytes.Split(selected, []byte{'\n'})

	switch len(selectedLinewise) {
	case 0:
		fmt.Println("nothing selected, exiting...")
		return nil

	case 1:
		line := string(selectedLinewise[0])
		args := strings.Split(line, "\t")
		return (&OpenCommand{}).Execute(args)

	default:
		log.Warn().Msg("unable to open multiple files right now")
		return nil
	}
}
