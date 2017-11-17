package main

import (
	"fmt"
	"log"

	"github.com/cmschuetz/bspwm-desktops/config"
	"github.com/cmschuetz/bspwm-desktops/handlers"
	"github.com/cmschuetz/bspwm-desktops/ipc"
	"github.com/cmschuetz/bspwm-desktops/monitors"
)

func main() {
	for {
		listen()
	}
}

func listen() {
	c, err := config.GetConfig()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(c)

	handlers := handlers.NewHandlers(c)

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
