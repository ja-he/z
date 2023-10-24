package cfg

// the global config, parsed by main
var GlobalCfg Cfg

type Cfg struct {
	Ks         map[string]K         `yaml:"Ks"`
	Blueprints map[string]Blueprint `yaml:"blueprints"`
}

type K struct {
	Path string `yaml:"path"` // when empty, sync will be assumed to be manual
	URL  string `yaml:"url"`
}

type Blueprint struct {
	Subdir    string            `yaml:"subdir"`
	Templates map[string]string `yaml:"templates"`
	Open      string            `yaml:"open"`
	View      string            `yaml:"view"`
	Post      []string          `yaml:"post"`
	Sources   []string          `yaml:"sources"`
	Objects   []string          `yaml:"objects"`
}

type TemplateFiller struct {
	K     K
	Name  string
	Today string
	Now   string
}
