package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"z/internal/cfg"
)

type OpenCommand struct{}

func (c *OpenCommand) Execute(args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("expected 3 args for command 'open' but got %d", len(args))
	}

	kID := args[0]
	k, ok := cfg.GlobalCfg.Ks[kID]
	if !ok {
		return fmt.Errorf("no such K '%s'", kID)
	}

	file := args[1]
	fullPath := path.Join(k.Path, file)
	_, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("file '%s' stat error (%s)", fullPath, err.Error())
	}

	ft := args[2]

	switch ft {
	case "Z":
		return fmt.Errorf("TODO: open Z-dir")

	case "D":
		return fmt.Errorf("TODO: open regular dir")

	default:
		ext := strings.TrimLeft(path.Ext(fullPath), ".")
		openCmd, err := func() (*exec.Cmd, error) {
			switch ext {
			case "md", "txt", "tex":
				nvimCmd := exec.Command("nvim", fullPath)
				return nvimCmd, nil

			case "png", "jpg", "jpeg", "tif":
				openCmd := exec.Command("feh", fullPath)
				return openCmd, nil

			case "pdf":
				openCmd := exec.Command("zathura", fullPath)
				return openCmd, nil

			case "xopp":
				openCmd := exec.Command("xournalpp", fullPath)
				return openCmd, nil

			default:
				return nil, fmt.Errorf("unknown file extension '%s'", ext)
			}
		}()
		if err != nil {
			return fmt.Errorf("error creating command (%s)", err.Error())
		}
		openCmd.Stdout, openCmd.Stderr, openCmd.Stdin = os.Stdout, os.Stderr, os.Stdin
		if err := openCmd.Run(); err != nil {
			return fmt.Errorf("open command error (%s)", err.Error())
		}
	}

	return nil
}
