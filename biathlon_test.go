package main

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	configContent := `{
		"laps": 2,
		"lapLen": 3651,
		"penaltyLen": 50,
		"firingLines": 1,
		"start": "09:30:00",
		"startDelta": "00:00:30"
	}`
	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(configContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

	config, err := loadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.Laps != 2 {
		t.Errorf("Expected Laps=2, got %d", config.Laps)
	}
	if config.LapLen != 3651 {
		t.Errorf("Expected LapLen=3651, got %d", config.LapLen)
	}
	if config.PenaltyLen != 50 {
		t.Errorf("Expected PenaltyLen=50, got %d", config.PenaltyLen)
	}
	if config.FiringLines != 1 {
		t.Errorf("Expected FiringLines=1, got %d", config.FiringLines)
	}
	if config.Start != "09:30:00" {
		t.Errorf("Expected Start=09:30:00, got %s", config.Start)
	}
	if config.StartDelta != "00:00:30" {
		t.Errorf("Expected StartDelta=00:00:30, got %s", config.StartDelta)
	}

	_, err = loadConfig("nonexistent.json")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestParseEvents(t *testing.T) {
	eventsContent := `[09:05:59.867] 1 1
[09:15:00.841] 2 1 09:30:00.000
[09:29:45.734] 3 1
[09:30:01.005] 4 1
[09:49:31.659] 5 1 1
[09:49:33.123] 6 1 1
[09:49:34.650] 6 1 2
[09:49:35.937] 6 1 4
[09:49:37.364] 6 1 5
[09:49:38.339] 7 1
[09:49:55.915] 8 1
[09:51:48.391] 9 1
[09:59:03.872] 10 1
[09:59:03.872] 11 1 Lost in the forest`
	tmpFile, err := os.CreateTemp("", "events-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(eventsContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

	events, err := parseEvents(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to parse events: %v", err)
	}

	if len(events) != 14 {
		t.Fatalf("Expected 14 events, got %d", len(events))
	}

	firstEvent := events[0]
	expectedTime, _ := time.Parse("15:04:05.000", "09:05:59.867")
	if !firstEvent.Time.Equal(expectedTime) {
		t.Errorf("First event time mismatch, expected %v, got %v", expectedTime, firstEvent.Time)
	}
	if firstEvent.Type != 1 {
		t.Errorf("First event type mismatch, expected 1, got %d", firstEvent.Type)
	}
	if firstEvent.Competitor != 1 {
		t.Errorf("First event competitor ID mismatch, expected 1, got %d", firstEvent.Competitor)
	}

	lastEvent := events[len(events)-1]
	if lastEvent.Type != 11 {
		t.Errorf("Last event type mismatch, expected 11, got %d", lastEvent.Type)
	}
	if lastEvent.Params != "Lost in the forest" {
		t.Errorf("Last event params mismatch, expected 'Lost in the forest', got '%s'", lastEvent.Params)
	}

	invalidContent := "[invalid time] 1 1"
	tmpFile2, err := os.CreateTemp("", "events-invalid-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile2.Name())

	if _, err := tmpFile2.Write([]byte(invalidContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpFile2.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = parseEvents(tmpFile2.Name())
	if err == nil {
		t.Error("Expected error for invalid event format")
	}
}

func TestSimulationProcessEvent(t *testing.T) {
	config := &Config{
		Laps:        2,
		LapLen:      3651,
		PenaltyLen:  50,
		FiringLines: 1,
		Start:       "09:30:00",
		StartDelta:  "00:00:30",
	}

	sim := NewSimulation(config)

	regTime, _ := time.Parse("15:04:05.000", "09:05:59.867")
	regEvent := Event{Time: regTime, Type: 1, Competitor: 1}
	sim.ProcessEvent(regEvent)

	if _, exists := sim.Competitors[1]; !exists {
		t.Error("Competitor not registered")
	}

	startTime, _ := time.Parse("15:04:05.000", "09:15:00.841")
	setStartEvent := Event{
		Time:       startTime,
		Type:       2,
		Competitor: 1,
		Params:     "09:30:00.000",
	}
	sim.ProcessEvent(setStartEvent)

	if sim.Competitors[1].StartTime.Format("15:04:05.000") != "09:30:00.000" {
		t.Error("Incorrect start time set")
	}

	startLineTime, _ := time.Parse("15:04:05.000", "09:29:45.734")
	startLineEvent := Event{Time: startLineTime, Type: 3, Competitor: 1}
	sim.ProcessEvent(startLineEvent)

	actualStartTime, _ := time.Parse("15:04:05.000", "09:30:01.005")
	startEvent := Event{Time: actualStartTime, Type: 4, Competitor: 1}
	sim.ProcessEvent(startEvent)

	if sim.Competitors[1].ActualStart.Format("15:04:05.000") != "09:30:01.005" {
		t.Error("Incorrect actual start time")
	}
	if sim.Competitors[1].Status != "Running" {
		t.Error("Competitor status not set to 'Running'")
	}

	firingRangeTime, _ := time.Parse("15:04:05.000", "09:49:31.659")
	firingRangeEvent := Event{Time: firingRangeTime, Type: 5, Competitor: 1, Params: "1"}
	sim.ProcessEvent(firingRangeEvent)

	if !sim.Competitors[1].OnRange {
		t.Error("Competitor not marked on firing range")
	}

	hitTime1, _ := time.Parse("15:04:05.000", "09:49:33.123")
	hitEvent1 := Event{Time: hitTime1, Type: 6, Competitor: 1, Params: "1"}
	sim.ProcessEvent(hitEvent1)

	hitTime2, _ := time.Parse("15:04:05.000", "09:49:34.650")
	hitEvent2 := Event{Time: hitTime2, Type: 6, Competitor: 1, Params: "2"}
	sim.ProcessEvent(hitEvent2)

	if sim.Competitors[1].Hits != 2 {
		t.Errorf("Expected 2 hits, got %d", sim.Competitors[1].Hits)
	}

	leaveRangeTime, _ := time.Parse("15:04:05.000", "09:49:38.339")
	leaveRangeEvent := Event{Time: leaveRangeTime, Type: 7, Competitor: 1}
	sim.ProcessEvent(leaveRangeEvent)

	if sim.Competitors[1].OnRange {
		t.Error("Competitor still marked on firing range")
	}
	if sim.Competitors[1].Shots != 5 {
		t.Errorf("Expected 5 shots, got %d", sim.Competitors[1].Shots)
	}

	penaltyEnterTime, _ := time.Parse("15:04:05.000", "09:49:55.915")
	penaltyEnterEvent := Event{Time: penaltyEnterTime, Type: 8, Competitor: 1}
	sim.ProcessEvent(penaltyEnterEvent)

	if !sim.Competitors[1].OnPenalty {
		t.Error("Competitor not marked on penalty lap")
	}

	penaltyLeaveTime, _ := time.Parse("15:04:05.000", "09:51:48.391")
	penaltyLeaveEvent := Event{Time: penaltyLeaveTime, Type: 9, Competitor: 1}
	sim.ProcessEvent(penaltyLeaveEvent)

	if sim.Competitors[1].OnPenalty {
		t.Error("Competitor still marked on penalty lap")
	}
	if sim.Competitors[1].PenaltyTime == 0 {
		t.Error("Penalty time not calculated")
	}

	lapTime, _ := time.Parse("15:04:05.000", "09:59:03.872")
	lapEvent := Event{Time: lapTime, Type: 10, Competitor: 1}
	sim.ProcessEvent(lapEvent)

	if len(sim.Competitors[1].LapTimes) != 1 {
		t.Errorf("Expected 1 lap time, got %d", len(sim.Competitors[1].LapTimes))
	}
	if sim.Competitors[1].CurrentLap != 2 {
		t.Errorf("Expected current lap=2, got %d", sim.Competitors[1].CurrentLap)
	}

	notFinishedTime, _ := time.Parse("15:04:05.000", "09:59:03.872")
	notFinishedEvent := Event{Time: notFinishedTime, Type: 11, Competitor: 1, Params: "Lost in the forest"}
	sim.ProcessEvent(notFinishedEvent)

	if sim.Competitors[1].Status != "NotFinished" {
		t.Error("Status not set to 'NotFinished'")
	}
	if sim.Competitors[1].Comment != "Lost in the forest" {
		t.Error("Incorrect comment")
	}
}

func TestSimulationDisqualification(t *testing.T) {
	config := &Config{
		Laps:        2,
		LapLen:      3651,
		PenaltyLen:  50,
		FiringLines: 1,
		Start:       "09:30:00",
		StartDelta:  "00:00:30",
	}

	sim := NewSimulation(config)

	regTime, _ := time.Parse("15:04:05.000", "09:05:59.867")
	regEvent := Event{Time: regTime, Type: 1, Competitor: 1}
	sim.ProcessEvent(regEvent)

	startTime, _ := time.Parse("15:04:05.000", "09:15:00.841")
	setStartEvent := Event{
		Time:       startTime,
		Type:       2,
		Competitor: 1,
		Params:     "09:30:00.000",
	}
	sim.ProcessEvent(setStartEvent)

	lateTime, _ := time.Parse("15:04:05.000", "09:30:45.734")
	lateEvent := Event{Time: lateTime, Type: 3, Competitor: 1}
	sim.ProcessEvent(lateEvent)

	if sim.Competitors[1].Status != "NotStarted" {
		t.Error("Expected disqualification for late arrival at start")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{time.Hour + 30*time.Minute + 15*time.Second + 123*time.Millisecond, "01:30:15.123"},
		{45 * time.Second, "00:00:45.000"},
		{2*time.Hour + 59*time.Millisecond, "02:00:00.059"},
	}

	for _, test := range tests {
		result := formatDuration(test.duration)
		if result != test.expected {
			t.Errorf("For duration %v expected %s, got %s", test.duration, test.expected, result)
		}
	}
}

func TestTotalTimeCalculation(t *testing.T) {
	startTime := time.Date(2023, 1, 1, 9, 30, 0, 0, time.UTC)

	comp := &Competitor{
		Status:      "Finished",
		ActualStart: startTime,
		LapTimes:    []time.Duration{15 * time.Minute, 15 * time.Minute},
		PenaltyTime: 5 * time.Minute,
	}

	total := comp.TotalTime()
	expected := 30*time.Minute + 5*time.Minute
	if total != expected {
		t.Errorf("Expected total time %v, got %v", expected, total)
	}

	compNotFinished := &Competitor{
		Status: "NotFinished",
	}
	if compNotFinished.TotalTime() != 0 {
		t.Error("Expected total time 0 for NotFinished competitor")
	}
}
