// [cn-fork] Package feature provides runtime toggleable feature modules.
package feature

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/jmoiron/sqlx/types"
	"github.com/zerodha/logf"
)

const (
	LiveChat     = "livechat"
	HelpCenter   = "helpcenter"
	AI           = "ai"
	CSAT         = "csat"
	EmailChannel = "email_channel"
	UserChat     = "user_chat"
)

// defaultFeatures defines the out-of-the-box state for feature toggles.
var defaultFeatures = map[string]bool{
	LiveChat:     true,
	HelpCenter:   false,
	AI:           false,
	CSAT:         true,
	EmailChannel: false,
	UserChat:     false,
}

// SettingProvider is the minimal interface required to read feature settings.
type SettingProvider interface {
	GetByPrefix(prefix string) (types.JSONText, error)
}

// Manager manages in-memory feature flags with reload support.
type Manager struct {
	mu       sync.RWMutex
	features map[string]bool
	settings SettingProvider
	lo       *logf.Logger
}

// New creates a new feature Manager and loads current settings.
func New(lo *logf.Logger, sp SettingProvider) (*Manager, error) {
	m := &Manager{
		features: make(map[string]bool),
		settings: sp,
		lo:       lo,
	}

	// Initialize with defaults
	for k, v := range defaultFeatures {
		m.features[k] = v
	}

	if sp != nil {
		if err := m.Reload(); err != nil {
			if lo != nil {
				lo.Error("failed to load feature settings, falling back to defaults", "error", err)
			}
		}
	}

	return m, nil
}

// Enabled checks if a feature module is enabled.
func (m *Manager) Enabled(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if enabled, ok := m.features[name]; ok {
		return enabled
	}
	return false
}

// All returns a copy of all current feature statuses.
func (m *Manager) All() map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make(map[string]bool, len(m.features))
	for k, v := range m.features {
		out[k] = v
	}
	return out
}

// Set sets a feature state in-memory (useful for testing or programmatic overrides).
func (m *Manager) Set(name string, enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.features[name] = enabled
}

// Reload reads all feature.* settings from the settings store and refreshes the in-memory map.
func (m *Manager) Reload() error {
	if m.settings == nil {
		return nil
	}

	data, err := m.settings.GetByPrefix("feature")
	if err != nil {
		return err
	}

	if len(data) == 0 || string(data) == "null" || string(data) == "{}" {
		return nil
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Reset to defaults first to ensure consistent state
	for k, v := range defaultFeatures {
		m.features[k] = v
	}

	// Apply database values: key pattern is "feature.<name>.enabled"
	for k, v := range raw {
		if !strings.HasPrefix(k, "feature.") {
			continue
		}
		name := strings.TrimPrefix(k, "feature.")
		name = strings.TrimSuffix(name, ".enabled")

		switch val := v.(type) {
		case bool:
			m.features[name] = val
		case string:
			m.features[name] = (val == "true" || val == "1")
		}
	}

	return nil
}
