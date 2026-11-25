package message

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"github.com/francesandprincesss/new-telegram/internal/reaction"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GORMRepository struct {
	db     *gorm.DB
	cipher bodyCipher
}

func NewRepository(db *gorm.DB, key []byte) (*GORMRepository, error) {
	cipher, err := newBodyCipher(key)
	if err != nil {
		return nil, err
	}
	return &GORMRepository{db: db, cipher: cipher}, nil
}

func (r *GORMRepository) CreateAndTouchDialog(ctx context.Context, input NewMessage) (Message, error) {
	encrypted, err := r.cipher.encrypt(input.Body, input.ID, input.DialogID, input.SenderID)
	if err != nil {
		return Message{}, apperror.Wrap(apperror.CodeInternal, "could not protect message", err)
	}
	var reply *ReplyPreview
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock the dialog before inserting so concurrent sends share a stable
		// chronology for pagination and read cutoffs, even within one microsecond.
		stamp := struct{ UpdatedAt time.Time }{}
		result := tx.Raw(`
			UPDATE dialogs
			SET updated_at = GREATEST(?, updated_at + INTERVAL '1 microsecond')
			WHERE id = ?
			RETURNING updated_at`, input.CreatedAt, input.DialogID).Scan(&stamp)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		if input.ReplyToID != nil {
			var original storedReply
			err := tx.Table("messages").Select("id, sender_id, body_ciphertext").
				Where("dialog_id = ? AND id = ?", input.DialogID, *input.ReplyToID).Take(&original).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperror.New(apperror.CodeNotFound, "quoted message not found in this dialog")
			}
			if err != nil {
				return err
			}
			reply, err = r.replyPreview(original, input.DialogID)
			if err != nil {
				return err
			}
		}
		input.CreatedAt = stamp.UpdatedAt
		return tx.Table("messages").Create(map[string]any{
			"id": input.ID, "dialog_id": input.DialogID, "sender_id": input.SenderID,
			"body_ciphertext": encrypted, "created_at": input.CreatedAt, "reply_to_id": input.ReplyToID,
		}).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Message{}, apperror.New(apperror.CodeNotFound, "dialog not found")
	}
	if err != nil {
		if apperror.CodeOf(err) == apperror.CodeNotFound {
			return Message{}, err
		}
		return Message{}, apperror.Wrap(apperror.CodeInternal, "could not send message", err)
	}
	return Message{
		ID: input.ID, DialogID: input.DialogID, SenderID: input.SenderID,
		Body: input.Body, CreatedAt: input.CreatedAt, ReplyTo: reply,
	}, nil
}

func (r *GORMRepository) ListByDialog(ctx context.Context, dialogID uuid.UUID, cursor *Cursor, limit int) ([]Message, error) {
	var rows []struct {
		Message
		BodyCiphertext []byte
		Original       storedReply `gorm:"embedded;embeddedPrefix:reply_"`
	}
	query := r.db.WithContext(ctx).Table("messages").Select(`messages.id, messages.dialog_id, messages.sender_id, messages.body_ciphertext, messages.created_at,
		original.id AS reply_id, original.sender_id AS reply_sender_id, original.body_ciphertext AS reply_body_ciphertext,
		(SELECT read_at FROM read_receipts
		 WHERE message_id = messages.id AND user_id <> messages.sender_id
		 ORDER BY read_at LIMIT 1) AS read_at`).
		Joins("LEFT JOIN messages AS original ON original.id = messages.reply_to_id AND original.dialog_id = messages.dialog_id").
		Where("messages.dialog_id = ?", dialogID)
	if cursor != nil {
		query = query.Where("(messages.created_at, messages.id) < (?, ?)", cursor.CreatedAt, cursor.ID)
	}
	if err := query.Order("messages.created_at DESC, messages.id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "could not load messages", err)
	}
	messages := make([]Message, len(rows))
	for i, row := range rows {
		body, err := r.cipher.decrypt(row.BodyCiphertext, row.ID, row.DialogID, row.SenderID)
		if err != nil {
			return nil, apperror.Wrap(apperror.CodeInternal, "could not read protected message", err)
		}
		messages[i] = row.Message
		messages[i].Body = body
		if row.Original.ID != uuid.Nil {
			messages[i].ReplyTo, err = r.replyPreview(row.Original, row.DialogID)
			if err != nil {
				return nil, apperror.Wrap(apperror.CodeInternal, "could not read quoted message", err)
			}
		}
	}
	if err := r.loadReactions(ctx, messages); err != nil {
		return nil, err
	}
	return messages, nil
}

type storedReply struct {
	ID             uuid.UUID
	SenderID       uuid.UUID
	BodyCiphertext []byte
}

func (r *GORMRepository) replyPreview(original storedReply, dialogID uuid.UUID) (*ReplyPreview, error) {
	body, err := r.cipher.decrypt(original.BodyCiphertext, original.ID, dialogID, original.SenderID)
	if err != nil {
		return nil, err
	}
	excerpt := strings.Join(strings.Fields(body), " ")
	const maxExcerpt = 180
	if characters := []rune(excerpt); len(characters) > maxExcerpt {
		excerpt = string(characters[:maxExcerpt-1]) + "…"
	}
	return &ReplyPreview{ID: original.ID, SenderID: original.SenderID, Excerpt: excerpt}, nil
}

func (r *GORMRepository) loadReactions(ctx context.Context, messages []Message) error {
	if len(messages) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(messages))
	positions := make(map[uuid.UUID]int, len(messages))
	for i, message := range messages {
		ids = append(ids, message.ID)
		positions[message.ID] = i
	}
	var reactions []reaction.Reaction
	err := r.db.WithContext(ctx).Table("reactions").Where("message_id IN ?", ids).Order("user_id").Find(&reactions).Error
	if err != nil {
		return apperror.Wrap(apperror.CodeInternal, "could not load reactions", err)
	}
	for _, item := range reactions {
		index := positions[item.MessageID]
		messages[index].Reactions = append(messages[index].Reactions, item)
	}
	return nil
}

func (r *GORMRepository) GetMetadataByID(ctx context.Context, messageID uuid.UUID) (Message, error) {
	var message Message
	err := r.db.WithContext(ctx).Table("messages").Select("id, dialog_id, sender_id").Where("id = ?", messageID).Take(&message).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Message{}, apperror.New(apperror.CodeNotFound, "message not found")
	}
	if err != nil {
		return Message{}, apperror.Wrap(apperror.CodeInternal, "could not load message", err)
	}
	return message, nil
}

func (r *GORMRepository) CountUnread(ctx context.Context, dialogID, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("messages").
		Where(`dialog_id = ? AND sender_id <> ? AND NOT EXISTS (
			SELECT 1 FROM read_receipts WHERE message_id = messages.id AND user_id = ?
		)`, dialogID, userID, userID).Count(&count).Error
	if err != nil {
		return 0, apperror.Wrap(apperror.CodeInternal, "could not count unread messages", err)
	}
	return count, nil
}
