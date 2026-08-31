package customerweb

import (
	"errors"
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/niflaot/corps-manager/internal/customers"
)

const maximumFilterDays = 3650

// Page renders the filterable customer directory.
type Page struct {
	service  *customers.Service
	template *template.Template
}

type pageData struct {
	Name  string
	Days  int
	Sort  customers.Sort
	Items []customers.Customer
}

// New creates the frequent-customer web page.
func New(service *customers.Service) *Page {
	functions := template.FuncMap{
		"money":       formatMoney,
		"displayName": displayName,
		"date": func(value time.Time) string {
			if value.IsZero() {
				return "—"
			}
			return value.In(time.FixedZone("America/Bogota", -5*60*60)).Format("02 Jan 2006 · 15:04")
		},
	}
	return &Page{service: service, template: template.Must(template.New("customers").Funcs(functions).Parse(pageHTML))}
}

// Handle renders filtered customers as a public read-only HTML page.
func (page *Page) Handle(ctx *fiber.Ctx) error {
	days, err := parseDays(ctx.Query("days"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "days must be between 0 and 3650")
	}
	query := customers.Query{Name: ctx.Query("name"), Days: days, Sort: customers.Sort(ctx.Query("sort"))}
	items, err := page.service.Search(ctx.UserContext(), query)
	if err != nil {
		return customerError(err)
	}
	if query.Sort == "" {
		query.Sort = customers.SortSpend
	}
	ctx.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	ctx.Set(fiber.HeaderCacheControl, "no-store")
	ctx.Set("X-Robots-Tag", "noindex, nofollow")
	ctx.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; img-src https:; base-uri 'none'; form-action 'self'; frame-ancestors 'none'")
	return page.template.Execute(ctx.Response().BodyWriter(), pageData{Name: ctx.Query("name"), Days: days,
		Sort: query.Sort, Items: items})
}

func parseDays(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	days, err := strconv.Atoi(value)
	if err != nil || days < 0 || days > maximumFilterDays {
		return 0, customers.ErrInvalidQuery
	}
	return days, nil
}

func customerError(err error) error {
	switch {
	case errors.Is(err, customers.ErrInvalidName), errors.Is(err, customers.ErrInvalidQuery):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, customers.ErrDisabled):
		return fiber.NewError(fiber.StatusServiceUnavailable, err.Error())
	default:
		return fiber.NewError(fiber.StatusInternalServerError, "customer directory failed")
	}
}

func displayName(value string) string {
	words := strings.Split(value, "_")
	for index, word := range words {
		if word != "" {
			letters := []rune(word)
			words[index] = strings.ToUpper(string(letters[0])) + string(letters[1:])
		}
	}
	return strings.Join(words, " ")
}

func formatMoney(value int64) string {
	digits := strconv.FormatInt(value, 10)
	for index := len(digits) - 3; index > 0; index -= 3 {
		digits = digits[:index] + "," + digits[index:]
	}
	return fmt.Sprintf("$%s", digits)
}
