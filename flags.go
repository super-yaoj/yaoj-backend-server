package main

import "flag"

type Config struct {
	FrontDomain string `yaml:"front_domain"`
	BackDomain  string `yaml:"back_domain"`
	DataDir     string `yaml:"data_dir"`
	TmpDir      string `yaml:"tmp_dir"`
	Listen      string `yaml:"listen"`
}

var configFile string
var genConfig bool

var config Config

func init() {
	flag.StringVar(&config.FrontDomain, "front-domain", "http://localhost:8080", "front end domain")
	flag.StringVar(&config.BackDomain, "back-domain", "http://localhost:8081", "back end domain")
	flag.StringVar(&config.DataDir, "data-dir", "local/data/", "data dir")
	flag.StringVar(&config.TmpDir, "tmp-dir", "local/tmp/", "tmp dir")
	flag.StringVar(&config.Listen, "listen", "0.0.0.0:8081", "listening address")
	flag.StringVar(&configFile, "config", "", "config file")
	flag.BoolVar(&genConfig, "genconfig", false, "generate default config file")
}
