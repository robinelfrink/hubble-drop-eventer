package main

import (
	"flag"
	event "hubble-drop-eventer/event"
	observe "hubble-drop-eventer/observe"
	"log"
)

var (
	server = flag.String("server", "localhost",
		"Hubble server address")
	port = flag.Int("port", 4245,
		"Hubble server port")
)

func main() {
	flag.Parse()

	eventChannel := make(chan event.DropEvent, 100)

	eventer, err := event.New(eventChannel)
	if err != nil {
		log.Fatalf("Eventer error: %v", err)
	}
	go eventer.Run()

	observer, err := observe.New(*server, *port, eventChannel)
	if err != nil {
		log.Fatalf("Observer error: %v", err)
	}

	observer.Run()

}
