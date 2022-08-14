package config

type Configs struct {
	FrontDomain  string `yaml:"front_domain"`
	BackDomain   string `yaml:"back_domain"`
	DataDir      string `yaml:"data_dir"`
	Listen       string `yaml:"listen"`
	DataSource   string `yaml:"data_source"`
	Sault        string `yaml:"sault"`
	DefaultGroup int    `yaml:"default_group"`
}