package config

import (
	"fmt"
	"log"
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
	RemoveFocused      bool `mapstructure:"remove-focused"`
	AppendWhenOccupied bool `mapstructure:"append-when-occupied"`
	WatchConfig        bool `mapstructure:"watch-config"`
	configChangeC      chan bool
	Renamers           []string
	Names              names
}

type names struct {
	Constant   string
	Static     []string
	Classified []map[string][]string
}

func GetConfig() (*Config, error) {
	var c Config
	c.configChangeC = make(chan bool)

	currentUser, err := user.Current()
	if err != nil {
		log.Println("Unable to obtain current user")
	}

	viperConf := newDefaultConfig()
	viperConf.SetConfigName(configFileName)
	viperConf.AddConfigPath(fmt.Sprint(os.Getenv(xdgConfigHome), defaultConfigPath))
	viperConf.AddConfigPath(fmt.Sprint(currentUser.HomeDir, defaultHomeConfig, defaultConfigPath))

	if err := viperConf.ReadInConfig(); err != nil {
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			log.Println(err)
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
	c.SetDefault("remove-focused", true)
	c.SetDefault("append-when-occupied", true)
	c.SetDefault("renamers", []string{numeric})
	c.SetDefault("watch-config", true)

	return c
}
