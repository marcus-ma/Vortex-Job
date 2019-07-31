package Vortex_Job

import (
	"encoding/json"
	"io/ioutil"
)

//程序配置项
type Config struct {
	ApiPort int `json:"apiPort"`
	ApiReadTimeout int `json:"apiReadTimeout"`
	ApiWriteTimeout int `json:"apiWriteTimeout"`
	BashDir string `json:"bashDir"`
	MongodbUri string `json:"mongodbUri"`
	MongodbConnectTimeout int64 `json:"mongodbConnectTimeout"`
}


var (
	G_config *Config
)

//加载配置项
func InitConfig(filename string) (err error)  {

	var(
		content []byte
		conf Config
	)

	if content,err = ioutil.ReadFile(filename);err!=nil{
		return
	}

	if err = json.Unmarshal(content,&conf);err!=nil{
		return
	}

	G_config = &conf

	return
}
