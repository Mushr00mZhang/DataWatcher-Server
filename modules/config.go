package modules

const ConfigPath = "./config.yml"

type Config struct {
	Elastic  *Elastic          `yaml:"Elastic"`  // Elasticsearch
	Watchers *[]*WatcherConfig `yaml:"Watchers"` // 监控列表
}
