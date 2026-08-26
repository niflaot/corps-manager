package performance

import (
	"encoding/json"
	"fmt"
	"math/bits"
	"sort"
	"strings"
	"time"

	"github.com/niflaot/corps-manager/internal/messages"
)

const (
	dashboardAccent       = 0x2ecc71
	employeeColumnWidth   = 20
	moneyColumnWidth      = 11
	serviceColumnWidth    = 10
	employeesContentLimit = 3000
)

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
		textDisplay{Type: 10, Content: fmt.Sprintf("## Resumen\n**Dinero histórico:** %s\n**Dinero del periodo:** %s\n**Servicio histórico:** %s\n**Servicio del periodo:** %s\n**Banco:** %s",
			money(state.HistoricalGenerated), money(state.PeriodGenerated), serviceDuration(state.HistoricalServiceMinutes),
			serviceDuration(state.PeriodServiceMinutes), money(state.Bank))},
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
	ranksByID := make(map[int64]RankSnapshot, len(state.Ranks))
	for _, rank := range state.Ranks {
		ranksByID[rank.ID] = rank
	}
	groupsByKey := make(map[string]*employeeRankGroup)
	for _, employee := range state.Employees {
		if !employee.Active {
			continue
		}
		rank, known := ranksByID[employee.RankID]
		if strings.TrimSpace(rank.Name) == "" {
			rank = RankSnapshot{ID: employee.RankID, Name: strings.TrimSpace(employee.RankName)}
		}
		if rank.Name == "" {
			rank.Name = "Sin rango"
		}
		key := fmt.Sprintf("id:%d", rank.ID)
		if rank.ID == 0 {
			key = "name:" + strings.ToLower(rank.Name)
		}
		group := groupsByKey[key]
		if group == nil {
			group = &employeeRankGroup{Rank: rank, Known: known}
			groupsByKey[key] = group
		}
		group.Employees = append(group.Employees, employee)
	}
	if len(groupsByKey) == 0 {
		return "Sin empleados en la respuesta actual."
	}
	groups := make([]employeeRankGroup, 0, len(groupsByKey))
	for _, group := range groupsByKey {
		sort.Slice(group.Employees, func(i, j int) bool {
			left, right := group.Employees[i], group.Employees[j]
			if left.PeriodGenerated != right.PeriodGenerated {
				return left.PeriodGenerated > right.PeriodGenerated
			}
			if left.HistoricalGenerated != right.HistoricalGenerated {
				return left.HistoricalGenerated > right.HistoricalGenerated
			}
			return strings.ToLower(left.Name) < strings.ToLower(right.Name)
		})
		groups = append(groups, *group)
	}
	sort.Slice(groups, func(i, j int) bool { return higherRank(groups[i], groups[j]) })

	var rendered strings.Builder
	truncated := false
groupsLoop:
	for _, group := range groups {
		prefix := fmt.Sprintf("### %s\n```text\n", escapeMarkdown(group.Rank.Name))
		header := employeeTableHeader()
		if rendered.Len()+len(prefix)+len(header)+len("```\n") > employeesContentLimit {
			truncated = true
			break
		}
		rendered.WriteString(prefix)
		rendered.WriteString(header)
		for _, employee := range group.Employees {
			row := employeeTableRow(employee)
			if rendered.Len()+len(row)+len("```\n")+64 > employeesContentLimit {
				truncated = true
				rendered.WriteString("```\n")
				break groupsLoop
			}
			rendered.WriteString(row)
		}
		rendered.WriteString("```\n")
	}
	if truncated {
		rendered.WriteString("_Lista truncada; consulta la API para ver el detalle completo._")
	}
	return strings.TrimSpace(rendered.String())
}

type employeeRankGroup struct {
	Rank      RankSnapshot
	Known     bool
	Employees []EmployeeState
}

func higherRank(left, right employeeRankGroup) bool {
	if left.Known != right.Known {
		return left.Known
	}
	leftPermissions := bits.OnesCount64(left.Rank.Permissions)
	rightPermissions := bits.OnesCount64(right.Rank.Permissions)
	if leftPermissions != rightPermissions {
		return leftPermissions > rightPermissions
	}
	if left.Rank.Permissions != right.Rank.Permissions {
		return left.Rank.Permissions > right.Rank.Permissions
	}
	if left.Rank.ID != right.Rank.ID {
		return left.Rank.ID > right.Rank.ID
	}
	return strings.ToLower(left.Rank.Name) < strings.ToLower(right.Rank.Name)
}
func employeeTableHeader() string {
	return fmt.Sprintf("%-*s %*s %*s %*s %*s\n%s %s %s %s %s\n", employeeColumnWidth, "Empleado",
		moneyColumnWidth, "$ semana", moneyColumnWidth, "$ total", serviceColumnWidth, "H semana",
		serviceColumnWidth, "H total", strings.Repeat("-", employeeColumnWidth),
		strings.Repeat("-", moneyColumnWidth), strings.Repeat("-", moneyColumnWidth),
		strings.Repeat("-", serviceColumnWidth), strings.Repeat("-", serviceColumnWidth))
}
func employeeTableRow(employee EmployeeState) string {
	name := tableCell(employee.Name, employeeColumnWidth)
	return fmt.Sprintf("%-*s %*s %*s %*s %*s\n", employeeColumnWidth, name, moneyColumnWidth,
		money(employee.PeriodGenerated), moneyColumnWidth, money(employee.HistoricalGenerated), serviceColumnWidth,
		serviceDuration(employee.PeriodServiceMinutes), serviceColumnWidth, serviceDuration(employee.HistoricalServiceMinutes))
}

func tableCell(value string, width int) string {
	value = strings.NewReplacer("\r", " ", "\n", " ", "\t", " ", "`", "'").Replace(strings.TrimSpace(value))
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	return string(runes[:width-1]) + "…"
}

func renderPeriods(state State, config Config) string {
	start := state.PeriodStartedAt.In(config.Timezone).Format("02 Jan 2006")
	lines := []string{fmt.Sprintf("## Cortes semanales\nPeriodo activo desde **%s**", start)}
	limit := min(len(state.Periods), 4)
	for _, period := range state.Periods[:limit] {
		lines = append(lines, fmt.Sprintf("%s → %s: **%s** · **%s de servicio**",
			period.StartedAt.In(config.Timezone).Format("02 Jan"), period.EndedAt.In(config.Timezone).Format("02 Jan"),
			money(period.Generated), serviceDuration(period.ServiceMinutes)))
	}
	if limit == 0 {
		lines = append(lines, "Aún no hay cortes cerrados.")
	}
	return strings.Join(lines, "\n")
}

func money(value int64) string { return fmt.Sprintf("$%s", grouped(value)) }

func serviceDuration(minutes int64) string {
	if minutes <= 0 {
		return "0m"
	}
	days := minutes / (24 * 60)
	hours := minutes % (24 * 60) / 60
	remainingMinutes := minutes % 60
	parts := make([]string, 0, 3)
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if remainingMinutes > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%dm", remainingMinutes))
	}
	return strings.Join(parts, " ")
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
