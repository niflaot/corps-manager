package performance

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pixelados-net/discord-bot/internal/messages"
)

const dashboardAccent = 0x2ecc71

type textDisplay struct {
	Type    int    `json:"type"`
	Content string `json:"content"`
}

type separator struct {
	Type    int  `json:"type"`
	Divider bool `json:"divider"`
	Spacing int  `json:"spacing"`
}

type container struct {
	Type        int   `json:"type"`
	AccentColor int   `json:"accent_color"`
	Components  []any `json:"components"`
}

// Render creates the managed Components V2 dashboard definition.
func Render(state State, config Config, guildID string) (messages.Definition, error) {
	name := escapeMarkdown(state.Name)
	if name == "" {
		name = fmt.Sprintf("Negocio #%d", state.BusinessID)
	}
	components := []any{
		textDisplay{Type: 10, Content: fmt.Sprintf("# 📊 %s\n`Negocio #%d`", name, state.BusinessID)},
		separator{Type: 14, Divider: true, Spacing: 1},
		textDisplay{Type: 10, Content: fmt.Sprintf("## Resumen\n**Histórico registrado:** %s\n**Periodo actual:** %s\n**Banco:** %s",
			money(state.HistoricalGenerated), money(state.PeriodGenerated), money(state.Bank))},
		textDisplay{Type: 10, Content: renderEmployees(state)},
		separator{Type: 14, Divider: true, Spacing: 1},
		textDisplay{Type: 10, Content: renderPeriods(state, config)},
		textDisplay{Type: 10, Content: fmt.Sprintf("-# Actualizado: %s · Próximo corte: %s",
			discordTime(state.LastCollectedAt, "R"), discordTime(nextBoundary(state.PeriodStartedAt), "F"))},
	}
	encoded, err := json.Marshal(container{Type: 17, AccentColor: dashboardAccent, Components: components})
	if err != nil {
		return messages.Definition{}, fmt.Errorf("encode performance dashboard: %w", err)
	}
	return messages.Definition{Key: config.MessageKey, GuildID: guildID, ChannelID: config.ChannelID,
		Payload: messages.Payload{Components: []messages.Component{encoded},
			AllowedMentions: messages.AllowedMentions{Parse: []string{}}}}, nil
}

func renderEmployees(state State) string {
	employees := make([]EmployeeState, 0, len(state.Employees))
	for _, employee := range state.Employees {
		if employee.Active {
			employees = append(employees, employee)
		}
	}
	sort.Slice(employees, func(i, j int) bool {
		if employees[i].PeriodGenerated == employees[j].PeriodGenerated {
			return employees[i].Name < employees[j].Name
		}
		return employees[i].PeriodGenerated > employees[j].PeriodGenerated
	})
	if len(employees) == 0 {
		return "## Empleados\nSin empleados en la respuesta actual."
	}
	lines := []string{"## Empleados", "`Semanal` · `Histórico`"}
	for _, employee := range employees {
		line := fmt.Sprintf("**%s** · %s · %s", escapeMarkdown(employee.Name),
			money(employee.PeriodGenerated), money(employee.HistoricalGenerated))
		if employee.RankName != "" {
			line += " · " + escapeMarkdown(employee.RankName)
		}
		lines = append(lines, line)
		if len(strings.Join(lines, "\n")) > 3000 {
			lines = append(lines[:len(lines)-1], "_Lista truncada; consulta la API para ver el detalle completo._")
			break
		}
	}
	return strings.Join(lines, "\n")
}

func renderPeriods(state State, config Config) string {
	start := state.PeriodStartedAt.In(config.Timezone).Format("02 Jan 2006")
	lines := []string{fmt.Sprintf("## Cortes semanales\nPeriodo activo desde **%s**", start)}
	limit := min(len(state.Periods), 4)
	for _, period := range state.Periods[:limit] {
		lines = append(lines, fmt.Sprintf("%s → %s: **%s**", period.StartedAt.In(config.Timezone).Format("02 Jan"),
			period.EndedAt.In(config.Timezone).Format("02 Jan"), money(period.Generated)))
	}
	if limit == 0 {
		lines = append(lines, "Aún no hay cortes cerrados.")
	}
	return strings.Join(lines, "\n")
}

func money(value int64) string { return fmt.Sprintf("$%s", grouped(value)) }

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
