package httpapi

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/niflaot/corps-manager/internal/agreements"
	appconfig "github.com/niflaot/corps-manager/platform/app"
	"github.com/niflaot/corps-manager/platform/health"
	"go.uber.org/zap"
)

type companyStub struct {
	items   []agreements.Company
	err     error
	deleted string
}

func (stub *companyStub) CreateCompany(_ context.Context, id, name string) (agreements.Company, error) {
	item := agreements.Company{ID: id, Name: name}
	stub.items = append(stub.items, item)
	return item, stub.err
}
func (stub *companyStub) ListCompanies(context.Context) ([]agreements.Company, error) {
	return stub.items, stub.err
}
func (stub *companyStub) DeleteCompany(_ context.Context, id string) error {
	stub.deleted = id
	return stub.err
}

func TestCompanyAPIAuthenticationAndMutations(t *testing.T) {
	stub := &companyStub{}
	app := New(zap.NewNop(), appconfig.Config{Environment: appconfig.EnvironmentTest}, Config{APIKey: "test-api-key-long", BodyLimit: 1 << 20}, health.New(nil), Dependencies{Companies: stub}, "test")
	request := authenticatedRequest(http.MethodPost, "/api/companies", `{"id":"company-a","name":"Empresa A"}`)
	request.Header.Del("Authorization")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized || len(stub.items) != 0 {
		t.Fatal("unauthenticated company creation accepted")
	}
	response, err = app.Test(authenticatedRequest(http.MethodPost, "/api/companies", `{"id":"company-a","name":"Empresa A"}`))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusCreated || len(stub.items) != 1 {
		t.Fatalf("status=%d items=%v", response.StatusCode, stub.items)
	}
	response, err = app.Test(authenticatedRequest(http.MethodPost, "/api/companies", `{"id":"b","name":"B","unexpected":true}`))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown JSON field accepted: %d", response.StatusCode)
	}
	stub.err = agreements.ErrCompanyInUse
	response, err = app.Test(authenticatedRequest(http.MethodDelete, "/api/companies/company-a", ""))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("company in use: %d", response.StatusCode)
	}
	stub.err = nil
	response, err = app.Test(authenticatedRequest(http.MethodDelete, "/api/companies/empty", ""))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent || stub.deleted != "empty" {
		t.Fatal("company deletion failed")
	}
}

func TestRetiredRoutesAreNotRegistered(t *testing.T) {
	app := New(zap.NewNop(), appconfig.Config{}, Config{APIKey: "test", BodyLimit: 1 << 20}, health.New(nil), Dependencies{}, "test")
	for _, path := range []string{"/customers", "/api/performance", "/api/inactivity", "/api/announcements/opening"} {
		response, err := app.Test(authenticatedRequest(http.MethodGet, path, ""))
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusNotFound {
			t.Fatalf("retired route still exists: %s", path)
		}
	}
	if companyError(errors.New("database secret")).Error() != "company operation failed" {
		t.Fatal("internal details leaked")
	}
}
