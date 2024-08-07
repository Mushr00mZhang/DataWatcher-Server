package modules

import (
	"log"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

const ConfigPath = "./config.yml"

type Config struct {
	Mutex       sync.Mutex        `yaml:"-"`           // 互斥锁
	Elastic     *Elastic          `yaml:"Elastic"`     // Elasticsearch
	Datasources *[]*Datasource    `yaml:"Datasources"` // 数据源列表
	Watchers    *[]*WatcherConfig `yaml:"Watchers"`    // 监控列表
}

// 保存配置文件
func (conf *Config) Save() {
	bytes, err := yaml.Marshal(conf)
	if err != nil {
		log.Fatalf("Save config file failed: %v", err)
	}
	os.WriteFile(ConfigPath, bytes, 0666)
}

// 读取配置文件
func NewConfig() *Config {
	bytes, err := os.ReadFile(ConfigPath)
	if err != nil {
		log.Fatalf("Read config file failed: %v", err)
		panic("Config file not found.")
	}
	var conf Config
	err = yaml.Unmarshal(bytes, &conf)
	if err != nil {
		log.Fatalf("Parse config file failed: %v", err)
		panic("Config file cannot parse.")
	}
	return &conf
}
