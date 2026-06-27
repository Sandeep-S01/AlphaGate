package risk

import (
	"strings"
	"testing"
	"time"
)

func TestSettingsValidateRejectsInvalidLimits(t *testing.T) {
	settings := DefaultSettings()
	settings.MinSignalStrength = -1

	if err := settings.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestBuildUpsertSettingsSQLUsesSettingsFields(t *testing.T) {
	settings := DefaultSettings()
	settings.MaxQuoteAmount = 250
	settings.MaxDailyTrades = 5
	settings.Cooldown = 10 * time.Minute
	settings.AllowedSymbols = []string{"BTCUSDT"}
	settings.MaxOrderQuoteAmount = 125
	settings.MaxPositionQuoteAmount = 500
	settings.MaxTotalExposureQuoteAmount = 1000
	settings.MaxOpenPositions = 2

	query, args := BuildUpsertSettingsSQL(settings)

	if !strings.Contains(query, "INSERT INTO risk_settings") {
		t.Fatalf("expected risk_settings insert, got %s", query)
	}
	if !strings.Contains(query, "ON CONFLICT") {
		t.Fatalf("expected upsert conflict handling, got %s", query)
	}
	if len(args) != 14 {
		t.Fatalf("expected 14 args, got %d", len(args))
	}
	if args[1] != 250.0 || args[4] != 5 || args[7] != int64((10*time.Minute).Seconds()) {
		t.Fatalf("unexpected args: %+v", args)
	}
	if args[10] != 125.0 || args[11] != 500.0 || args[12] != 1000.0 || args[13] != 2 {
		t.Fatalf("unexpected hardened args: %+v", args)
	}
}
