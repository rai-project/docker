package docker

import (
	"strings"

	"time"

	"github.com/k0kubun/pp"
	"github.com/rai-project/config"
	"github.com/rai-project/serializer"
	"github.com/rai-project/serializer/bson"
	"github.com/rai-project/serializer/json"
	"github.com/rai-project/vipertags"
)

type dockerConfig struct {
	Serializer     serializer.Serializer `json:"-" config:"-"`
	SerializerName string                `json:"serializer_name" config:"docker.serializer" default:"json"`
	TimeLimit      time.Duration         `json:"time_limit" config:"docker.time_limit" default:"1h"`
	Repository     string                `json:"repository" config:"docker.repository"`
	Tag            string                `json:"tag" config:"docker.tag" default:"latest"`
	User           string                `json:"user" config:"docker.user"`
	RaiUserName    string                `json:"user_name" config:"docker.rai_user_name"`
	MemoryLimit    int64                 `json:"memory_limit" config:"docker.memory_limit"`
	Endpoints      []string              `json:"endpoints" config:"docker.endpoints" default:"/run/docker.sock /var/run/docker.sock"`
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
	switch strings.ToLower(a.SerializerName) {
	case "json":
		a.Serializer = json.New()
	case "bson":
		a.Serializer = bson.New()
	default:
		log.WithField("serializer", a.SerializerName).
			Warn("Cannot find serializer")
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
