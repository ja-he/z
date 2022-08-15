package cfg

type Z struct {
	Open    string   `yaml:"open"`
	Post    []string `yaml:"post"`
	Sources []string `yaml:"sources"`
	Objects []string `yaml:"objects"`
}
