package bootstrap

import (
	"testing"

	appconfig "github.com/niflaot/corps-manager/platform/app"
	"go.uber.org/fx"
)

func TestModuleGraph(t *testing.T) {
	if err := fx.ValidateApp(Module, fx.Supply(appconfig.Version("test")), fx.NopLogger); err != nil {
		t.Fatalf("ValidateApp() error = %v", err)
	}
}
