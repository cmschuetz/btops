package monitors

import (
	"encoding/json"
	"strconv"

	"github.com/cmschuetz/bspwm-desktops/ipc"
)

type bspwmState struct {
	Monitors Monitors
}

type Monitor struct {
	Name     string
	Id       int
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

func (d Desktop) Clients() (clients []Client) {
	nodes := d.Nodes()

	for _, node := range nodes {
		if node.Client == nil {
			continue
		}

		clients = append(clients, *node.Client)
	}

	return clients
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
	if _, err := ipc.Send("monitor", "-a", name); err != nil {
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
