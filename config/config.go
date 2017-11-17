package config

import (
	"fmt"
	"math"
	"os"
	"os/user"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

const (
	configFileName    = "config"
	defaultConfigPath = "/btops"
	defaultHomeConfig = "/.config"
	xdgConfigHome     = "XDG_CONFIG_HOME"
	numeric           = "numeric"
	applicationNames  = "application-names"
)

type Config struct {
	Min                int
	Max                int
	RemoveEmpty        bool `mapstructure:"remove-empty"`
	AppendWhenOccupied bool `mapstructure:"append-when-occupied"`
	Renamers           []string
	ConstantName       string                `mapstructure:"constant-name"`
	StaticNames        []string              `mapstructure:"static-names"`
	ClassifiedNames    []map[string][]string `mapstructure:"classified-names"`
	WatchConfig        bool                  `mapstructure:"watch-config"`
	configChangeC      chan bool
}

func GetConfig() (*Config, error) {
	var c Config
	c.configChangeC = make(chan bool)

	currentUser, err := user.Current()
	if err != nil {
		fmt.Println("Unable to obtain current user")
	}

	viperConf := newDefaultConfig()
	viperConf.SetConfigName(configFileName)
	viperConf.AddConfigPath(fmt.Sprint(os.Getenv(xdgConfigHome), defaultConfigPath))
	viperConf.AddConfigPath(fmt.Sprint(currentUser.HomeDir, defaultHomeConfig, defaultConfigPath))

	if err := viperConf.ReadInConfig(); err != nil {
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			fmt.Println(err)
		default:
			return nil, err
		}
	}

	if err := viperConf.Unmarshal(&c); err != nil {
		return nil, err
	}

	if c.WatchConfig {
		viperConf.WatchConfig()
		viperConf.OnConfigChange(func(e fsnotify.Event) {
			c.configChangeC <- true
		})
	}

	return &c, nil
}

func (c Config) ConfigChanged() bool {
	select {
	case changed := <-c.configChangeC:
		return changed
	default:
		return false
	}
}

func newDefaultConfig() *viper.Viper {
	c := viper.New()

	c.SetDefault("min", 1)
	c.SetDefault("max", math.MaxInt64)
	c.SetDefault("remove-empty", true)
	c.SetDefault("append-when-occupied", true)
	c.SetDefault("default-naming-scheme", numeric)
	c.SetDefault("watch-config", true)

	return c
}
