package cfg

// the global config, parsed by main
var GlobalCfg Cfg

type Cfg struct {
	Ks []K `yaml:"Ks"`
}

type K struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
	URL  string `yaml:"url"`
}
