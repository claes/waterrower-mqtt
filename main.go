package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var debug *bool

type WaterrowerMQTTBridge struct {
	eventChannel chan AtomicEvent
	MQTTClient   mqtt.Client
	S4           *S4
}

func NewWaterrowerMQTTBridge(waterrowerUSBDevice *string, mqttBroker string) *WaterrowerMQTTBridge {

	eventChannel := make(chan AtomicEvent)
	s4 := NewS4(*waterrowerUSBDevice, eventChannel, nil, *debug)

	opts := mqtt.NewClientOptions().AddBroker(mqttBroker)
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("Could not connect to broker %s, %v\n", mqttBroker, token.Error())
		panic(token.Error())
	} else if *debug {
		fmt.Printf("Connected to MQTT broker: %s\n", mqttBroker)
	}

	bridge := &WaterrowerMQTTBridge{
		eventChannel: eventChannel,
		MQTTClient:   client,
		S4:           s4,
	}

	funcs := map[string]func(client mqtt.Client, message mqtt.Message){
		// "samsungremote/key/send":          bridge.onKeySend,
		// "samsungremote/key/reconnectsend": bridge.onKeyReconnectSend,
	}
	for key, function := range funcs {
		token := client.Subscribe(key, 0, function)
		token.Wait()
	}
	time.Sleep(2 * time.Second)
	return bridge
}

var sendMutex sync.Mutex

func (bridge *WaterrowerMQTTBridge) publishEvents() {
	for {
		event := <-bridge.eventChannel
		jsonData, err := json.Marshal(event)
		if err != nil {
			fmt.Println("Error serializing to JSON:", err)
			continue
		}
		bridge.PublishMQTT("waterrower/event", string(jsonData), false)
		bridge.PublishMQTT("waterrower/event/influx",
			fmt.Sprintf("event label=%s,value=%d,time=%d",
				event.Label, event.Value, event.Time), false)
	}
}

func (bridge *WaterrowerMQTTBridge) publishAggregatedData() {
	for {
		time.Sleep(1 * time.Second)
		jsonData, err := json.Marshal(bridge.S4.aggregator.event)
		if err != nil {
			fmt.Println("Error serializing to JSON:", err)
			continue
		}
		bridge.PublishMQTT("waterrower/aggregated", string(jsonData), false)
		bridge.PublishMQTT("waterrower/aggregated/influx",
			fmt.Sprintf("aggregated time_start=%d,time=%d,start_distance_meters=%d,total_distance_meters=%d,"+
				"stroke_rate=%d,watts=%d,calories=%d,speed_m_s=%f,heart_rate=%d",
				bridge.S4.aggregator.event.Time_start,
				bridge.S4.aggregator.event.Time,
				bridge.S4.aggregator.event.Start_distance_meters,
				bridge.S4.aggregator.event.Total_distance_meters,
				bridge.S4.aggregator.event.Stroke_rate,
				bridge.S4.aggregator.event.Watts,
				bridge.S4.aggregator.event.Calories,
				bridge.S4.aggregator.event.Speed_m_s,
				bridge.S4.aggregator.event.Heart_rate), false)
	}
}

func (bridge *WaterrowerMQTTBridge) PublishMQTT(topic string, message string, retained bool) {
	token := bridge.MQTTClient.Publish(topic, 0, retained, message)
	token.Wait()
}

var distance uint64
var duration time.Duration

func (bridge *WaterrowerMQTTBridge) MainLoop() {
	distance = 2000
	duration = 0
	workout := NewS4Workout()
	workout.AddSingleWorkout(duration, distance)
	go bridge.publishEvents()
	//go bridge.publishAggregatedData()
	bridge.S4.Run(&S4Workout{})
}

func printHelp() {
	fmt.Println("Usage: samsung-mqtt [OPTIONS]")
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

	bridge := NewWaterrowerMQTTBridge(usbDevice, *mqttBroker)

	go func() {
		for {
			time.Sleep(8 * time.Second)
			//bridge.reconnectIfNeeded()
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	fmt.Printf("Started\n")
	go bridge.MainLoop()
	<-c
	fmt.Printf("Shut down\n")

	os.Exit(0)
}
