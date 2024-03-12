package conf

import (
	"bufio"
	"encoding/json"
	"os"
)

type config struct {
	Env              string                     `json:"env"`
	Port             string                     `json:"port"`
	AesKey           string                     `json:"aes_key"`
	Mysql            Mysql                      `json:"mysql"`
	Email            Email                      `json:"email"`
	Redis            Redis                      `json:"redis"`
	Kafka            Kafka                      `json:"kafka"`
	Log              Log                        `json:"log"`
	Jira             Jira                       `json:"jira"`
	Sso              Sso                        `json:"sso"`
	HorusEye         HorusEye                   `json:"horus_eye"`
	Minio            Minio                      `json:"minio"`
	Nacos            Nacos                      `json:"nacos"`
	Yops             Yops                       `json:"yops"`
	APPAuth          map[string]Auth            // 存放认证用户信息
	ExcludeAuth      map[string]map[string]bool // 存放不校验的URL
	LoginExcludeAuth map[string]map[string]bool // 存放不校验的URL
}

type Auth struct {
	Name   string
	Secret string
}

type Yops struct {
	Host string            `json:"host"`
	Api  map[string]string `json:"api"`
	Env  map[string]string `json:"env"`
}

type Nacos struct {
	Endpoints []NacosEndpoint `json:"endpoints"`
	Username  string          `json:"username"`
	Password  string          `json:"password"`
	Namespace string          `json:"namespace"`
}
type NacosEndpoint struct {
	Addr string `json:"addr"`
	Port int    `json:"port"`
}

type Minio struct {
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Bucket    string `json:"bucket"`
	Secure    bool   `json:"secure"`
}

type Jira struct {
	Jql        string `json:"jql"`
	Server     string `json:"server"`
	User       string `json:"user"`
	Password   string `json:"password"`
	Transition struct {
		AwaitStatus     string `json:"await_status"`
		AcceptOperate   string `json:"accept_operate"`
		AcceptStatus    string `json:"accept_status"`
		SecurityOperate string `json:"security_operate"`
		SecurityStatus  string `json:"security_status"`
		ExecOperate     string `json:"exec_operate"`
		ExecStatus      string `json:"exec_status"`
		ExecEndOperate  string `json:"exec_end_operate"`
		ExecEndStatus   string `json:"exec_end_status"`
		RejectOperate   string `json:"reject_operate"`
		RejectStatus    string `json:"reject_status"`
	} `json:"transition"`
	Accept string `json:"accept"`
	Reject string `json:"reject"`
}

type Mysql struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"db_name"`
}

type Log struct {
	Level      string `json:"level"`
	Filename   string `json:"filename"`
	MaxSize    int    `json:"maxsize"`
	MaxAge     int    `json:"max_age"`
	MaxBackups int    `json:"max_backups"`
}

type Email struct {
	Host     string   `json:"host"`
	Port     int      `json:"port"`
	Username string   `json:"user"`
	Password string   `json:"password"`
	Sender   string   `json:"sender"`
	To       []string `json:"to"`
}

type Redis struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	DB       int    `json:"db"`
	Password string `json:"password"`
}

type Kafka struct {
	BootstrapServers []string `json:"bootstrap_servers"`
	Topic            string   `json:"topic"`
	Key              string   `json:"key"`
}
type HorusEye struct {
	Host      string `json:"host"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}
type Sso struct {
	Enable bool   `json:"enable"`
	Host   string `json:"host"`
	AppId  string `json:"app_id"`
}
type RedisKey struct {
	SubnetList   string
	SubnetDetail string
}

var Config *config

type api struct {
	Method string
	Uri    string
}

// LoginExcludeAuth 登录后都有权限的接口
var LoginExcludeAuth = []api{
	{"GET", "/auth/*"},
}

// ExcludeAuth 都有权限的接口
var ExcludeAuth = []api{
	{"GET", "/auth/public_key"},
	{"POST", "/auth/login"},
} // 存放不校验的URL

func InitConfig() {
	Config = &config{}
	file, err := os.Open("conf/config.json")
	defer file.Close()
	if err != nil {
		panic(err)
	}
	reader := bufio.NewReader(file)
	decoder := json.NewDecoder(reader)
	if err = decoder.Decode(&Config); err != nil {
		panic(err)
	}
}
