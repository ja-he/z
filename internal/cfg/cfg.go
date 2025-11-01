// Package cfg provides the global config, parsed by main.
package cfg

// GlobalCfg is the global config, parsed by main.
var GlobalCfg Cfg

// Cfg is the top level config.
type Cfg struct {
	Settings   Settings             `yaml:"settings"`
	Ks         map[string]K         `yaml:"Ks"`
	Blueprints map[string]Blueprint `yaml:"blueprints"`
}

// Settings contains application-wide settings.
type Settings struct {
	Color *bool `yaml:"color"` // Enable colored output in logs (default: true if nil)
}

// A K is a single 'Kasten', a directory of Zs (files).
type K struct {
	Path string `yaml:"path"` // when empty, sync will be assumed to be manual
	URL  string `yaml:"url"`
}

// A Blueprint is a template for a new Z (file).
type Blueprint struct {
	Subdir    string            `yaml:"subdir"`
	Templates map[string]string `yaml:"templates"`
	Open      string            `yaml:"open"`
	View      string            `yaml:"view"`
	Post      []string          `yaml:"post"`
	Sources   []string          `yaml:"sources"`
	Objects   []string          `yaml:"objects"`
}

// TemplateFiller is the data passed to templates.
type TemplateFiller struct {
	K     K
	Name  string
	Today string
	Now   string
}
