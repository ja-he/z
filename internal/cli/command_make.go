package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"z/internal/cfg"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type MakeCommand struct {
	Path string `short:"C" long:"directory" description:"the directory to run in" default:"."`
}

func (c *MakeCommand) Execute(_ []string) error {
	zPath := path.Join(c.Path, ".z", "z.yml")
	zYAML, err := os.ReadFile(zPath)
	if err != nil {
		return fmt.Errorf("could not open '%s' (%s)", zPath, err.Error())
	}
	z := cfg.Z{}
	if err := yaml.Unmarshal(zYAML, &z); err != nil {
		return fmt.Errorf("could not read yaml in '%s' (%s)", zPath, err.Error())
	}
	for i, post := range z.Post {
		postCmd := exec.Command("bash", "-c", fmt.Sprintf("cd '%s' ; %s", c.Path, post))
		log.Info().Int("i", i).Str("command", postCmd.String()).Msg("running post command:")
		postCmd.Stdout, postCmd.Stderr, postCmd.Stdin = os.Stdout, os.Stderr, os.Stdin
		if err := postCmd.Run(); err != nil {
			return fmt.Errorf("unable to run post command %d from '%s' (%s)", i, zPath, err.Error())
		}
	}
	return nil
}
