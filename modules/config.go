package modules

const ConfigPath = "./config.yml"

type Config struct {
	Elastic     *Elastic          `yaml:"Elastic"`     // Elasticsearch
	Watchers    *[]*WatcherConfig `yaml:"Watchers"`    // 监控列表
	Datasources *[]*Datasource    `yaml:"Datasources"` // 数据源列表
}
