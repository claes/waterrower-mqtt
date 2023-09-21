package main

// Copied from https://github.com/olympum/oarsman and adjusted
// Copyright olympum

import "fmt"

type AggregateEvent struct {
	Time_start            int64
	Time                  int64
	Start_distance_meters uint64
	Total_distance_meters uint64
	Stroke_rate           uint64
	Watts                 uint64
	Calories              uint64
	Speed_m_s             float64
	Heart_rate            uint64
}

const MAX_RESOLUTION_MILLIS = 10000

type Aggregator struct {
	event                 *AggregateEvent
	atomicEventChannel    chan<- AtomicEvent
	aggregateEventChannel chan<- AggregateEvent
}

func newAggregator(atomicEventChannel chan<- AtomicEvent, aggregateEventChannel chan<- AggregateEvent) Aggregator {
	return Aggregator{
		atomicEventChannel:    atomicEventChannel,
		aggregateEventChannel: aggregateEventChannel,
		event:                 &AggregateEvent{}}
}

func (aggregator *Aggregator) send(event *AggregateEvent) bool {
	if aggregator.aggregateEventChannel == nil {
		fmt.Println("(send) aggregator.aggregateEventChannel == nil")
		return false
	}

	toBeSent := *event

	newEvent := AggregateEvent{}
	newEvent.Start_distance_meters = toBeSent.Total_distance_meters
	newEvent.Total_distance_meters = toBeSent.Total_distance_meters
	aggregator.event = &newEvent

	aggregator.aggregateEventChannel <- toBeSent
	fmt.Println("Sent aggregate event", event)
	return true
}

func (aggregator *Aggregator) complete() {
	e := aggregator.event
	delta_time := float64(e.Time - e.Time_start)
	delta_distance := float64(e.Total_distance_meters - e.Start_distance_meters)
	if delta_time > 0 && delta_distance > 0 {
		e.Speed_m_s = delta_distance * 1000.0 / delta_time
		aggregator.send(e)
	}

}

func (aggregator *Aggregator) consume(atomicEvent AtomicEvent) {
	if aggregator.atomicEventChannel != nil {
		aggregator.atomicEventChannel <- atomicEvent
		fmt.Println("Sent atomic event", atomicEvent)
	}

	if aggregator.aggregateEventChannel == nil {
		fmt.Println("(consume) aggregator.aggregateEventChannel == nil")
		return
	}

	aggregateEvent := aggregator.event
	fmt.Println("Current aggregate event", aggregateEvent)
	if aggregateEvent.Time_start == 0 {
		aggregateEvent.Time_start = atomicEvent.Time
	}
	aggregateEvent.Time = atomicEvent.Time

	v := atomicEvent.Value
	switch atomicEvent.Label {
	case "total_distance_meters":
		aggregateEvent.Total_distance_meters = v
		if v == 0 {
			aggregator.send(aggregateEvent)
		}
	case "stroke_rate":
		aggregateEvent.Stroke_rate = v
	case "watts":
		if v > 0 {
			aggregateEvent.Watts = v
		}
	case "calories":
		aggregateEvent.Calories = v
	case "heart_rate":
		if v > 0 {
			aggregateEvent.Heart_rate = v
		}
	}

	if aggregateEvent.Time-aggregateEvent.Time_start >= MAX_RESOLUTION_MILLIS {
		aggregator.complete()
	}

	// auto-laps every 2000 meters
	if aggregateEvent.Total_distance_meters%2000 == 0 {
		aggregator.complete()
	}

	fmt.Println("Current aggregate event", aggregateEvent)
}
