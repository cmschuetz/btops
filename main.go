package main

import (
	"fmt"
	"strconv"

	"github.com/cmschuetz/bspwm-desktops/ipc"
	"github.com/cmschuetz/bspwm-desktops/rules"

	"github.com/cmschuetz/bspwm-desktops/monitors"
)

func main() {
	c, err := rules.GetConfig()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(c)

	handlers := rules.NewHandlers(c)

	sub, err := ipc.NewSubscriber()
	if err != nil {
		fmt.Println(err)
	}
	defer sub.Close()

	for sub.Scanner.Scan() {
		monitors, err := monitors.GetMonitors()
		if err != nil {
			fmt.Println("Unable to obtain monitors:", err)
		}

		handlers.Handle(monitors)
	}
}

func adjustDesktops(monitors *monitors.Monitors) error {
	for _, monitor := range *monitors {
		// Remove empty desktops
		for _, desktop := range monitor.EmptyDesktops() {
			if *desktop == monitor.Desktops[len(monitor.Desktops)-1] {
				continue
			}

			err := monitor.RemoveDesktop(desktop.Id)
			if err == nil {
				return nil
			}

			fmt.Println("Unable to remove desktop:", err)
		}

		// Append desktops if needed
		if !monitor.Desktops[len(monitor.Desktops)-1].IsEmpty() {
			err := monitor.AppendDesktop(strconv.Itoa(len(monitor.Desktops) + 1))
			if err == nil {
				return nil
			}

			fmt.Println("Unable to append desktop:", err)
		}

		// Rename desktops if needed
		for i, desktop := range monitor.Desktops {
			name := strconv.Itoa(i + 1)
			if desktop.Name == name {
				continue
			}

			err := desktop.Rename(name)
			if err == nil {
				return nil
			}

			fmt.Println("Unable to rename desktop:", err)
		}
	}

	return nil
}
