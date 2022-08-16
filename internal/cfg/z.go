package cfg

import (
	"fmt"
	"os"
	"path"

	"gopkg.in/yaml.v3"
)

type Z struct {
	Open    string   `yaml:"open"`
	Post    []string `yaml:"post"`
	Sources []string `yaml:"sources"`
	Objects []string `yaml:"objects"`
}

func ReadZ(dir string) (*Z, error) {
	base := path.Base(dir)
	if base == ".z" || base == "z.yml" {
		return nil, fmt.Errorf("ReadZ(<dir>) expects <dir> to be path to note dir which contains .z/z.yml, not either .z or z.yml's path")
	}
	data, err := os.ReadFile(path.Join(dir, ".z", "z.yml"))
	if err != nil {
		return nil, fmt.Errorf("unable to read (%s)", err.Error())
	}
	z := Z{}
	if err := yaml.Unmarshal(data, &z); err != nil {
		return nil, fmt.Errorf("unable to unmarshal (%s)", err.Error())
	}
	return &z, nil
}
