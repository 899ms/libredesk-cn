// [cn-fork]
package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_9_1 seeds the initial feature toggle settings.
func V2_9_1(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES
			('feature.livechat.enabled', 'true'::jsonb),
			('feature.helpcenter.enabled', 'false'::jsonb),
			('feature.ai.enabled', 'false'::jsonb),
			('feature.csat.enabled', 'true'::jsonb),
			('feature.email_channel.enabled', 'false'::jsonb),
			('feature.user_chat.enabled', 'false'::jsonb)
		ON CONFLICT (key) DO NOTHING;
	`); err != nil {
		return err
	}

	return nil
}
