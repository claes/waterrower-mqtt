package lib

// Copied from https://github.com/olympum/oarsman and adjusted
// Copyright olympum

import (
	"container/list"
	"fmt"
	"log/slog"
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
		slog.Info("Starting single duration workout", "durationSeconds", durationSeconds)
		if durationSeconds >= 18000 {
			slog.Info("Workout time must be less than 18,000 seconds", "durationSeconds", durationSeconds)
		}
		payload := fmt.Sprintf("%04X", durationSeconds)
		workoutPacket = Packet{cmd: WorkoutSetDurationRequest, data: []byte(payload)}
	} else if distanceMeters > 0 {
		slog.Info("Starting single distance workout", "distanceMeters", distanceMeters)
		if distanceMeters >= 64000 {
			slog.Info("Workout distance must be less than 64,000 meters", "distanceMeters", distanceMeters)
		}
		payload := Meters + fmt.Sprintf("%04X", distanceMeters)
		workoutPacket = Packet{cmd: WorkoutSetDistanceRequest, data: []byte(payload)}
	} else {
		slog.Debug("Undefined workout", "durationSeconds", durationSeconds, "distanceMeters", distanceMeters)
	}
	workout.workoutPackets.PushFront(workoutPacket)
}
