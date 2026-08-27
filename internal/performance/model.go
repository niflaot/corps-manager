// Package performance tracks business earnings over durable weekly periods.
package performance

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// EmployeeSnapshot is one employee counter returned by SARP.
type EmployeeSnapshot struct {
	// CharacterID identifies the employee character.
	CharacterID int64 `json:"characterId"`
	// RankID identifies the employee's current business rank.
	RankID int64 `json:"rankId,omitempty"`
	// Name is the current character display name.
	Name string `json:"name"`
	// RankName is the current business rank.
	RankName string `json:"rankName,omitempty"`
	// Earnings is the current upstream cumulative counter.
	Earnings int64 `json:"earnings"`
	// DutyTime is the current upstream duty-time value.
	DutyTime int64 `json:"dutyTime,omitempty"`
	// HistoricalDutyTime is the upstream historical duty-time value.
	HistoricalDutyTime int64 `json:"historicalDutyTime,omitempty"`
	// LastActivity is the last reported activity value.
	LastActivity string `json:"lastActivity,omitempty"`
	// LastLogin is the last reported login value.
	LastLogin string `json:"lastLogin,omitempty"`
}

// RankSnapshot describes one business rank returned by SARP.
type RankSnapshot struct {
	// ID identifies the rank.
	ID int64 `json:"id"`
	// Name is the rank display name.
	Name string `json:"name"`
	// Permissions is the upstream permission bitmask used for hierarchy ordering.
	Permissions uint64 `json:"permissions"`
	// Paycheck is the configured paycheck for the rank.
	Paycheck int64 `json:"paycheck"`
}

// Snapshot is the current business projection returned by SARP.
type Snapshot struct {
	// BusinessID identifies the monitored business.
	BusinessID int64 `json:"businessId"`
	// Name is the business display name.
	Name string `json:"name"`
	// Bank is the current business bank balance.
	Bank int64 `json:"bank"`
	// Employees contains current employee counters.
	Employees []EmployeeSnapshot `json:"employees"`
	// Ranks contains the current business rank definitions.
	Ranks []RankSnapshot `json:"ranks"`
}

// EmployeeState stores durable earnings for one current or former employee.
type EmployeeState struct {
	EmployeeSnapshot
	// Baseline is the most recently observed upstream earnings counter.
	Baseline int64 `json:"baseline"`
	// ServiceBaseline is the most recently observed total service-minute counter.
	ServiceBaseline int64 `json:"serviceBaseline"`
	// ServiceInitialized reports whether the service baseline has been captured.
	ServiceInitialized bool `json:"serviceInitialized"`
	// HistoricalGenerated is the total observed for this employee.
	HistoricalGenerated int64 `json:"historicalGenerated"`
	// PeriodGenerated is the total observed in the active period.
	PeriodGenerated int64 `json:"periodGenerated"`
	// HistoricalServiceMinutes is the total observed service time for this employee.
	HistoricalServiceMinutes int64 `json:"historicalServiceMinutes"`
	// PeriodServiceMinutes is the observed service time in the active period.
	PeriodServiceMinutes int64 `json:"periodServiceMinutes"`
	// Active reports whether the employee appeared in the latest snapshot.
	Active bool `json:"active"`
	// LastSeenAt is the last snapshot containing this employee.
	LastSeenAt time.Time `json:"lastSeenAt"`
}

// Period contains one completed weekly cut.
type Period struct {
	// StartedAt is the inclusive period boundary.
	StartedAt time.Time `json:"startedAt"`
	// EndedAt is the exclusive period boundary.
	EndedAt time.Time `json:"endedAt"`
	// Generated is the business total accumulated in the period.
	Generated int64 `json:"generated"`
	// ServiceMinutes is the business service time accumulated in the period.
	ServiceMinutes int64 `json:"serviceMinutes"`
	// Employees maps character IDs to their period totals.
	Employees map[string]int64 `json:"employees"`
	// EmployeeServiceMinutes maps character IDs to their period service time.
	EmployeeServiceMinutes map[string]int64 `json:"employeeServiceMinutes"`
}

// CurrentPeriodBackfill identifies employees whose first observed counters belong to the active period.
type CurrentPeriodBackfill struct {
	// PeriodStartedAt guards against correcting a different weekly period.
	PeriodStartedAt time.Time `json:"periodStartedAt"`
	// CharacterIDs contains the newly joined employees to correct.
	CharacterIDs []int64 `json:"characterIds"`
}

