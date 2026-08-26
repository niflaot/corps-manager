package performance

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/pixelados-net/discord-bot/internal/messages"
	"github.com/pixelados-net/discord-bot/platform/clock"
	"go.uber.org/zap"
)

const saveAttempts = 3

// Service collects, aggregates, and publishes business performance.
type Service struct {
	config     Config
	source     Source
	repository Repository
	messages   *messages.Service
	clock      clock.Clock
	log        *zap.Logger
	guildID    string
}

// NewService creates the business performance application service.
func NewService(config Config, source Source, repository Repository, messageService *messages.Service,
	serviceClock clock.Clock, log *zap.Logger, guildID string) *Service {
	return &Service{config: config, source: source, repository: repository, messages: messageService,
		clock: serviceClock, log: log, guildID: guildID}
}

// Enabled reports whether automatic collection is configured.
func (service *Service) Enabled() bool { return service.config.Enabled }

// Get returns the current persisted aggregate.
func (service *Service) Get(ctx context.Context) (State, error) {
	if !service.config.Enabled {
		return State{}, ErrDisabled
	}
	return service.repository.Get(ctx, service.config.BusinessID)
}

// Refresh collects one snapshot, persists its deltas, and updates the managed dashboard.
func (service *Service) Refresh(ctx context.Context) (State, error) {
	if !service.config.Enabled {
		return State{}, ErrDisabled
	}
	snapshot, err := service.source.Fetch(ctx, service.config.BusinessID)
	if err != nil {
		return State{}, fmt.Errorf("fetch business performance: %w", err)
	}
	now := service.clock.Now().In(service.config.Timezone)
	for attempt := 0; attempt < saveAttempts; attempt++ {
		current, getErr := service.repository.Get(ctx, service.config.BusinessID)
		if errors.Is(getErr, ErrNotFound) {
			current = State{BusinessID: service.config.BusinessID, Employees: map[string]EmployeeState{}}
		} else if getErr != nil {
			return State{}, fmt.Errorf("read business performance: %w", getErr)
		}
		next := service.aggregate(current, snapshot, now)
		saved, saveErr := service.repository.Save(ctx, next, current.Revision)
		if errors.Is(saveErr, ErrConflict) {
			continue
		}
		if saveErr != nil {
			return State{}, fmt.Errorf("save business performance: %w", saveErr)
		}
		if err := service.publish(ctx, saved); err != nil {
			return saved, fmt.Errorf("publish business performance: %w", err)
		}
		service.log.Debug("business performance refreshed", zap.Int64("business_id", saved.BusinessID),
			zap.Int64("period_generated", saved.PeriodGenerated), zap.Int64("historical_generated", saved.HistoricalGenerated))
		return saved, nil
	}
	return State{}, ErrConflict
}

func (service *Service) aggregate(state State, snapshot Snapshot, now time.Time) State {
	if state.Employees == nil {
		state.Employees = map[string]EmployeeState{}
	}
	currentBoundary := periodBoundary(now, service.config.CutoffWeekday)
	if state.PeriodStartedAt.IsZero() {
		state.PeriodStartedAt = currentBoundary
	} else {
		for state.PeriodStartedAt.Before(currentBoundary) {
			boundary := state.PeriodStartedAt.AddDate(0, 0, 7)
			if boundary.After(currentBoundary) {
				boundary = currentBoundary
			}
			state.archive(boundary, service.config.HistoryLimit)
		}
	}
	state.Name, state.Bank = snapshot.Name, snapshot.Bank
	for key, employee := range state.Employees {
		employee.Active = false
		state.Employees[key] = employee
	}
	for _, observed := range snapshot.Employees {
		key := strconv.FormatInt(observed.CharacterID, 10)
		employee, exists := state.Employees[key]
		if !exists {
			employee.HistoricalGenerated = max(observed.Earnings, 0)
		} else if observed.Earnings > employee.Baseline {
			delta := observed.Earnings - employee.Baseline
			employee.HistoricalGenerated += delta
			employee.PeriodGenerated += delta
		}
		employee.EmployeeSnapshot = observed
		employee.Baseline = observed.Earnings
		employee.Active = true
		employee.LastSeenAt = now
		state.Employees[key] = employee
	}
	state.HistoricalGenerated, state.PeriodGenerated = totals(state.Employees)
	state.Initialized = true
	state.LastCollectedAt = now
	return state
}

func (state *State) archive(boundary time.Time, limit int) {
	employees := make(map[string]int64)
	for key, employee := range state.Employees {
		if employee.PeriodGenerated > 0 {
			employees[key] = employee.PeriodGenerated
		}
		employee.PeriodGenerated = 0
		state.Employees[key] = employee
	}
	completed := Period{StartedAt: state.PeriodStartedAt, EndedAt: boundary,
		Generated: state.PeriodGenerated, Employees: employees}
	state.Periods = append([]Period{completed}, state.Periods...)
	if len(state.Periods) > limit {
		state.Periods = state.Periods[:limit]
	}
	state.PeriodGenerated = 0
	state.PeriodStartedAt = boundary
}

func totals(employees map[string]EmployeeState) (int64, int64) {
	var historical, period int64
	for _, employee := range employees {
		historical += employee.HistoricalGenerated
		period += employee.PeriodGenerated
	}
	return historical, period
}

func periodBoundary(now time.Time, weekday time.Weekday) time.Time {
	local := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	days := (int(local.Weekday()) - int(weekday) + 7) % 7
	return local.AddDate(0, 0, -days)
}

func (service *Service) publish(ctx context.Context, state State) error {
	definition, err := Render(state, service.config, service.guildID)
	if err != nil {
		return err
	}
	record, err := service.messages.Get(ctx, service.config.MessageKey)
	if errors.Is(err, messages.ErrNotFound) {
		_, err = service.messages.Create(ctx, definition, fmt.Sprintf("performance-create-%d", state.BusinessID))
		return err
	}
	if err != nil {
		return err
	}
	_, err = service.messages.Replace(ctx, record.Key, record.Revision, definition,
		fmt.Sprintf("performance-replace-%d-%d-%d", state.BusinessID, state.Revision, record.Revision))
	return err
}
