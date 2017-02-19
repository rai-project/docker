package docker

import (
	"strconv"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/k0kubun/pp"
	"github.com/rai-project/config"
	"github.com/rai-project/vipertags"
)

type dockerConfig struct {
	TimeLimit         time.Duration     `json:"time_limit" config:"docker.time_limit" default:"1h"`
	Image             string            `json:"image" config:"docker.image"`
	Username          string            `json:"username" config:"docker.username"`
	MemoryLimitString string            `json:"memory_limit" config:"docker.memory_limit"`
	MemoryLimit       int64             `json:"-" config:"-"`
	Env               map[string]string `json:"env" config:"docker.env"`
	Host              string            `json:"host" config:"docker.host" default:"unix:///var/run/docker.sock" env:"DOCKER_HOST"`
	APIVersion        string            `json:"api_version" config:"docker.api_version" default:"" env:"DOCKER_API_VERSION"`
	CertPath          string            `json:"cert_path" config:"docker.cert_path" default:"" env:"DOCKER_CERT_PATH"`
	TLSVerify         bool              `json:"tls_verify" config:"docker.tls_verify" default:"false" env:"DOCKER_TLS_VERIFY"`
}

var (
	Config = &dockerConfig{}
)

func (dockerConfig) ConfigName() string {
	return "Docker"
}

func (dockerConfig) SetDefaults() {
}

func (a *dockerConfig) Read() {
	vipertags.Fill(a)
	if a.MemoryLimitString != "" {
		if bts, err := humanize.ParseBytes(a.MemoryLimitString); err == nil {
			a.MemoryLimit = int64(bts)
		} else if bts, err := strconv.ParseInt(a.MemoryLimitString, 10, 0); err == nil {
			a.MemoryLimit = bts
		}
	}
}

func (c dockerConfig) String() string {
	return pp.Sprintln(c)
}

func (c dockerConfig) Debug() {
	log.Debug("Docker Config = ", c)
}

func init() {
	config.Register(Config)
}
