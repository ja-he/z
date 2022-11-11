package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"z/internal/cfg"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type PreviewCommand struct {
}

func (c *PreviewCommand) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expected a single (single-quoted) arg for 'preview' but got %d", len(args))
	}

	sArgs := strings.Split(args[0], "\t")
	if len(sArgs) != 3 {
		return fmt.Errorf("expected three tab-separated args in arg string %d", len(args))
	}

	kID := sArgs[0]
	k, ok := cfg.GlobalCfg.Ks[kID]
	if !ok {
		return fmt.Errorf("no such K '%s'", kID)
	}

	file := sArgs[1]
	fullPath := path.Join(k.Path, file)
	_, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("file '%s' stat error (%s)", fullPath, err.Error())
	}

	zType := sArgs[2]

	termwidth, _, err := func() (int, int, error) {
		wCmd := exec.Command("tput", "cols")
		wData, err := wCmd.Output()
		if err != nil {
			return 0, 0, err
		}
		w, err := strconv.Atoi(string(bytes.TrimRight(wData, "\n")))
		if err != nil {
			return 0, 0, err
		}

		hCmd := exec.Command("tput", "lines")
		hData, err := hCmd.Output()
		if err != nil {
			return 0, 0, err
		}
		h, err := strconv.Atoi(string(bytes.TrimRight(hData, "\n")))
		if err != nil {
			return 0, 0, err
		}

		return w, h, nil
	}()
	if err != nil {
		log.Warn().Err(err).Msg("don't have termwidth,-height data")
	}

	switch zType {

	case "Z":
		z, err := cfg.ReadZ(fullPath)
		if err != nil {
			return err
		}
		printable, _ := yaml.Marshal(z)
		fmt.Printf(string(printable))

	case "D":
		cmd := exec.Command("ls", "-a1", fullPath)
		cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("error running preview command (%s)", err.Error())
		}

	case "F", "S", "O":
		ext := path.Ext(file)
		var cmd *exec.Cmd = nil
		switch strings.TrimPrefix(ext, ".") {
		case "txt", "md", "tex", "bib":
			cmd = exec.Command("bat", "--color", "always", "--decorations", "never", fullPath)
		case "jpeg", "jpg", "png", "tif", "gif":
			cmd = exec.Command("catimg", "-w", fmt.Sprint(termwidth*2), fullPath)
		case "pdf":
			cmd = exec.Command("pdftotext", fullPath, "-")
		case "html":
			cmd = exec.Command("w3m", "-dump", fullPath, "-cols", fmt.Sprint(termwidth))
		}
		if cmd != nil {
			cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("error running preview command (%s)", err.Error())
			}
		} else {
			fmt.Printf("extension '%s' not previewable\n", ext)
		}

	default:
		return fmt.Errorf("Unknown Z-Type '%s'", zType)

	}

	return nil
}
