package lib

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type WaterrowerMQTTBridge struct {
	eventChannel          chan AtomicEvent
	aggregateEventChannel chan AggregateEvent
	MQTTClient            mqtt.Client
	S4                    *S4
}

func CreateWaterrowerDevice(waterrowerUSBDevice *string) (*S4, chan AtomicEvent, chan AggregateEvent) {
	eventChannel := make(chan AtomicEvent)
	aggregateEventChannel := make(chan AggregateEvent)
	s4 := NewS4(*waterrowerUSBDevice, eventChannel, aggregateEventChannel)
	return s4, eventChannel, aggregateEventChannel
}

func CreateMQTTClient(mqttBroker string) mqtt.Client {
	slog.Info("Creating MQTT client", "broker", mqttBroker)
	opts := mqtt.NewClientOptions().AddBroker(mqttBroker)
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		slog.Error("Could not connect to broker", "mqttBroker", mqttBroker, "error", token.Error())
		panic(token.Error())
	}
	slog.Info("Connected to MQTT broker", "mqttBroker", mqttBroker)
	return client
}

func NewWaterrowerMQTTBridge(s4 *S4, eventChannel chan AtomicEvent, aggregateEventChannel chan AggregateEvent,
	client mqtt.Client) *WaterrowerMQTTBridge {

	bridge := &WaterrowerMQTTBridge{
		eventChannel:          eventChannel,
		aggregateEventChannel: aggregateEventChannel,
		MQTTClient:            client,
		S4:                    s4,
	}

	time.Sleep(2 * time.Second)
	return bridge
}

// func NewWaterrowerMQTTBridge(waterrowerUSBDevice *string, mqttBroker string) *WaterrowerMQTTBridge {

// 	eventChannel := make(chan AtomicEvent)
// 	aggregateEventChannel := make(chan AggregateEvent)
// 	s4 := NewS4(*waterrowerUSBDevice, eventChannel, aggregateEventChannel, *debug)

// 	opts := mqtt.NewClientOptions().AddBroker(mqttBroker)
// 	client := mqtt.NewClient(opts)
// 	if token := client.Connect(); token.Wait() && token.Error() != nil {
// 		slog.Error("Could not connect to broker", "broker", mqttBroker, "error", token.Error())
// 		panic(token.Error())
// 	}
// 	slog.Info("Connected to MQTT broker", "broker", mqttBroker)

// 	bridge := &WaterrowerMQTTBridge{
// 		eventChannel:          eventChannel,
// 		aggregateEventChannel: aggregateEventChannel,
// 		MQTTClient:            client,
// 		S4:                    s4,
// 	}

// 	time.Sleep(2 * time.Second)
// 	return bridge
// }

var sendMutex sync.Mutex

func (bridge *WaterrowerMQTTBridge) publishEvents() {
	for {
		event := <-bridge.eventChannel
		jsonData, err := json.Marshal(event)
		if err != nil {
			slog.Error("Error serializing to JSON:", "error", err)
			continue
		}
		bridge.PublishMQTT("waterrower/event", string(jsonData), false)
		bridge.PublishMQTT("waterrower/event/influx",
			fmt.Sprintf("event label=\"%s\",value=%du,time=%di %d000000",
				event.Label, event.Value, event.Time, event.Time), false)
	}
}

func (bridge *WaterrowerMQTTBridge) publishAggregateEvents() {
	for {
		event := <-bridge.aggregateEventChannel
		jsonData, err := json.Marshal(event)
		if err != nil {
			slog.Error("Error serializing to JSON:", "error", err)
			continue
		}
		bridge.PublishMQTT("waterrower/aggregated", string(jsonData), false)
		bridge.PublishMQTT("waterrower/aggregated/influx",
			fmt.Sprintf("aggregated time_start=%di,time=%di,start_distance_meters=%du,total_distance_meters=%du,"+
				"stroke_rate=%du,watts=%du,calories=%du,speed_m_s=%f,heart_rate=%du %d000000",
				event.Time_start,
				event.Time,
				event.Start_distance_meters,
				event.Total_distance_meters,
				event.Stroke_rate,
				event.Watts,
				event.Calories,
				event.Speed_m_s,
				event.Heart_rate,
				event.Time), false)
	}

	/*
		Time_start            int64
		Time                  int64
		Start_distance_meters uint64
		Total_distance_meters uint64
		Stroke_rate           uint64
		Watts                 uint64
		Calories              uint64
		Speed_m_s             float64
		Heart_rate            uint64
	*/
}

func (bridge *WaterrowerMQTTBridge) publishAggregatedData() {
	for {
		time.Sleep(1 * time.Second)
		jsonData, err := json.Marshal(bridge.S4.aggregator.event)
		if err != nil {
			slog.Error("Error serializing to JSON:", "error", err)
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
	go bridge.publishAggregateEvents()

	bridge.S4.Run(&workout)
}
