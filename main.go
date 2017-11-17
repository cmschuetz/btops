package main

import (
	"fmt"
	"log"

	"github.com/cmschuetz/btops/config"
	"github.com/cmschuetz/btops/handlers"
	"github.com/cmschuetz/btops/ipc"
	"github.com/cmschuetz/btops/monitors"
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
