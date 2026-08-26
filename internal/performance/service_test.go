package performance

import (
	"testing"
	"time"
)

func TestAggregateTracksDeltasAndTuesdayCut(t *testing.T) {
	location, err := time.LoadLocation("America/Bogota")
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{config: Config{CutoffWeekday: time.Tuesday, Timezone: location, HistoryLimit: 10}}
	initialTime := time.Date(2026, time.August, 19, 12, 0, 0, 0, location)
	state := service.aggregate(State{BusinessID: 1995}, Snapshot{BusinessID: 1995, Name: "Market", Bank: 500,
		Employees: []EmployeeSnapshot{{CharacterID: 1, Name: "Alice", Earnings: 100},
			{CharacterID: 2, Name: "Bob", Earnings: 50}}}, initialTime)
	if state.HistoricalGenerated != 150 || state.PeriodGenerated != 0 {
		t.Fatalf("initial totals = historical %d, period %d", state.HistoricalGenerated, state.PeriodGenerated)
	}
	state = service.aggregate(state, Snapshot{BusinessID: 1995, Name: "Market", Bank: 700,
		Employees: []EmployeeSnapshot{{CharacterID: 1, Name: "Alice", Earnings: 130}}}, initialTime.AddDate(0, 0, 2))
	if state.HistoricalGenerated != 180 || state.PeriodGenerated != 30 || state.Employees["2"].Active {
		t.Fatalf("second aggregate = %#v", state)
	}
	state = service.aggregate(state, Snapshot{BusinessID: 1995, Name: "Market", Bank: 900,
		Employees: []EmployeeSnapshot{{CharacterID: 1, Name: "Alice", Earnings: 150}}}, initialTime.AddDate(0, 0, 7))
	if state.HistoricalGenerated != 200 || state.PeriodGenerated != 20 || len(state.Periods) != 1 || state.Periods[0].Generated != 30 {
		t.Fatalf("cut aggregate = %#v", state)
	}
}

func TestAggregateDoesNotSubtractResetCounter(t *testing.T) {
	location := time.UTC
	service := &Service{config: Config{CutoffWeekday: time.Tuesday, Timezone: location, HistoryLimit: 10}}
	now := time.Date(2026, time.August, 19, 0, 0, 0, 0, location)
	state := service.aggregate(State{BusinessID: 1}, Snapshot{BusinessID: 1,
		Employees: []EmployeeSnapshot{{CharacterID: 10, Earnings: 500}}}, now)
	state = service.aggregate(state, Snapshot{BusinessID: 1,
		Employees: []EmployeeSnapshot{{CharacterID: 10, Earnings: 5}}}, now.Add(time.Hour))
	if state.HistoricalGenerated != 500 || state.PeriodGenerated != 0 || state.Employees["10"].Baseline != 5 {
		t.Fatalf("reset aggregate = %#v", state)
	}
}

func TestRenderProducesValidManagedMessage(t *testing.T) {
	config := Config{BusinessID: 1995, ChannelID: "456", MessageKey: "business-performance", Timezone: time.UTC}
	state := State{BusinessID: 1995, Name: "Warehouse", Bank: 1000, HistoricalGenerated: 300,
		PeriodGenerated: 50, PeriodStartedAt: time.Now().AddDate(0, 0, -1), LastCollectedAt: time.Now(),
		Employees: map[string]EmployeeState{"1": {EmployeeSnapshot: EmployeeSnapshot{CharacterID: 1, Name: "Alice"},
			HistoricalGenerated: 300, PeriodGenerated: 50, Active: true}}}
	definition, err := Render(state, config, "123")
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if err := definition.Validate(); err != nil {
		t.Fatalf("rendered definition is invalid: %v", err)
	}
}
