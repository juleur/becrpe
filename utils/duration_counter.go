package utils

import (
	"strconv"
	"strings"
	"time"
)

// DurationCounter takes a duration(hours:mins:seconds) array
// then adds duration each other
func DurationCounter(durations []string) string {
	var finalDuration time.Duration
	for _, dur := range durations {
		durSplit := strings.Split(dur, ":")
		d, err := strconv.Atoi(durSplit[0])
		if err != nil {
			d += 0
		}
		finalDuration += time.Duration(d) * time.Hour
		d, err = strconv.Atoi(durSplit[1])
		if err != nil {
			d += 0
		}
		finalDuration += time.Duration(d) * time.Minute
		d, err = strconv.Atoi(durSplit[2])
		if err != nil {
			d += 0
		}
		finalDuration += time.Duration(d) * time.Second
	}
	duration := func(r rune) rune {
		if r == 'h' || r == 'm' {
			return ':'
		}
		return r
	}
	// use slice to remove 1h32m124s from duration
	return strings.Map(duration, finalDuration.String()[:len(finalDuration.String())-1])
}
