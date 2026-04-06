package service

import "time"

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now()
}

func NewSystemClock() Clock {
	return systemClock{}
}
