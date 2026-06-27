package strategy

import (
	"strings"
	"testing"

	"sentra/internal/pine"
)

func TestPredefinedTemplatesIncludesDocumentStrategies(t *testing.T) {
	templates := PredefinedTemplates()
	if len(templates) != 10 {
		t.Fatalf("expected 10 predefined templates, got %d", len(templates))
	}

	expectedIDs := []string{
		StrategyTrendFollowingMTF,
		StrategyMomentumBreakoutVolume,
		StrategyAdaptiveMeanReversion,
		StrategyStatArbPairs,
		StrategyVWAPReversion,
		StrategyCryptoMarketMaking,
		StrategyFundingRateArbitrage,
		StrategyGridTrading,
		StrategyMultiFactorMomentum,
		StrategySmartMoneyOrderFlow,
	}

	seen := map[string]bool{}
	for _, tmpl := range templates {
		seen[tmpl.ID] = true
	}
	for _, id := range expectedIDs {
		if !seen[id] {
			t.Fatalf("expected template id %q", id)
		}
	}
}

func TestPredefinedTemplatesIntegrity(t *testing.T) {
	templates := PredefinedTemplates()
	names := map[string]bool{}
	validStatuses := map[TemplateSupportStatus]bool{
		TemplateExecutableNative: true,
		TemplateExecutablePine:   true,
		TemplateTemplateOnly:     true,
		TemplateBlockedByData:    true,
	}

	for _, tmpl := range templates {
		if tmpl.ID == "" {
			t.Fatalf("template has empty id: %+v", tmpl)
		}
		if tmpl.Name == "" || tmpl.Category == "" || tmpl.Summary == "" {
			t.Fatalf("template %q is missing display metadata: %+v", tmpl.ID, tmpl)
		}
		if names[tmpl.Name] {
			t.Fatalf("duplicate template name %q", tmpl.Name)
		}
		names[tmpl.Name] = true
		if !validStatuses[tmpl.SupportStatus] {
			t.Fatalf("template %q has invalid support status %q", tmpl.ID, tmpl.SupportStatus)
		}
		if len(tmpl.RequiredData) == 0 {
			t.Fatalf("template %q must declare required data", tmpl.ID)
		}
		if len(tmpl.RiskRules) == 0 {
			t.Fatalf("template %q must declare risk rules", tmpl.ID)
		}
		if tmpl.SupportStatus == TemplateBlockedByData && len(tmpl.Blockers) == 0 {
			t.Fatalf("data-blocked template %q must explain blockers", tmpl.ID)
		}
		if tmpl.SupportStatus == TemplateExecutablePine && tmpl.PineCode == "" {
			t.Fatalf("executable Pine template %q must include pine code", tmpl.ID)
		}
	}
}

func TestGetPredefinedTemplate(t *testing.T) {
	tmpl, ok := GetPredefinedTemplate(StrategyMultiFactorMomentum)
	if !ok {
		t.Fatalf("expected multi-factor template")
	}
	if tmpl.ID != StrategyMultiFactorMomentum {
		t.Fatalf("expected id %q, got %q", StrategyMultiFactorMomentum, tmpl.ID)
	}

	if _, ok := GetPredefinedTemplate("missing-template"); ok {
		t.Fatalf("expected missing template lookup to return false")
	}
}

func TestExecutablePineTemplatesParse(t *testing.T) {
	for _, tmpl := range PredefinedTemplates() {
		if tmpl.SupportStatus != TemplateExecutablePine {
			continue
		}
		res := pine.NewParser(tmpl.PineCode).Parse()
		if len(res.Errors) > 0 {
			t.Fatalf("template %q Pine code should parse without errors: %v", tmpl.ID, res.Errors)
		}
	}
}

func TestExecutableTemplatesHaveExecutionProfiles(t *testing.T) {
	for _, tmpl := range PredefinedTemplates() {
		if tmpl.SupportStatus != TemplateExecutablePine {
			continue
		}
		if tmpl.ExecutionProfile.RecommendedInterval == "" {
			t.Fatalf("template %s missing recommended interval", tmpl.ID)
		}
		if tmpl.ExecutionProfile.CooldownBars <= 0 {
			t.Fatalf("template %s missing cooldown bars", tmpl.ID)
		}
		if tmpl.ExecutionProfile.MinHoldingBars <= 0 {
			t.Fatalf("template %s missing min holding bars", tmpl.ID)
		}
		if tmpl.ExecutionProfile.MaxTradesPerDay <= 0 {
			t.Fatalf("template %s missing max trades per day", tmpl.ID)
		}
	}
}

func TestMultiFactorMomentumExecutablePineUsesMACDConfirmation(t *testing.T) {
	tmpl, ok := GetPredefinedTemplate(StrategyMultiFactorMomentum)
	if !ok {
		t.Fatalf("expected multi-factor template")
	}

	if !strings.Contains(tmpl.PineCode, "ta.macd(close, 12, 26, 9)") {
		t.Fatalf("multi-factor momentum Pine should calculate MACD confirmation")
	}
	if !strings.Contains(tmpl.PineCode, "ta.crossover(macdLine, macdLine.signal)") {
		t.Fatalf("multi-factor momentum Pine should require bullish MACD crossover confirmation")
	}
	if !strings.Contains(tmpl.PineCode, "ta.crossunder(macdLine, macdLine.signal)") {
		t.Fatalf("multi-factor momentum Pine should require bearish MACD crossunder confirmation")
	}
	if !strings.Contains(tmpl.PineCode, "macdLine > macdLine.signal") || !strings.Contains(tmpl.PineCode, "rsi14 > 55") {
		t.Fatalf("multi-factor momentum Pine should include risk-on long continuation confirmation")
	}
	if !strings.Contains(tmpl.PineCode, "strategy.entry(\"SHORT\", strategy.short)") {
		t.Fatalf("multi-factor momentum Pine should include a short-side momentum entry")
	}
	if !tmpl.ExecutionProfile.ShortingEnabled {
		t.Fatalf("multi-factor momentum execution profile should enable shorting")
	}
	if !tmpl.ExecutionProfile.RegimeFilterEnabled {
		t.Fatalf("multi-factor momentum execution profile should enable the ATR regime filter")
	}
	if tmpl.ExecutionProfile.RegimeFilterPeriod <= 0 || tmpl.ExecutionProfile.RegimeMinATRPercent <= 0 || tmpl.ExecutionProfile.RegimeMaxATRPercent <= 0 {
		t.Fatalf("multi-factor momentum execution profile should define ATR regime bounds, got %+v", tmpl.ExecutionProfile)
	}
	if tmpl.ExecutionProfile.PositionSizePercent != 10 {
		t.Fatalf("multi-factor momentum execution profile should keep 10%% baseline allocation, got %+v", tmpl.ExecutionProfile)
	}
	for _, blocker := range tmpl.Blockers {
		if strings.Contains(strings.ToLower(blocker), "omits macd") {
			t.Fatalf("multi-factor momentum should not report MACD as omitted after executable confirmation is added")
		}
	}
}
