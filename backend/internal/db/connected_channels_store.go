package db

import (
	"context"
	"database/sql"
	"time"

	"ytagent/backend/internal/models"
)

type ConnectedChannelStore struct {
	db *sql.DB
}

func NewConnectedChannelStore(db *sql.DB) *ConnectedChannelStore {
	return &ConnectedChannelStore{db: db}
}

// Upsert links a channel by its YouTube channel ID — reconnecting the same
// channel (e.g. after revoking and redoing consent) replaces its tokens
// rather than creating a duplicate row.
func (s *ConnectedChannelStore) Upsert(ctx context.Context, ch models.ConnectedChannel) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO connected_channels
			(channel_id, channel_title, channel_thumbnail, subscriber_count, google_email,
			 access_token, refresh_token, token_expiry, scopes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (channel_id) DO UPDATE SET
			channel_title     = EXCLUDED.channel_title,
			channel_thumbnail = EXCLUDED.channel_thumbnail,
			subscriber_count  = EXCLUDED.subscriber_count,
			google_email      = EXCLUDED.google_email,
			access_token      = EXCLUDED.access_token,
			-- Google occasionally omits a fresh refresh_token on re-consent;
			-- never overwrite a working one with an empty string.
			refresh_token     = COALESCE(NULLIF(EXCLUDED.refresh_token, ''), connected_channels.refresh_token),
			token_expiry      = EXCLUDED.token_expiry,
			scopes            = EXCLUDED.scopes,
			updated_at        = now()
		RETURNING id`,
		ch.ChannelID, ch.ChannelTitle, ch.ChannelThumbnail, ch.SubscriberCount, ch.GoogleEmail,
		ch.AccessToken, ch.RefreshToken, ch.TokenExpiry, ch.Scopes,
	).Scan(&id)
	return id, err
}

func (s *ConnectedChannelStore) List(ctx context.Context) ([]models.ConnectedChannel, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, channel_id, channel_title, channel_thumbnail, subscriber_count, google_email,
		       access_token, refresh_token, token_expiry, scopes, created_at, updated_at
		FROM connected_channels ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.ConnectedChannel
	for rows.Next() {
		var ch models.ConnectedChannel
		var thumbnail, email sql.NullString
		if err := rows.Scan(&ch.ID, &ch.ChannelID, &ch.ChannelTitle, &thumbnail, &ch.SubscriberCount, &email,
			&ch.AccessToken, &ch.RefreshToken, &ch.TokenExpiry, &ch.Scopes, &ch.CreatedAt, &ch.UpdatedAt); err != nil {
			return nil, err
		}
		ch.ChannelThumbnail = thumbnail.String
		ch.GoogleEmail = email.String
		out = append(out, ch)
	}
	return out, rows.Err()
}

func (s *ConnectedChannelStore) GetByID(ctx context.Context, id string) (*models.ConnectedChannel, error) {
	var ch models.ConnectedChannel
	var thumbnail, email sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, channel_id, channel_title, channel_thumbnail, subscriber_count, google_email,
		       access_token, refresh_token, token_expiry, scopes, created_at, updated_at
		FROM connected_channels WHERE id = $1`, id,
	).Scan(&ch.ID, &ch.ChannelID, &ch.ChannelTitle, &thumbnail, &ch.SubscriberCount, &email,
		&ch.AccessToken, &ch.RefreshToken, &ch.TokenExpiry, &ch.Scopes, &ch.CreatedAt, &ch.UpdatedAt)
	if err != nil {
		return nil, err
	}
	ch.ChannelThumbnail = thumbnail.String
	ch.GoogleEmail = email.String
	return &ch, nil
}

// UpdateTokens persists a refreshed access token so the next request doesn't
// need to hit Google's token endpoint again before its new expiry.
func (s *ConnectedChannelStore) UpdateTokens(ctx context.Context, id, accessToken string, expiry time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE connected_channels SET access_token = $1, token_expiry = $2, updated_at = now() WHERE id = $3`,
		accessToken, expiry, id,
	)
	return err
}

func (s *ConnectedChannelStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM connected_channels WHERE id = $1`, id)
	return err
}
