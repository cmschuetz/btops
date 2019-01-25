package monitors

import (
	"encoding/json"
	"sort"
	"strconv"

	"github.com/cmschuetz/btops/ipc"
)

type bspwmState struct {
	Monitors Monitors
}

type Monitor struct {
	Name     string
	Id       int
	FocusedDesktopId int
	Desktops []Desktop
}

type Monitors []Monitor

type Desktop struct {
	Name string
	Id   int
	Root *Node
}

type Node struct {
	Id          int
	Client      *Client
	FirstChild  *Node
	SecondChild *Node
}

type Client struct {
	ClassName string
}

type Clients struct {
	clients map[string]int
}

func newClients(nodes []*Node) (clients Clients) {
	clients.clients = make(map[string]int, len(nodes))

	for _, node := range nodes {
		if node.Client == nil {
			continue
		}

		clients.clients[node.Client.ClassName]++
	}

	return clients
}

func (c Clients) Names() (names []string) {
	names = make([]string, 0, len(c.clients))

	for key := range c.clients {
		names = append(names, key)
	}

	sort.Strings(names)
	return names
}

func GetMonitors() (*Monitors, error) {
	jsonState, err := ipc.Send("wm", "-d")
	if err != nil {
		return nil, err
	}

	var state bspwmState
	if err = json.Unmarshal(jsonState, &state); err != nil {
		return nil, err
	}

	return &state.Monitors, nil
}

func (d Desktop) IsEmpty() bool {
	return d.Root == nil
}

func (d Desktop) Clients() (clients Clients) {
	return newClients(d.Nodes())
}

func (d Desktop) Nodes() (nodes []*Node) {
	collectNodes(d.Root, &nodes)
	return nodes
}

func collectNodes(node *Node, nodes *[]*Node) {
	if node == nil {
		return
	}

	*nodes = append(*nodes, node)
	collectNodes(node.FirstChild, nodes)
	collectNodes(node.SecondChild, nodes)
}

func (d *Desktop) Rename(name string) error {
	if _, err := ipc.Send("desktop", strconv.Itoa(d.Id), "-n", name); err != nil {
		return err
	}

	d.Name = name
	return nil
}

func (m *Monitor) AppendDesktop(name string) error {
	if _, err := ipc.Send("monitor", m.Name, "-a", name); err != nil {
		return err
	}

	m.Desktops = append(m.Desktops, Desktop{Name: name})
	return nil
}

func (m *Monitor) RemoveDesktop(id int) error {
	if _, err := ipc.Send("desktop", strconv.Itoa(id), "-r"); err != nil {
		return err
	}

	for i := range m.Desktops {
		if m.Desktops[i].Id != id {
			continue
		}

		m.Desktops = append(m.Desktops[:i], m.Desktops[i+1:]...)
		break
	}

	return nil
}

func (m *Monitor) EmptyDesktops() (desktops []*Desktop) {
	for i := range m.Desktops {
		if !m.Desktops[i].IsEmpty() {
			continue
		}

		desktops = append(desktops, &m.Desktops[i])
	}

	return desktops
}
