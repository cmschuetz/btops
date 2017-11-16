package main

import (
	"fmt"
	"log"

	"github.com/cmschuetz/bspwm-desktops/ipc"
	"github.com/cmschuetz/bspwm-desktops/monitors"
	"github.com/cmschuetz/bspwm-desktops/rules"
)

func main() {
	for {
		listen()
	}
}

func listen() {
	c, err := rules.GetConfig()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(c)

	handlers := rules.NewHandlers(c)

	sub, err := ipc.NewSubscriber()
	if err != nil {
		log.Fatal(err)
	}
	defer sub.Close()

	for !c.ConfigChanged() && sub.Scanner.Scan() {
		monitors, err := monitors.GetMonitors()
		if err != nil {
			fmt.Println("Unable to obtain monitors:", err)
		}

		handlers.Handle(monitors)
	}
}
