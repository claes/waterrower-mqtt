package main

// Copied from https://github.com/olympum/oarsman and adjusted
// Copyright olympum

import (
	"container/list"
	"fmt"
	"time"
)

type S4Workout struct {
	workoutPackets *list.List
	state          int
}

func NewS4Workout() S4Workout {
	workout := S4Workout{workoutPackets: list.New(), state: Unset}
	return workout
}

func (workout S4Workout) AddSingleWorkout(duration time.Duration, distanceMeters uint64) {
	// prepare workout instructions
	durationSeconds := uint64(duration.Seconds())
	var workoutPacket Packet

	if durationSeconds > 0 {
		fmt.Printf("Starting single duration workout: %d seconds\n", durationSeconds)
		if durationSeconds >= 18000 {
			fmt.Printf("Workout time must be less than 18,000 seconds (was %d)\n", durationSeconds)
		}
		payload := fmt.Sprintf("%04X", durationSeconds)
		workoutPacket = Packet{cmd: WorkoutSetDurationRequest, data: []byte(payload)}
	} else if distanceMeters > 0 {
		fmt.Printf("Starting single distance workout: %d meters\n", distanceMeters)
		if distanceMeters >= 64000 {
			fmt.Printf("Workout distance must be less than 64,000 meters (was %d)\n", distanceMeters)
		}
		payload := Meters + fmt.Sprintf("%04X", distanceMeters)
		workoutPacket = Packet{cmd: WorkoutSetDistanceRequest, data: []byte(payload)}
	} else {
		printDebug("Undefined workout")
	}
	workout.workoutPackets.PushFront(workoutPacket)
}
