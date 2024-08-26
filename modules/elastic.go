package modules

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	es7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/estransport"
)

type Elastic struct {
	Addresses []string    `yaml:"Addresses"`
	Username  string      `yaml:"Username"`
	Password  string      `yaml:"Password"`
	Client    *es7.Client `yaml:"-" json:"-"`
}

func (conf *Elastic) Init() {
	client, err := es7.NewClient(es7.Config{
		Addresses: conf.Addresses,
		Username:  conf.Username,
		Password:  conf.Password,
	})
	if err != nil {
		log.Fatalf("Connect to es failed: %v", err)
		panic(err)
	}
	conf.Client = client
	log.Println(client.Transport.(*estransport.Client).URLs())
}

func (conf *Elastic) Log(index string, data interface{}) error {
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
	defer res.Body.Close()
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusCreated {
		_bytes, _ := io.ReadAll(res.Body)
		fmt.Printf("Log index failed. Error:%s\n", string(_bytes))
	}
	return nil
}

const LogIndex = "logs"
const LogLevelDebug = "Debug"
const LogLevelInfo = "Info"
const LogLevelWarn = "Warn"
const LogLevelError = "Error"

type Log struct {
	Timestamp time.Time   `json:"@timestamp"`
	Level     string      `json:"Level"`
	Info      string      `json:"Info"`
	Detail    string      `json:"Detail"`
	Extend    interface{} `json:"Extend"`
}

func (conf *Elastic) New(level string, info string, detail string, extend interface{}) Log {
	return Log{
		Timestamp: time.Now().Local(),
		Level:     level,
		Info:      info,
		Detail:    detail,
		Extend:    extend,
	}
}
func (conf *Elastic) NewDebug(info string, detail string, extend interface{}) {
	log := conf.New(LogLevelDebug, info, detail, extend)
	conf.Log(LogIndex, log)
}
func (conf *Elastic) NewInfo(info string, detail string, extend interface{}) {
	log := conf.New(LogLevelInfo, info, detail, extend)
	conf.Log(LogIndex, log)
}
func (conf *Elastic) NewWarn(info string, detail string, extend interface{}) {
	log := conf.New(LogLevelWarn, info, detail, extend)
	conf.Log(LogIndex, log)
}
func (conf *Elastic) NewError(info string, detail string, extend interface{}) {
	log := conf.New(LogLevelError, info, detail, extend)
	conf.Log(LogIndex, log)
}
