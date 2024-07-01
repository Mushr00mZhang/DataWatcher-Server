package es

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	elasticsearch7 "github.com/elastic/go-elasticsearch/v7"
)

type Config struct {
	Addresses []string               `yaml:"Addresses"`
	Client    *elasticsearch7.Client `json:"-"`
}

func (conf *Config) Init() {
	client, err := elasticsearch7.NewClient(elasticsearch7.Config{
		Addresses: conf.Addresses,
	})
	if err != nil {
		log.Fatalf("Connect to es failed: %v", err)
		panic(err)
	}
	conf.Client = client
}

func (conf *Config) Log(index string, data interface{}) error {
	if conf.Client == nil {
		conf.Init()
	}
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(data)
	if err != nil {
		return err
	}
	res, err := conf.Client.Index(
		strings.ToLower(index),
		&buf,
		conf.Client.Index.WithContext(context.Background()),
		conf.Client.Index.WithRefresh("true"),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return nil
}

const LogIndex = "logs"
const LogLevelDebug = "Debug"
const LogLevelInfo = "Info"
const LogLevelWarn = "Warn"
const LogLevelError = "Error"

type Log struct {
	Timestamp time.Time   `json:"timestamp"`
	Level     string      `json:"Level"`
	Info      string      `json:"Info"`
	Detail    string      `json:"Detail"`
	Extend    interface{} `json:"Extend"`
}

func (conf *Config) New(level string, info string, detail string, extend interface{}) Log {
	return Log{
		Timestamp: time.Now().Local(),
		Level:     level,
		Info:      info,
		Detail:    detail,
		Extend:    extend,
	}
}
func (conf *Config) NewDebug(info string, detail string, extend interface{}) {
	log := conf.New(LogLevelDebug, info, detail, extend)
	conf.Log(LogIndex, log)
}
func (conf *Config) NewInfo(info string, detail string, extend interface{}) {
	log := conf.New(LogLevelInfo, info, detail, extend)
	conf.Log(LogIndex, log)
}
func (conf *Config) NewWarn(info string, detail string, extend interface{}) {
	log := conf.New(LogLevelWarn, info, detail, extend)
	conf.Log(LogIndex, log)
}
func (conf *Config) NewError(info string, detail string, extend interface{}) {
	log := conf.New(LogLevelError, info, detail, extend)
	conf.Log(LogIndex, log)
}
