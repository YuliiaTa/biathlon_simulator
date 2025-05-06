package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Config struct {
	Laps        int    `json:"laps"`
	LapLen      int    `json:"lapLen"`
	PenaltyLen  int    `json:"penaltyLen"`
	FiringLines int    `json:"firingLines"`
	Start       string `json:"start"`
	StartDelta  string `json:"startDelta"`
}

type Event struct {
	Time       time.Time
	Type       int
	Competitor int
	Params     string
}

type Competitor struct {
	ID           int
	StartTime    time.Time
	ActualStart  time.Time
	LapTimes     []time.Duration
	PenaltyStart time.Time
	PenaltyTime  time.Duration
	Hits         int
	Shots        int
	Status       string
	Comment      string
	CurrentLap   int
	OnPenalty    bool
	OnRange      bool
}

func loadConfig(filename string) (*Config, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(file, &config); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	if config.Laps <= 0 {
		return nil, fmt.Errorf("invalid laps count: %d", config.Laps)
	}

	return &config, nil
}

func main() {
	time.Local = time.UTC

	config, err := loadConfig("config.json")
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	events, err := parseEvents("events.txt")
	if err != nil {
		fmt.Println("Error parsing events:", err)
		return
	}

	if len(events) == 0 {
		fmt.Println("No events to process")
		return
	}

	sim := NewSimulation(config)
	for _, event := range events {
		sim.ProcessEvent(event)
	}

	sim.GenerateReport()
}
