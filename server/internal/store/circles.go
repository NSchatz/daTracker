package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/model"
)

func generateInviteCode() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate invite code: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func (s *Store) CreateCircle(ctx context.Context, name string, createdBy uuid.UUID) (*model.Circle, error) {
	code, err := generateInviteCode()
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var c model.Circle
	err = tx.QueryRow(ctx,
		`INSERT INTO circles (name, invite_code, created_by)
		 VALUES ($1, $2, $3)
		 RETURNING id, name, invite_code, created_by, created_at`,
		name, code, createdBy,
	).Scan(&c.ID, &c.Name, &c.InviteCode, &c.CreatedBy, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert circle: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO circle_members (circle_id, user_id, role)
		 VALUES ($1, $2, 'admin')`,
		c.ID, createdBy,
	)
	if err != nil {
		return nil, fmt.Errorf("add creator as admin: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create circle: %w", err)
	}

	return &c, nil
}

func (s *Store) GetCircleByInviteCode(ctx context.Context, code string) (*model.Circle, error) {
	var c model.Circle
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, invite_code, created_by, created_at
		 FROM circles WHERE invite_code = $1`,
		code,
	).Scan(&c.ID, &c.Name, &c.InviteCode, &c.CreatedBy, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get circle by invite code: %w", err)
	}
	return &c, nil
}

func (s *Store) AddMember(ctx context.Context, circleID, userID uuid.UUID, role string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO circle_members (circle_id, user_id, role)
		 VALUES ($1, $2, $3)
		 ON CONFLICT DO NOTHING`,
		circleID, userID, role,
	)
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}
	return nil
}

func (s *Store) GetMembers(ctx context.Context, circleID uuid.UUID) ([]model.CircleMember, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT cm.circle_id, cm.user_id, cm.role, cm.joined_at, u.display_name, u.email
		 FROM circle_members cm
		 JOIN users u ON u.id = cm.user_id
		 WHERE cm.circle_id = $1`,
		circleID,
	)
	if err != nil {
		return nil, fmt.Errorf("get members: %w", err)
	}
	defer rows.Close()

	var members []model.CircleMember
	for rows.Next() {
		var m model.CircleMember
		if err := rows.Scan(&m.CircleID, &m.UserID, &m.Role, &m.JoinedAt, &m.DisplayName, &m.Email); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate members: %w", err)
	}
	return members, nil
}

func (s *Store) GetUserCircles(ctx context.Context, userID uuid.UUID) ([]model.Circle, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT c.id, c.name, c.invite_code, c.created_by, c.created_at
		 FROM circles c
		 JOIN circle_members cm ON cm.circle_id = c.id
		 WHERE cm.user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get user circles: %w", err)
	}
	defer rows.Close()

	var circles []model.Circle
	for rows.Next() {
		var c model.Circle
		if err := rows.Scan(&c.ID, &c.Name, &c.InviteCode, &c.CreatedBy, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan circle: %w", err)
		}
		circles = append(circles, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate circles: %w", err)
	}
	return circles, nil
}
