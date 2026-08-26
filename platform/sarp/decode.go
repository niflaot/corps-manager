package sarp

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pixelados-net/discord-bot/internal/performance"
)

type rawBusiness struct {
	ID        flexibleInt   `json:"id"`
	Name      string        `json:"name"`
	Bank      flexibleInt   `json:"bank"`
	Employees []rawEmployee `json:"employees"`
}

type rawEmployee struct {
	ID                 flexibleInt `json:"id"`
	CharacterID        flexibleInt `json:"character_id"`
	Name               string      `json:"name"`
	CharacterName      string      `json:"character_name"`
	RankName           string      `json:"rank_name"`
	Earnings           flexibleInt `json:"earnings"`
	DutyTime           flexibleInt `json:"duty_time"`
	HistoricalDutyTime flexibleInt `json:"historical_duty_time"`
	LastActivity       string      `json:"last_activity"`
	LastLogin          string      `json:"last_login"`
}

func decodeSnapshot(body []byte) (performance.Snapshot, error) {
	data := json.RawMessage(body)
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(data, &envelope); err != nil {
		return performance.Snapshot{}, fmt.Errorf("decode SARP response: %w", err)
	}
	if rawError, ok := envelope["error"]; ok {
		var message string
		_ = json.Unmarshal(rawError, &message)
		if message != "" {
			return performance.Snapshot{}, fmt.Errorf("SARP endpoint error: %s", message)
		}
	}
	if payload, ok := envelope["payload"]; ok && string(payload) != "null" {
		data = payload
	}
	var nested map[string]json.RawMessage
	if json.Unmarshal(data, &nested) == nil {
		if business, ok := nested["business"]; ok && string(business) != "null" {
			data = business
		}
	}
	var raw rawBusiness
	if err := json.Unmarshal(data, &raw); err != nil {
		return performance.Snapshot{}, fmt.Errorf("decode SARP business: %w", err)
	}
	if raw.ID <= 0 {
		return performance.Snapshot{}, fmt.Errorf("SARP business response is missing id")
	}
	if raw.Employees == nil {
		return performance.Snapshot{}, fmt.Errorf("SARP business response does not expose employees")
	}
	snapshot := performance.Snapshot{BusinessID: int64(raw.ID), Name: strings.TrimSpace(raw.Name), Bank: int64(raw.Bank),
		Employees: make([]performance.EmployeeSnapshot, 0, len(raw.Employees))}
	for _, employee := range raw.Employees {
		characterID := employee.CharacterID
		if characterID == 0 {
			characterID = employee.ID
		}
		if characterID <= 0 {
			continue
		}
		name := strings.TrimSpace(employee.Name)
		if name == "" {
			name = strings.TrimSpace(employee.CharacterName)
		}
		snapshot.Employees = append(snapshot.Employees, performance.EmployeeSnapshot{
			CharacterID: int64(characterID), Name: name, RankName: strings.TrimSpace(employee.RankName),
			Earnings: int64(employee.Earnings), DutyTime: int64(employee.DutyTime),
			HistoricalDutyTime: int64(employee.HistoricalDutyTime), LastActivity: employee.LastActivity,
			LastLogin: employee.LastLogin,
		})
	}
	return snapshot, nil
}

func responseMessage(body []byte) string {
	var value struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &value) == nil {
		if value.Error != "" {
			return value.Error
		}
		if value.Message != "" {
			return value.Message
		}
	}
	message := strings.TrimSpace(string(body))
	if len(message) > 200 {
		message = message[:200]
	}
	return message
}
