// [cn-fork]
package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_9_2 adds seen_at to conversation_messages and max_open_conversations to users.
func V2_9_2(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		ALTER TABLE conversation_messages ADD COLUMN IF NOT EXISTS seen_at TIMESTAMPTZ NULL;
		CREATE INDEX IF NOT EXISTS index_conversation_messages_on_seen_at ON conversation_messages (seen_at);
		ALTER TABLE users ADD COLUMN IF NOT EXISTS max_open_conversations INT DEFAULT 0 NOT NULL;
	`); err != nil {
		return err
	}

	return nil
}