// State is the persisted business performance aggregate.
type State struct {
	// BusinessID identifies the monitored business.
	BusinessID int64 `json:"businessId"`
	// Name is the latest business display name.
	Name string `json:"name"`
	// Bank is the latest business bank balance.
	Bank int64 `json:"bank"`
	// HistoricalGenerated is the tracked all-time total.
	HistoricalGenerated int64 `json:"historicalGenerated"`
	// PeriodGenerated is the active weekly total.
	PeriodGenerated int64 `json:"periodGenerated"`
	// HistoricalServiceMinutes is the tracked all-time service total.
	HistoricalServiceMinutes int64 `json:"historicalServiceMinutes"`
	// PeriodServiceMinutes is the service total in the active weekly period.
	PeriodServiceMinutes int64 `json:"periodServiceMinutes"`
	// PeriodStartedAt is the active weekly boundary.
	PeriodStartedAt time.Time `json:"periodStartedAt"`
	// Employees contains current and former employee aggregates.
	Employees map[string]EmployeeState `json:"employees"`
	// Ranks contains the latest business rank definitions.
	Ranks []RankSnapshot `json:"ranks"`
	// Periods contains completed cuts from newest to oldest.
	Periods []Period `json:"periods"`
	// Initialized reports whether a baseline has been captured.
	Initialized bool `json:"initialized"`
	// LastCollectedAt is the latest successful upstream observation.
	LastCollectedAt time.Time `json:"lastCollectedAt"`
	// Revision is the optimistic persistence revision.
	Revision uint64 `json:"revision"`
	// UpdatedAt is the persistence update timestamp.
	UpdatedAt time.Time `json:"updatedAt"`
}

func (state *State) backfillCurrentPeriod(characterIDs []int64) error {
	seen := make(map[int64]bool, len(characterIDs))
	for _, characterID := range characterIDs {
		if characterID <= 0 || seen[characterID] {
			return ErrInvalidBackfill
		}
		seen[characterID] = true
		key := strconv.FormatInt(characterID, 10)
		employee, exists := state.Employees[key]
		if !exists || !employee.Active {
			return ErrInvalidBackfill
		}
		employee.PeriodGenerated = employee.HistoricalGenerated
		employee.PeriodServiceMinutes = employee.HistoricalServiceMinutes
		state.Employees[key] = employee
	}
	state.HistoricalGenerated, state.PeriodGenerated, state.HistoricalServiceMinutes,
		state.PeriodServiceMinutes = totals(state.Employees)
	return nil
}

func employeeDisplayName(value string) string {
	parts := strings.SplitN(strings.TrimSpace(value), "_", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return strings.ReplaceAll(strings.TrimSpace(value), "_", " ")
	}
	surname := []rune(parts[1])
	return parts[0] + " " + string(unicode.ToUpper(surname[0])) + "."
}

func money(value int64) string { return fmt.Sprintf("$%s", grouped(value)) }

func serviceDuration(minutes int64) string {
	if minutes <= 0 {
		return "0m"
	}
	days, hours, remaining := minutes/(24*60), minutes%(24*60)/60, minutes%60
	parts := make([]string, 0, 3)
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if remaining > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%dm", remaining))
	}
	return strings.Join(parts, " ")
}

func serviceTableDuration(minutes int64) string {
	if minutes >= 24*60 {
		days, hours := minutes/(24*60), minutes%(24*60)/60
		if hours == 0 {
			return fmt.Sprintf("%dd", days)
		}
		return fmt.Sprintf("%dd%dh", days, hours)
	}
	if minutes >= 60 {
		hours, remaining := minutes/60, minutes%60
		if remaining == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dh%dm", hours, remaining)
	}
	return fmt.Sprintf("%dm", max(minutes, 0))
}

func grouped(value int64) string {
	negative := value < 0
	if negative {
		value = -value
	}
	digits := fmt.Sprintf("%d", value)
	for index := len(digits) - 3; index > 0; index -= 3 {
		digits = digits[:index] + "," + digits[index:]
	}
	if negative {
		return "-" + digits
	}
	return digits
}

func discordTime(value time.Time, style string) string {
	return fmt.Sprintf("<t:%d:%s>", value.Unix(), style)
}

func nextBoundary(start time.Time) time.Time { return start.AddDate(0, 0, 7) }

func escapeMarkdown(value string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "*", "\\*", "_", "\\_", "`", "\\`", "~", "\\~")
	return strings.TrimSpace(replacer.Replace(value))
}
