package customerweb

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/niflaot/corps-manager/internal/customers"
)

type repositoryStub struct{ query customers.Query }

func (*repositoryStub) Record(context.Context, string, string, string, int64) (customers.Customer, error) {
	return customers.Customer{}, nil
}

func (*repositoryStub) List(context.Context) ([]customers.Customer, error) { return nil, nil }

func (repository *repositoryStub) Search(_ context.Context, query customers.Query) ([]customers.Customer, error) {
	repository.query = query
	return []customers.Customer{{Name: "jane_doe", Visits: 2, TotalSpent: 5000, AttendantCount: 1}}, nil
}

func (*repositoryStub) Get(context.Context, string) (customers.Customer, error) {
	return customers.Customer{}, nil
}

func (*repositoryStub) Delete(context.Context, string) error { return nil }

func TestHandleFiltersAndRendersCustomers(t *testing.T) {
	t.Parallel()
	repository := &repositoryStub{}
	service := customers.NewService(customers.Config{Enabled: true}, repository, nil, "")
	application := fiber.New()
	application.Get("/customers", New(service).Handle)
	response, err := application.Test(httptest.NewRequest("GET",
		"/customers?name=Jane+Doe&days=30&sort=spend", nil))
	if err != nil {
		t.Fatalf("application.Test: %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response.StatusCode != 200 || !strings.Contains(string(body), "Jane Doe") ||
		!strings.Contains(string(body), "$5,000") {
		t.Fatalf("status = %d, body = %q", response.StatusCode, body)
	}
	if repository.query.Name != "jane_doe" || repository.query.Days != 30 ||
		repository.query.Sort != customers.SortSpend {
		t.Fatalf("query = %+v", repository.query)
	}
}
