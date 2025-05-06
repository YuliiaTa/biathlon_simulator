package main

import (
	"fmt"
	"sort"
	"time"
)

func (s *Simulation) GenerateReport() {
	fmt.Println("\n FINAL REPORT")

	fmt.Println("Participants and their statuses:")
	for id, comp := range s.Competitors {
		fmt.Printf("Participant %d: status=%s, laps=%d/%d\n",
			id, comp.Status, comp.CurrentLap, s.Config.Laps)
	}

	var competitors []*Competitor
	for _, comp := range s.Competitors {
		competitors = append(competitors, comp)
	}

	// Sort by finish time
	sort.Slice(competitors, func(i, j int) bool {
		ci, cj := competitors[i], competitors[j]
		if ci.Status == "Finished" && cj.Status == "Finished" {
			return ci.TotalTime() < cj.TotalTime()
		}
		return ci.Status == "Finished"
	})

	for _, comp := range competitors {
		switch comp.Status {
		case "NotStarted":
			fmt.Printf("[NotStarted] %d\n", comp.ID)
		case "NotFinished":
			fmt.Printf("[NotFinished] %d\n", comp.ID)
		case "Finished":
			// Format lap times
			var lapInfo []string
			for i := 0; i < s.Config.Laps; i++ {
				if i < len(comp.LapTimes) {
					lapTime := formatDuration(comp.LapTimes[i])
					speed := float64(s.Config.LapLen) / comp.LapTimes[i].Seconds()
					lapInfo = append(lapInfo, fmt.Sprintf("{%s, %.3f}", lapTime, speed))
				} else {
					lapInfo = append(lapInfo, "{,}")
				}
			}

			// Format penalty time
			penaltyTimeStr := formatDuration(comp.PenaltyTime)
			penaltySpeed := 0.0
			if comp.PenaltyTime > 0 {
				penaltySpeed = float64(s.Config.PenaltyLen) / comp.PenaltyTime.Seconds()
			}

			// Total time
			totalTime := formatDuration(comp.ActualStart.Sub(comp.StartTime) + comp.TotalTime())

			fmt.Printf("%d %s %v {%s, %.3f} %d/%d\n",
				comp.ID,
				totalTime,
				lapInfo,
				penaltyTimeStr,
				penaltySpeed,
				comp.Hits,
				comp.Shots,
			)
		}
	}
}

func (c *Competitor) TotalTime() time.Duration {
	var total time.Duration
	for _, lap := range c.LapTimes {
		total += lap
	}
	return total + c.PenaltyTime
}


func formatDuration(d time.Duration) string {
	total := int(d.Seconds())
	hours := total / 3600
	minutes := (total % 3600) / 60
	seconds := total % 60
	millis := (d.Milliseconds() % 1000)
	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, millis)
}
