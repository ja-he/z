package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"z/internal/cfg"

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
