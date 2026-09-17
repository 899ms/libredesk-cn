// [cn-fork]
package feature

import (
	"encoding/json"
	"testing"

	"github.com/jmoiron/sqlx/types"
)

type mockSettingProvider struct {
	data map[string]any
}

func (m *mockSettingProvider) GetByPrefix(prefix string) (types.JSONText, error) {
	b, err := json.Marshal(m.data)
	return types.JSONText(b), err
}

func TestFeatureDefaults(t *testing.T) {
	mgr, err := New(nil, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !mgr.Enabled(LiveChat) {
		t.Errorf("expected %s to be enabled by default", LiveChat)
	}
	if mgr.Enabled(AI) {
		t.Errorf("expected %s to be disabled by default", AI)
	}
	if mgr.Enabled(HelpCenter) {
		t.Errorf("expected %s to be disabled by default", HelpCenter)
	}
	if !mgr.Enabled(CSAT) {
		t.Errorf("expected %s to be enabled by default", CSAT)
	}
	if mgr.Enabled("non_existent") {
		t.Errorf("expected non_existent to be false")
	}
}

func TestFeatureReload(t *testing.T) {
	mockSP := &mockSettingProvider{
		data: map[string]any{
			"feature.ai.enabled":         true,
			"feature.livechat.enabled":   false,
			"feature.helpcenter.enabled": true,
		},
	}

	mgr, err := New(nil, mockSP)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !mgr.Enabled(AI) {
		t.Errorf("expected %s to be enabled after load", AI)
	}
	if mgr.Enabled(LiveChat) {
		t.Errorf("expected %s to be disabled after load", LiveChat)
	}
	if !mgr.Enabled(HelpCenter) {
		t.Errorf("expected %s to be enabled after load", HelpCenter)
	}

	// Now update settings and reload
	mockSP.data["feature.ai.enabled"] = false
	if err := mgr.Reload(); err != nil {
		t.Fatalf("failed to reload: %v", err)
	}

	if mgr.Enabled(AI) {
		t.Errorf("expected %s to be disabled after reload", AI)
	}
}

func TestFeatureAll(t *testing.T) {
	mgr, err := New(nil, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	all := mgr.All()
	if len(all) != 6 {
		t.Errorf("expected 6 features, got %d", len(all))
	}
	if all[LiveChat] != true || all[AI] != false {
		t.Errorf("unexpected feature states: %+v", all)
	}
}
