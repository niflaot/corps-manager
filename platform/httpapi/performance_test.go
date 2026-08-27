package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/niflaot/corps-manager/internal/performance"
	appconfig "github.com/niflaot/corps-manager/platform/app"
	"github.com/niflaot/corps-manager/platform/health"
	"go.uber.org/zap"
)

type performanceHTTPStub struct {
	request performance.CurrentPeriodBackfill
	state   performance.State
	err     error
}

func (stub *performanceHTTPStub) Get(context.Context) (performance.State, error) {
	return stub.state, stub.err
}

func (stub *performanceHTTPStub) Refresh(context.Context) (performance.State, error) {
	return stub.state, stub.err
}

func (stub *performanceHTTPStub) BackfillCurrentPeriod(_ context.Context,
	request performance.CurrentPeriodBackfill) (performance.State, error) {
	stub.request = request
	return stub.state, stub.err
}

func TestPerformanceBackfillRouteIsProtectedAndDecodesGuard(t *testing.T) {
	stub := &performanceHTTPStub{state: performance.State{BusinessID: 1995}}
	application := New(zap.NewNop(), appconfig.Config{Environment: appconfig.EnvironmentTest},
		Config{APIKey: "test-api-key-long", BodyLimit: 1 << 20}, health.New(nil),
		Dependencies{Performance: stub}, "1.0.0")
	path := "/api/performance/current-period/backfill"
	unauthorized, err := application.Test(httptest.NewRequest(http.MethodPost, path, nil))
	if err != nil {
		t.Fatal(err)
	}
	if unauthorized.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", unauthorized.StatusCode)
	}
	body := `{"periodStartedAt":"2026-08-25T00:00:00-05:00","characterIds":[8247,13507]}`
	response, err := application.Test(authenticatedRequest(http.MethodPost, path, body))
	if err != nil || response.StatusCode != http.StatusOK {
		t.Fatalf("backfill status = %d, error = %v", response.StatusCode, err)
	}
	expected := time.Date(2026, time.August, 25, 0, 0, 0, 0, time.FixedZone("-05", -5*60*60))
	if !stub.request.PeriodStartedAt.Equal(expected) || len(stub.request.CharacterIDs) != 2 ||
		stub.request.CharacterIDs[0] != 8247 {
		t.Fatalf("backfill request = %#v", stub.request)
	}
}
