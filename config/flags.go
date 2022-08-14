package config

import "flag"

var configFile string
var genConfig bool

var Global Configs

func init() {
	flag.StringVar(&Global.FrontDomain, "front-domain", "http://localhost:8080", "front end domain")
	flag.StringVar(&Global.BackDomain, "back-domain", "http://localhost:8081", "back end domain")
	flag.StringVar(&Global.DataDir, "data-dir", "local/data/", "data dir")
	flag.StringVar(&Global.Listen, "listen", "0.0.0.0:8081", "listening address")
	flag.StringVar(&Global.DataSource, "datasource", "yaoj@tcp(127.0.0.1:3306)/yaoj?charset=utf8mb4&parseTime=True&multiStatements=true", "data source name")
	flag.StringVar(&Global.Sault, "sault", "3.1y4a1o5j9", "password sault")
	flag.IntVar(&Global.DefaultGroup, "default-group", 1, "default permission group")
	flag.StringVar(&configFile, "config", "", "config file")
	flag.BoolVar(&genConfig, "genconfig", false, "generate default config file")
}

func GenConfig() bool {
	return genConfig
}

func ConfigFile() string {
	return configFile
}