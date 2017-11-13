package rules

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"strings"

	"github.com/cmschuetz/bspwm-desktops/monitors"
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
	Min                 *int
	Max                 *int
	RemoveEmpty         bool   `mapstructure:"remove-empty"`
	AppendWhenOccupied  bool   `mapstructure:"append-when-occupied"`
	DefaultNamingScheme string `mapstructure:"default-naming-scheme"`
}

type baseHandler struct {
	config *Config
}

type handlerFunc func(*monitors.Monitors) bool

type AppendHandler struct{ baseHandler }
type RemoveHandler struct{ baseHandler }
type RenameHandler struct {
	baseHandler
	handler handlerFunc
}

type Handler interface {
	Initialize(*Config)
	ShouldHandle() bool
	Handle(*monitors.Monitors) bool
}

type Handlers []Handler

func GetConfig() (*Config, error) {
	var c Config

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

	return &c, nil
}

func NewHandlers(c *Config) *Handlers {
	h := Handlers{
		&AppendHandler{},
		&RemoveHandler{},
		&RenameHandler{},
	}

	for i := range h {
		h[i].Initialize(c)
	}

	return &h
}

func newDefaultConfig() *viper.Viper {
	c := viper.New()

	c.SetDefault("remove-empty", true)
	c.SetDefault("append-when-occupied", true)
	c.SetDefault("default-naming-scheme", numeric)

	return c
}

func (h Handlers) Handle(m *monitors.Monitors) {
	for _, handler := range h {
		if !handler.ShouldHandle() {
			continue
		}

		if handler.Handle(m) {
			return
		}
	}
}

func (b *baseHandler) Initialize(c *Config) {
	b.config = c
}

func (a AppendHandler) ShouldHandle() bool {
	return a.config.AppendWhenOccupied
}

func (a AppendHandler) Handle(m *monitors.Monitors) bool {
	for _, monitor := range *m {
		dCount := len(monitor.Desktops)

		if a.config.Max != nil && *a.config.Max <= dCount {
			continue
		}

		if a.config.Min != nil && *a.config.Min > dCount ||
			!monitor.Desktops[dCount-1].IsEmpty() {
			err := monitor.AppendDesktop(strconv.Itoa(dCount + 1))
			if err != nil {
				fmt.Println("Unable to append desktop to monitor: ", monitor.Name, err)
				continue
			}

			return true
		}
	}

	return false
}

func (r RemoveHandler) ShouldHandle() bool {
	return r.config.RemoveEmpty
}

func (r RemoveHandler) Handle(m *monitors.Monitors) bool {
	for _, monitor := range *m {
		for _, desktop := range monitor.EmptyDesktops() {
			if *desktop == monitor.Desktops[len(monitor.Desktops)-1] {
				continue
			}

			if r.config.Min != nil && *r.config.Min >= len(monitor.Desktops) {
				continue
			}

			err := monitor.RemoveDesktop(desktop.Id)
			if err != nil {
				fmt.Println("Unable to remove desktop: ", desktop.Name, err)
				continue
			}

			return true
		}
	}

	return false
}

func (r *RenameHandler) Initialize(c *Config) {
	r.baseHandler.Initialize(c)
	r.handler = r.GetHandler()
}

func (r RenameHandler) GetHandler() func(*monitors.Monitors) bool {
	switch r.config.DefaultNamingScheme {
	case numeric:
		return r.NumericHandler
	case applicationNames:
		return r.ApplicationNamesHandler
	}

	return nil
}

func (r RenameHandler) ShouldHandle() bool {
	return r.handler != nil
}

func (r RenameHandler) NumericHandler(m *monitors.Monitors) bool {
	for _, monitor := range *m {
		for i, desktop := range monitor.Desktops {
			name := strconv.Itoa(i + 1)
			if desktop.Name == name {
				continue
			}

			err := desktop.Rename(name)
			if err != nil {
				fmt.Println("Unable to rename desktop: ", desktop.Name, err)
				continue
			}

			return true
		}
	}

	return false
}

func (r RenameHandler) ApplicationNamesHandler(m *monitors.Monitors) bool {
	for _, monitor := range *m {
		for _, desktop := range monitor.Desktops {
			clients := desktop.Clients()
			var names []string
			for _, client := range clients {
				names = append(names, client.ClassName)
			}
			desktopName := strings.Join(names, " ")
			fmt.Println(desktopName)
			if desktop.Name == desktopName {
				continue
			}

			err := desktop.Rename(desktopName)
			if err != nil {
				fmt.Println("Unable to rename desktop: ", desktop.Name, err)
				continue
			}

			return true
		}
	}

	return false
}

func (r RenameHandler) Handle(m *monitors.Monitors) bool {
	return r.handler(m)
}
