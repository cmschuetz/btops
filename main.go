package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/cmschuetz/bspwm-desktops/ipc"
)

const (
	focusedMonitor           = "M"
	unfocusedMonitor         = "m"
	freeFocusedDesktop       = "F"
	freeUnfocusedDesktop     = "f"
	occupiedFocusedDesktop   = "O"
	occupiedUnfocusedDesktop = "o"
	urgentFocusedDesktop     = "U"
	urgentUnfocusedDesktop   = "u"
)

type report []*monitor
type item string

type monitor struct {
	name     string
	focused  bool
	modified bool
	desktops []*desktop
}

type desktop struct {
	name    string
	focused bool
	free    bool
}

func main() {
	sub, err := ipc.NewSubscriber()
	if err != nil {
		fmt.Println(err)
	}
	defer sub.Close()

	for sub.Scanner.Scan() {
		r := newReport(sub.Scanner.Text())

		for _, m := range *r {
			d := m.desktopToRemove()

			if d != "" {
				fmt.Println("desktop to remove:", d)
				dID, err := m.desktopID(d)
				if err != nil {
					fmt.Println(err)
				} else {
					fmt.Println("Attempting to remove", dID)
					err = removeDesktop(dID)
					if err != nil {
						fmt.Println(err)
					} else {
						fmt.Println("Desktop", d, "removed")
						break
					}
				}
			}

			d = m.desktopToAdd()

			if d != "" {
				fmt.Println("Monitor to add to:", m.name)
				err = addDesktop(m.name, d)
				if err != nil {
					fmt.Println(err)
				} else {
					break
				}
			}
		}
	}
}

func newReport(rawReport string) *report {
	var r report
	var m *monitor
	var d *desktop

	items := strings.Split(strings.TrimPrefix(rawReport, "W"), ":")
	for _, strItem := range items {
		i := item(strItem)
		switch {
		case i.IsMonitor():
			m = newMonitor(i)
			r = append(r, m)
		case i.IsDesktop():
			d = newDesktop(i)
			m.desktops = append(m.desktops, d)
		}
	}

	return &r
}

func newMonitor(monitorItem item) *monitor {
	return &monitor{
		name:    string(monitorItem)[1:],
		focused: monitorItem.IsFocused(),
	}
}

func newDesktop(desktopItem item) *desktop {
	return &desktop{
		name:    string(desktopItem[1:]),
		focused: desktopItem.IsFocused(),
		free:    desktopItem.IsFree(),
	}
}

func (i item) IsMonitor() bool {
	switch string(i[0]) {
	case focusedMonitor, unfocusedMonitor:
		return true
	}

	return false
}

func (i item) IsDesktop() bool {
	switch string(i[0]) {
	case
		occupiedFocusedDesktop, occupiedUnfocusedDesktop,
		freeFocusedDesktop, freeUnfocusedDesktop,
		urgentFocusedDesktop, urgentUnfocusedDesktop:
		return true
	}

	return false
}

func (i item) IsFree() bool {
	switch string(i[0]) {
	case freeFocusedDesktop, freeUnfocusedDesktop:
		return true
	}

	return false
}

func (i item) IsFocused() bool {
	switch string(i[0]) {
	case
		focusedMonitor, freeFocusedDesktop,
		occupiedFocusedDesktop, urgentFocusedDesktop:
		return true
	}

	return false
}

func (m monitor) desktopID(desktopName string) (name string, err error) {
	rawMDesktops, err := exec.Command("bspc", "query", "-D", "-m", m.name).Output()
	if err != nil {
		return name, err
	}

	mDesktops := strings.Fields(string(rawMDesktops))

	rawMDesktopNames, err := exec.Command("bspc", "query", "-D", "-m", m.name, "--names").Output()
	if err != nil {
		return name, err
	}

	mDesktopNames := strings.Fields(string(rawMDesktopNames))

	var goodIds []string
	for i, dID := range mDesktops {
		if mDesktopNames[i] == desktopName {
			goodIds = append(goodIds, dID)
		}
	}

	for _, id := range goodIds {
		rawFreeDesktops, _ := exec.Command("bspc", "query", "-D", "-d", fmt.Sprintf("%s.%s", id, "!occupied")).Output()
		// if err != nil {
		// 	return name, err
		// }

		elID := strings.TrimSpace(string(rawFreeDesktops))

		if elID == id {
			return elID, nil
		}
	}

	return name, nil
}

func (m monitor) desktopToRemove() (name string) {
	for i := range m.desktops {
		if m.desktops[i].free && i != len(m.desktops)-1 {
			return m.desktops[i].name
		}
	}

	return name
}

func (m monitor) desktopToAdd() (name string) {
	if m.desktops[len(m.desktops)-1].free {
		return name
	}

	return strconv.Itoa(len(m.desktops) + 1)
}

func addDesktop(monitor, name string) error {
	return exec.Command("bspc", "monitor", monitor, "-a", name).Run()
}

func removeDesktop(desktop string) error {
	return exec.Command("bspc", "desktop", desktop, "-r").Run()
}
