package services

import (
	"log"
	"os"
	"server/modules"

	"gopkg.in/yaml.v3"
)

type ConfigService struct {
}

// 读取配置文件
func (service ConfigService) Read() *modules.Config {
	bytes, err := os.ReadFile(modules.ConfigPath)
	if err != nil {
		log.Fatalf("Read config file failed: %v", err)
		panic("Config file not found.")
	}
	var conf modules.Config
	err = yaml.Unmarshal(bytes, &conf)
	if err != nil {
		log.Fatalf("Parse config file failed: %v", err)
		panic("Config file cannot parse.")
	}
	return &conf
}

// 保存配置文件
func (service ConfigService) Save(conf *modules.Config) {
	bytes, err := yaml.Marshal(conf)
	if err != nil {
		log.Fatalf("Save config file failed: %v", err)
	}
	os.WriteFile(modules.ConfigPath, bytes, 0666)
}
