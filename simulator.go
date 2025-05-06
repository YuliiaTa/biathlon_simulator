package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func parseEvents(filename string) ([]Event, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read events file: %w", err)
	}

	content := strings.ReplaceAll(string(file), "\r", "")
	lines := strings.Split(content, "\n")

	var events []Event

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			return nil, fmt.Errorf("invalid event format: %s", line)
		}

		timeStr := strings.Trim(parts[0], "[]")
		eventTime, err := time.Parse("15:04:05.000", timeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid time format in line '%s': %w", line, err)
		}

		eventType, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, fmt.Errorf("invalid event type in line '%s': %w", line, err)
		}

		competitorID, err := strconv.Atoi(strings.TrimSpace(parts[2]))
		if err != nil {
			return nil, fmt.Errorf("invalid competitor ID in line '%s': %w", line, err)
		}

		params := ""
		if len(parts) > 3 {
			params = strings.Join(parts[3:], " ")
		}

		events = append(events, Event{
			Time:       eventTime,
			Type:       eventType,
			Competitor: competitorID,
			Params:     params,
		})
	}

	return events, nil
}

type Simulation struct {
	Config      *Config
	Competitors map[int]*Competitor
	LastLapTime map[int]time.Time
}

func NewSimulation(config *Config) *Simulation {
	return &Simulation{
		Config:      config,
		Competitors: make(map[int]*Competitor),
		LastLapTime: make(map[int]time.Time),
	}
}

func (s *Simulation) ProcessEvent(event Event) {
	comp, exists := s.Competitors[event.Competitor]
	if !exists && event.Type != 1 {
		return
	}

	switch event.Type {
	case 1: // The competitor registered
		s.Competitors[event.Competitor] = &Competitor{
			ID:     event.Competitor,
			Status: "Registered",
		}
		fmt.Printf("[%s] Competitor(%d) registered\n",
			event.Time.Format("15:04:05.000"), event.Competitor)

	case 2: // The start time was set by a draw
		startTime, err := time.Parse("15:04:05.000", event.Params)
		if err != nil {
			return
		}
		comp.StartTime = startTime
		fmt.Printf("[%s] Start time for competitor(%d) set by draw to %s\n",
			event.Time.Format("15:04:05.000"), event.Competitor, event.Params)

		startDelta, _ := time.ParseDuration(strings.Replace(s.Config.StartDelta, ":", "h", 1) + "m0s")
		maxStartTime := comp.StartTime.Add(startDelta)
		if event.Time.After(maxStartTime) {
			comp.Status = "NotStarted"
			fmt.Printf("[%s] Competitor(%d) disqualified (late to start)\n",
				event.Time.Format("15:04:05.000"), event.Competitor)
		}

	case 3: // The competitor is on the start line
		startDelta, _ := time.ParseDuration(strings.Replace(s.Config.StartDelta, ":", "h", 1) + "m0s")
		maxStartTime := comp.StartTime.Add(startDelta)
		if event.Time.After(maxStartTime) {
			comp.Status = "NotStarted"
			fmt.Printf("[%s] Competitor(%d) disqualified (late to start)\n",
				event.Time.Format("15:04:05.000"), event.Competitor)
		}

	case 4: // The competitor has started
		comp.ActualStart = event.Time
		comp.Status = "Running"
		comp.CurrentLap = 1
		s.LastLapTime[event.Competitor] = event.Time
		fmt.Printf("[%s] Competitor(%d) started the race\n",
			event.Time.Format("15:04:05.000"), event.Competitor)

	case 5: // The competitor is on the firing range
		comp.OnRange = true
		fmt.Printf("[%s] Competitor(%d) entered firing range (%s)\n",
			event.Time.Format("15:04:05.000"), event.Competitor, event.Params)

	case 6: // The target has been hit
		comp.Hits++
		fmt.Printf("[%s] Target(%s) hit by competitor(%d)\n",
			event.Time.Format("15:04:05.000"), event.Params, event.Competitor)

	case 7: // The competitor left the firing range
		comp.OnRange = false
		comp.Shots += 5
		fmt.Printf("[%s] Competitor(%d) left firing range\n",
			event.Time.Format("15:04:05.000"), event.Competitor)

	case 8: // The competitor entered the penalty laps
		comp.OnPenalty = true
		comp.PenaltyStart = event.Time
		fmt.Printf("[%s] Competitor(%d) entered penalty laps\n",
			event.Time.Format("15:04:05.000"), event.Competitor)

	case 9: // The competitor left the penalty laps
		comp.OnPenalty = false
		comp.PenaltyTime += event.Time.Sub(comp.PenaltyStart)
		fmt.Printf("[%s] Competitor(%d) exited penalty laps\n",
			event.Time.Format("15:04:05.000"), event.Competitor)

	case 10: // The competitor ended the main lap
		if startTime, ok := s.LastLapTime[event.Competitor]; ok {
			lapDuration := event.Time.Sub(startTime)
			comp.LapTimes = append(comp.LapTimes, lapDuration)
			s.LastLapTime[event.Competitor] = event.Time
		}
		comp.CurrentLap++

		// Automatic finish after the final lap
		if comp.CurrentLap > s.Config.Laps {
			comp.Status = "Finished"
			comp.CurrentLap--
			fmt.Printf("[%s] Competitor(%d) finished (auto)\n",
				event.Time.Format("15:04:05.000"), event.Competitor)
		}

	case 11: // The competitor can’t continue
		comp.Status = "NotFinished"
		comp.Comment = event.Params
		fmt.Printf("[%s] Competitor(%d) cannot continue: %s\n",
			event.Time.Format("15:04:05.000"), event.Competitor, event.Params)

	case 32: // The competitor is disqualified
		comp.Status = "NotStarted"
		fmt.Printf("[%s] Competitor(%d) disqualified\n",
			event.Time.Format("15:04:05.000"), event.Competitor)

	case 33: // The competitor has finished
		fmt.Printf("[DEBUG] Checking finish for competitor %d: current lap %d, total laps %d\n",
			event.Competitor, comp.CurrentLap, s.Config.Laps)
		if comp.CurrentLap >= s.Config.Laps {
			comp.Status = "Finished"
			if startTime, ok := s.LastLapTime[event.Competitor]; ok {
				lapDuration := event.Time.Sub(startTime)
				comp.LapTimes = append(comp.LapTimes, lapDuration)
			}
			fmt.Printf("[%s] Competitor(%d) finished\n",
				event.Time.Format("15:04:05.000"), event.Competitor)
		}
	}
}
