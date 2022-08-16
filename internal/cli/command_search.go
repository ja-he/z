package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"z/internal/cfg"

	"github.com/rs/zerolog/log"
)

type SearchCommand struct {
	Text SearchTextCommand `command:"text"`
	File SearchFileCommand `command:"file"`
}

type SearchTextCommand struct{}

func (c *SearchTextCommand) Execute(args []string) error {
	pathsArg := ""
	sedConvertKToPathPipeline := ""
	sedConvertPathToKPipeline := ""
	for kID, k := range cfg.GlobalCfg.Ks {
		pathsArg += fmt.Sprintf(` "%s"`, k.Path)
		sedConvertKToPathPipeline += fmt.Sprintf(
			`| sed "s/^%s /%s\//"`,
			kID,
			strings.ReplaceAll(k.Path, `/`, `\/`),
		)
		sedConvertPathToKPipeline += fmt.Sprintf(
			`| sed "s/^%s\//%s /"`,
			strings.ReplaceAll(k.Path, `/`, `\/`),
			kID,
		)
	}

	cmdStr :=
		`rg --line-number --with-filename . --color=never --field-match-separator ' '` + pathsArg + " " +
			sedConvertPathToKPipeline + " | " +
			"fzf --ansi --preview " +
			fmt.Sprintf(
				`'bat --color=always --decorations=never $(echo {1..2} %s) --highlight-line {3}'`,
				sedConvertKToPathPipeline,
			)
	cmd := exec.Command("bash", "-c", cmdStr)

	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running bash command (%s)", err.Error())
	}

	return nil
}

type SearchFileCommand struct{}

func (c *SearchFileCommand) Execute(args []string) error {

	resultsMtx := sync.Mutex{}
	fzfCmd := exec.Command("fzf", "--preview", "z preview {}")
	fzfCmd.Stderr = os.Stderr
	resultsWriter, err := fzfCmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("could not open stdin pipe for fzf (%s)", err.Error())
	}
	writeResult := func(data []byte) error {
		resultsMtx.Lock()
		_, err := resultsWriter.Write(data)
		resultsMtx.Unlock()
		return err
	}
	stdoutPipe, err := fzfCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("could not open stdout pipe for fzf (%s)", err.Error())
	}
	if err := fzfCmd.Start(); err != nil {
		return fmt.Errorf("could not start fzf (%s)", err.Error())
	}

	for id, k := range cfg.GlobalCfg.Ks {
		entries, err := os.ReadDir(k.Path)
		if err != nil {
			return fmt.Errorf("unable to read dir '%s' for K '%s'", k.Path, id)
		}
		for i := range entries {
			if entries[i].Name()[0] == '.' {
				continue
			}
			if entries[i].Type().IsDir() {
				dir := entries[i].Name()
				dirEntries, err := os.ReadDir(path.Join(k.Path, dir))
				if err != nil {
					log.Warn().Str("dir", dir).Msg("could not open dir for reading")
				} else {
					hasZ := func() bool {
						for _, e := range dirEntries {
							if e.Name() == ".z" {
								return true
							}
						}
						return false
					}()
					if hasZ {
						if err := writeResult([]byte(fmt.Sprintf("%s\t%s\t%s\n", id, dir, "Z"))); err != nil {
							log.Warn().Err(err).Msg("error writing result")
						}
						z, err := cfg.ReadZ(path.Join(k.Path, dir))
						if err != nil {
							return fmt.Errorf("unable to get z-data from dir (%s)", err.Error())
						}
						for _, source := range z.Sources {
							if err := writeResult([]byte(fmt.Sprintf("%s\t%s\t%s\n", id, path.Join(dir, source), "S"))); err != nil {
								log.Warn().Err(err).Msg("error writing result")
							}
						}
						for _, object := range z.Objects {
							if err := writeResult([]byte(fmt.Sprintf("%s\t%s\t%s\n", id, path.Join(dir, object), "O"))); err != nil {
								log.Warn().Err(err).Msg("error writing result")
							}
						}
					} else {
						for _, e := range dirEntries {
							if e.Name()[0] == '.' {
								continue
							}
							if err := writeResult([]byte(fmt.Sprintf("%s\t%s\t%s\n", id, path.Join(dir, e.Name()), "F"))); err != nil {
								log.Warn().Err(err).Msg("error writing result")
							}
						}
					}
				}
			} else {
				if err := writeResult([]byte(fmt.Sprintf("%s\t%s\t%s\n", id, entries[i].Name(), "F"))); err != nil {
					log.Warn().Err(err).Msg("error writing result")
				}
			}
		}
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
