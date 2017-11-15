package rules

import (
	"fmt"
	"math"
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
	Min                int
	Max                int
	RemoveEmpty        bool `mapstructure:"remove-empty"`
	AppendWhenOccupied bool `mapstructure:"append-when-occupied"`
	Renamers           []string
	ConstantName       string                `mapstructure:"constant-name"`
	StaticNames        []string              `mapstructure:"static-names"`
	ClassifiedNames    []map[string][]string `mapstructure:"classified-names"`
}

type baseHandler struct {
	config *Config
}

type handlerFunc func(*monitors.Monitors) bool

type AppendHandler struct{ baseHandler }
type RemoveHandler struct{ baseHandler }
type RenameHandler struct {
	baseHandler
	renamers []Renamer
}

type Renamer interface {
	Initialize(*Config)
	CanRename(*monitors.Desktop, int) bool
	Rename(*monitors.Desktop, int) bool
}

type constantRenamer struct{ name string }
type staticRenamer struct{ names []string }
type clientRenamer struct{}
type numericRenamer struct{}

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

	c.SetDefault("min", 1)
	c.SetDefault("max", math.MaxInt64)
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

		if a.config.Max <= dCount {
			continue
		}

		for i, desktop := range monitor.Desktops {
			if desktop.IsEmpty() {
				break
			}

			if i == dCount-1 || a.config.Min > dCount {
				err := monitor.AppendDesktop("")
				if err != nil {
					fmt.Println("Unable to append desktop to monitor: ", monitor.Name, err)
					continue
				}

				return true
			}
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

			if r.config.Min >= len(monitor.Desktops) {
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
	r.renamers = *NewRenamers(c)
}

func NewRenamers(c *Config) *[]Renamer {
	var renamers []Renamer
	var renamer Renamer

	for _, r := range c.Renamers {
		switch r {
		case "constant":
			renamer = &constantRenamer{}
		case "static":
			renamer = &staticRenamer{}
		case "client":
			renamer = &clientRenamer{}
		case "numeric":
			renamer = &numericRenamer{}
		default:
			continue
		}

		renamer.Initialize(c)
		renamers = append(renamers, renamer)
	}

	return &renamers
}

func (r RenameHandler) ShouldHandle() bool {
	return len(r.renamers) > 0
}

func (r RenameHandler) Handle(m *monitors.Monitors) bool {
	for _, monitor := range *m {
		for i, desktop := range monitor.Desktops {
			for _, renamer := range r.renamers {
				if !renamer.CanRename(&desktop, i) {
					continue
				}

				if !renamer.Rename(&desktop, i) {
					break
				}

				return true
			}
		}
	}

	return false
}

func (c *constantRenamer) Initialize(conf *Config) {
	c.name = conf.ConstantName
}

func (c constantRenamer) CanRename(desktop *monitors.Desktop, desktopIdx int) bool {
	return true
}

func (c constantRenamer) Rename(desktop *monitors.Desktop, desktopIdx int) bool {
	if desktop.Name == c.name {
		return false
	}

	err := desktop.Rename(c.name)
	if err != nil {
		fmt.Println("Unable to rename desktop: ", desktop.Name, err)
		return false
	}

	return true
}

func (s *staticRenamer) Initialize(conf *Config) {
	s.names = conf.StaticNames
}

func (s staticRenamer) CanRename(desktop *monitors.Desktop, desktopIdx int) bool {
	if desktopIdx >= len(s.names) {
		return false
	}

	return true
}

func (s staticRenamer) Rename(desktop *monitors.Desktop, desktopIdx int) bool {
	if desktop.Name == s.names[desktopIdx] {
		return false
	}

	err := desktop.Rename(s.names[desktopIdx])
	if err != nil {
		fmt.Println("Unable to rename desktop: ", desktop.Name, err)
		return false
	}

	return true
}

func (c *clientRenamer) Initialize(conf *Config) {}
func (c clientRenamer) CanRename(desktop *monitors.Desktop, desktopIdx int) bool {
	return len(desktop.Clients().Names()) > 0
}
func (c clientRenamer) Rename(desktop *monitors.Desktop, desktopIdx int) bool {
	name := strings.Join(desktop.Clients().Names(), " ")

	if desktop.Name == name {
		return false
	}

	err := desktop.Rename(name)
	if err != nil {
		fmt.Println("Unable to rename desktop: ", desktop.Name, err)
		return false
	}

	return true
}

func (n *numericRenamer) Initialize(conf *Config) {}
func (n numericRenamer) CanRename(desktop *monitors.Desktop, desktopIdx int) bool {
	return true
}
func (n numericRenamer) Rename(desktop *monitors.Desktop, desktopIdx int) bool {
	numericName := strconv.Itoa(desktopIdx + 1)

	if desktop.Name == numericName {
		return false
	}

	err := desktop.Rename(numericName)
	if err != nil {
		fmt.Println("Unable to rename desktop: ", desktop.Name, err)
		return false
	}

	return true
}
