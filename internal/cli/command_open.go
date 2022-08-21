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

	zType := args[2]

	switch zType {
	case "Z":
		zPath := path.Join(fullPath, ".z", "z.yml")
		zYAML, err := os.ReadFile(zPath)
		if err != nil {
			return fmt.Errorf("could not open '%s' (%s)", zPath, err.Error())
		}
		z := cfg.Z{}
		if err := yaml.Unmarshal(zYAML, &z); err != nil {
			return fmt.Errorf("could not read yaml in '%s' (%s)", zPath, err.Error())
		}
		openCmd := exec.Command("bash", "-c", fmt.Sprintf("cd '%s' ; %s", fullPath, z.Open))
		openCmd.Stdout, openCmd.Stderr, openCmd.Stdin = os.Stdout, os.Stderr, os.Stdin
		if err := openCmd.Run(); err != nil {
			return fmt.Errorf("could not run open command from '%s' (%s)", zPath, err.Error())
		}
		for i, post := range z.Post {
			fmt.Println("running post command", i, ":")
			postCmd := exec.Command("bash", "-c", fmt.Sprintf("cd '%s' ; %s", fullPath, post))
			fmt.Println(postCmd.String())
			postCmd.Stdout, postCmd.Stderr, postCmd.Stdin = os.Stdout, os.Stderr, os.Stdin
			if err := postCmd.Run(); err != nil {
				return fmt.Errorf("unable to run post command %d from '%s' (%s)", i, zPath, err.Error())
			}
		}
		return nil

	case "D":
		return fmt.Errorf("TODO: open regular dir")

	case "F", "S", "O":
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
		if zType == "S" {
			dir, _ := path.Split(fullPath)
			z, err := cfg.ReadZ(dir)
			if err != nil {
				return fmt.Errorf("unable to read .z/z.yml to do post hooks (%s)", err.Error())
			}
			for i, post := range z.Post {
				postCmd := exec.Command("bash", "-c", fmt.Sprintf("cd '%s' ; %s", dir, post))
				postCmd.Stdout, postCmd.Stderr, postCmd.Stdin = os.Stdout, os.Stderr, os.Stdin
				if err := postCmd.Run(); err != nil {
					return fmt.Errorf("unable to run post command %d (%s)", i, err.Error())
				}
			}
		}

	default:
		return fmt.Errorf("Unknown Z-Type '%s'", zType)
	}

	return nil
}
