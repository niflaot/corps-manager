package performance

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestAggregateTracksDeltasAndTuesdayCut(t *testing.T) {
	location, err := time.LoadLocation("America/Bogota")
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{config: Config{CutoffWeekday: time.Tuesday, Timezone: location, HistoryLimit: 10}}
	initialTime := time.Date(2026, time.August, 19, 12, 0, 0, 0, location)
	state := service.aggregate(State{BusinessID: 1995}, Snapshot{BusinessID: 1995, Name: "Market", Bank: 500,
		Employees: []EmployeeSnapshot{{CharacterID: 1, Name: "Alice", Earnings: 100, HistoricalDutyTime: 60, DutyTime: 15},
			{CharacterID: 2, Name: "Bob", Earnings: 50, HistoricalDutyTime: 30}}}, initialTime)
	if state.HistoricalGenerated != 150 || state.PeriodGenerated != 0 || state.HistoricalServiceMinutes != 105 ||
		state.PeriodServiceMinutes != 0 {
		t.Fatalf("initial totals = historical %d, period %d", state.HistoricalGenerated, state.PeriodGenerated)
	}
	state = service.aggregate(state, Snapshot{BusinessID: 1995, Name: "Market", Bank: 700,
		Employees: []EmployeeSnapshot{{CharacterID: 1, Name: "Alice", Earnings: 130, HistoricalDutyTime: 120}}}, initialTime.AddDate(0, 0, 2))
	if state.HistoricalGenerated != 180 || state.PeriodGenerated != 30 || state.HistoricalServiceMinutes != 150 ||
		state.PeriodServiceMinutes != 45 || state.Employees["2"].Active {
		t.Fatalf("second aggregate = %#v", state)
	}
	state = service.aggregate(state, Snapshot{BusinessID: 1995, Name: "Market", Bank: 900,
		Employees: []EmployeeSnapshot{{CharacterID: 1, Name: "Alice", Earnings: 150, HistoricalDutyTime: 180}}}, initialTime.AddDate(0, 0, 7))
	if state.HistoricalGenerated != 200 || state.PeriodGenerated != 20 || len(state.Periods) != 1 || state.Periods[0].Generated != 30 {
		t.Fatalf("cut aggregate = %#v", state)
	}
	if state.Periods[0].ServiceMinutes != 45 || state.PeriodServiceMinutes != 60 || state.HistoricalServiceMinutes != 210 {
		t.Fatalf("service cut aggregate = %#v", state)
	}
}

func TestRenderEmployeesGroupsRanksByAuthorityAndAlignsColumns(t *testing.T) {
	state := State{Ranks: []RankSnapshot{
		{ID: 142, Name: "Mechanic", Permissions: 3},
		{ID: 903, Name: "Chief Executive Officer", Permissions: 255},
	}, Employees: map[string]EmployeeState{
		"1": {EmployeeSnapshot: EmployeeSnapshot{CharacterID: 1, RankID: 142, Name: "Mechanic_Name"},
			HistoricalGenerated: 1200, Active: true},
		"2": {EmployeeSnapshot: EmployeeSnapshot{CharacterID: 2, RankID: 903, Name: "Executive_Name"},
			HistoricalGenerated: 250000, PeriodGenerated: 5000, Active: true},
	}}
	rendered := renderEmployees(state)
	executiveIndex := strings.Index(rendered, "Chief Executive Officer")
	mechanicIndex := strings.Index(rendered, "### Mechanic")
	if executiveIndex < 0 || mechanicIndex < 0 || executiveIndex >= mechanicIndex {
		t.Fatalf("rank order is incorrect:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Empleado") || !strings.Contains(rendered, "$5,000") ||
		!strings.Contains(rendered, "H sem") || !strings.Contains(rendered, "Executive N.") ||
		strings.Contains(rendered, "Executive_Name") ||
		!strings.Contains(rendered, "```text") {
		t.Fatalf("employee table is incomplete:\n%s", rendered)
	}
	for _, line := range strings.Split(employeeTableHeader()+employeeTableRow(state.Employees["2"]), "\n") {
		if utf8.RuneCountInString(line) > 50 {
			t.Fatalf("table line wraps at %d columns: %q", utf8.RuneCountInString(line), line)
		}
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

func TestAggregateCountsFirstObservationForEmployeeAddedAfterInitialization(t *testing.T) {
	service := &Service{config: Config{CutoffWeekday: time.Tuesday, Timezone: time.UTC, HistoryLimit: 10}}
	now := time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC)
	state := service.aggregate(State{BusinessID: 1}, Snapshot{BusinessID: 1,
		Employees: []EmployeeSnapshot{{CharacterID: 1, Earnings: 100, HistoricalDutyTime: 60}}}, now)
	state = service.aggregate(state, Snapshot{BusinessID: 1, Employees: []EmployeeSnapshot{
		{CharacterID: 1, Earnings: 100, HistoricalDutyTime: 60},
		{CharacterID: 2, Earnings: 500, HistoricalDutyTime: 30, DutyTime: 20},
	}}, now.Add(time.Hour))
	added := state.Employees["2"]
	if added.HistoricalGenerated != 500 || added.PeriodGenerated != 500 ||
		added.HistoricalServiceMinutes != 50 || added.PeriodServiceMinutes != 50 {
		t.Fatalf("new employee aggregate = %#v", added)
	}
}

func TestBackfillCurrentPeriodCorrectsOnlySelectedActiveEmployees(t *testing.T) {
	state := State{Employees: map[string]EmployeeState{
		"1": {HistoricalGenerated: 1000, PeriodGenerated: 200, HistoricalServiceMinutes: 60,
			PeriodServiceMinutes: 10, Active: true},
		"2": {HistoricalGenerated: 500, PeriodGenerated: 100, HistoricalServiceMinutes: 30,
			PeriodServiceMinutes: 5, Active: true},
	}}
	if err := state.backfillCurrentPeriod([]int64{1}); err != nil {
		t.Fatal(err)
	}
	if state.Employees["1"].PeriodGenerated != 1000 || state.Employees["1"].PeriodServiceMinutes != 60 ||
		state.Employees["2"].PeriodGenerated != 100 || state.PeriodGenerated != 1100 ||
		state.PeriodServiceMinutes != 65 {
		t.Fatalf("backfilled state = %#v", state)
	}
	if err := state.backfillCurrentPeriod([]int64{1, 1}); err != ErrInvalidBackfill {
		t.Fatalf("duplicate backfill error = %v", err)
	}
}

func TestRenderProducesValidManagedMessage(t *testing.T) {
	config := Config{BusinessID: 1995, ChannelID: "456", Timezone: time.UTC}
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
	if definition.Key != performanceMessageKey || definition.ChannelID != "456" {
		t.Fatalf("performance definition = %#v", definition)
	}
}
