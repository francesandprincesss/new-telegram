package message

import (
	"context"
	"crypto/hmac"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ProtectStoredMessages verifies the configured key and encrypts old rows before
// the server starts listening. Committed batches survive a restart.
func (r *GORMRepository) ProtectStoredMessages(ctx context.Context) error {
	db := r.db.WithContext(ctx)
	if err := db.Exec(`INSERT INTO message_encryption_key (singleton, verifier)
		VALUES (TRUE, ?) ON CONFLICT (singleton) DO NOTHING`, r.cipher.verifier).Error; err != nil {
		return fmt.Errorf("initialize message encryption key: %w", err)
	}
	var stored struct{ Verifier []byte }
	if err := db.Table("message_encryption_key").Where("singleton").Take(&stored).Error; err != nil {
		return fmt.Errorf("verify message encryption key: %w", err)
	}
	if !hmac.Equal(stored.Verifier, r.cipher.verifier) {
		return errors.New("MESSAGE_ENCRYPTION_KEY does not match the database")
	}

	for {
		var batch []NewMessage
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Table("messages").Select("id, dialog_id, sender_id, body").
				Where("body IS NOT NULL").Order("id").Limit(128).
				Clauses(clause.Locking{Strength: "UPDATE"}).Find(&batch).Error; err != nil {
				return err
			}
			for _, row := range batch {
				encrypted, err := r.cipher.encrypt(row.Body, row.ID, row.DialogID, row.SenderID)
				if err != nil {
					return err
				}
				if err := tx.Table("messages").Where("id = ?", row.ID).
					UpdateColumns(map[string]any{"body": nil, "body_ciphertext": encrypted}).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("encrypt stored messages: %w", err)
		}
		if len(batch) == 0 {
			return nil
		}
	}
}
