package main

import (
	"fmt"

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
