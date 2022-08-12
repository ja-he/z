package cfg

// the global config, parsed by main
var GlobalCfg Cfg

type Cfg struct {
	Ks         map[string]K         `yaml:"Ks"`
	Blueprints map[string]Blueprint `yaml:"blueprints"`
}

type K struct {
	Path string `yaml:"path"`
	URL  string `yaml:"url"`
}

type Blueprint struct {
	Templates map[string]string `yaml:"templates"`
	Open      string            `yaml:"open"`
}

type TemplateFiller struct {
	K     K
	Name  string
	Today string
}
