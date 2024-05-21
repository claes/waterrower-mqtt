package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/claes/waterrower-mqtt/lib"
)

var debug *bool

func printHelp() {
	fmt.Println("Usage: waterrower-mqtt [OPTIONS]")
	fmt.Println("Options:")
	flag.PrintDefaults()
}

func main() {
	usbDevice := flag.String("usbdevice", "", "Waterrower USB Device")
	mqttBroker := flag.String("broker", "tcp://localhost:1883", "MQTT broker URL")
	help := flag.Bool("help", false, "Print help")
	debug = flag.Bool("debug", false, "Debug logging")
	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	s4, eventChannel, aggregateEventChannel := lib.CreateWaterrowerDevice(usbDevice)
	bridge := lib.NewWaterrowerMQTTBridge(
		s4, eventChannel, aggregateEventChannel,
		lib.CreateMQTTClient(*mqttBroker))

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	fmt.Printf("Started\n")
	go bridge.MainLoop()
	<-c
	bridge.S4.Exit()
	fmt.Printf("Shut down\n")

	os.Exit(0)
}
