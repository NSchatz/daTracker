package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (s *Store) UpsertFCMToken(ctx context.Context, userID uuid.UUID, token string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO fcm_tokens (user_id, token)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id) DO UPDATE SET token = $2, updated_at = now()`,
		userID, token,
	)
	if err != nil {
		return fmt.Errorf("upsert fcm token: %w", err)
	}
	return nil
}

func (s *Store) GetFCMTokensForCircle(ctx context.Context, circleID uuid.UUID, excludeUserID uuid.UUID) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT ft.token FROM fcm_tokens ft
		 JOIN circle_members cm ON cm.user_id = ft.user_id
		 WHERE cm.circle_id = $1 AND ft.user_id != $2`,
		circleID, excludeUserID,
	)
	if err != nil {
		return nil, fmt.Errorf("get fcm tokens for circle: %w", err)
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, fmt.Errorf("scan fcm token: %w", err)
		}
		tokens = append(tokens, token)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate fcm tokens: %w", err)
	}
	return tokens, nil
}
