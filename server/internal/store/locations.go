package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/model"
)

// InsertLocations bulk-inserts location points using a single INSERT with multiple VALUE tuples.
func (s *Store) InsertLocations(ctx context.Context, userID uuid.UUID, locs []model.LocationInput) error {
	if len(locs) == 0 {
		return nil
	}

	// Build: INSERT INTO locations (user_id, point, speed, battery_level, accuracy, recorded_at) VALUES ...
	// ST_MakePoint takes (lng, lat) and we cast to geography.
	// Each row occupies 7 params: user_id, lng, lat, speed, battery_level, accuracy, recorded_at
	valueStrings := make([]string, 0, len(locs))
	args := make([]any, 0, len(locs)*7)

	for i, loc := range locs {
		base := i * 7
		valueStrings = append(valueStrings, fmt.Sprintf(
			"($%d, ST_SetSRID(ST_MakePoint($%d, $%d), 4326)::geography, $%d, $%d, $%d, $%d)",
			base+1, base+2, base+3, base+4, base+5, base+6, base+7,
		))
		args = append(args, userID, loc.Lng, loc.Lat, loc.Speed, loc.BatteryLevel, loc.Accuracy, loc.RecordedAt)
	}

	query := "INSERT INTO locations (user_id, point, speed, battery_level, accuracy, recorded_at) VALUES " +
		strings.Join(valueStrings, ", ")

	_, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert locations: %w", err)
	}
	return nil
}

// GetLatestLocations returns the most recent location for each circle member.
func (s *Store) GetLatestLocations(ctx context.Context, circleID uuid.UUID) ([]model.Location, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT ON (l.user_id)
			l.id,
			l.user_id,
			ST_Y(l.point::geometry) AS lat,
			ST_X(l.point::geometry) AS lng,
			l.speed,
			l.battery_level,
			l.accuracy,
			l.recorded_at
		FROM locations l
		JOIN circle_members cm ON cm.user_id = l.user_id
		WHERE cm.circle_id = $1
		ORDER BY l.user_id, l.recorded_at DESC
	`, circleID)
	if err != nil {
		return nil, fmt.Errorf("get latest locations: %w", err)
	}
	defer rows.Close()

	var locs []model.Location
	for rows.Next() {
		var l model.Location
		if err := rows.Scan(&l.ID, &l.UserID, &l.Lat, &l.Lng, &l.Speed, &l.BatteryLevel, &l.Accuracy, &l.RecordedAt); err != nil {
			return nil, fmt.Errorf("scan location: %w", err)
		}
		locs = append(locs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate locations: %w", err)
	}
	return locs, nil
}

// GetHistory returns locations for a single user within a time range, ordered by recorded_at ASC.
func (s *Store) GetHistory(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]model.Location, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			id,
			user_id,
			ST_Y(point::geometry) AS lat,
			ST_X(point::geometry) AS lng,
			speed,
			battery_level,
			accuracy,
			recorded_at
		FROM locations
		WHERE user_id = $1
		  AND recorded_at >= $2
		  AND recorded_at <= $3
		ORDER BY recorded_at ASC
	`, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("get history: %w", err)
	}
	defer rows.Close()

	var locs []model.Location
	for rows.Next() {
		var l model.Location
		if err := rows.Scan(&l.ID, &l.UserID, &l.Lat, &l.Lng, &l.Speed, &l.BatteryLevel, &l.Accuracy, &l.RecordedAt); err != nil {
			return nil, fmt.Errorf("scan history location: %w", err)
		}
		locs = append(locs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate history: %w", err)
	}
	return locs, nil
}

// DeleteLocationsOlderThan deletes rows older than N days. Returns count of deleted rows.
func (s *Store) DeleteLocationsOlderThan(ctx context.Context, days int) (int64, error) {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM locations WHERE recorded_at < now() - ($1 || ' days')::interval`,
		days,
	)
	if err != nil {
		return 0, fmt.Errorf("delete old locations: %w", err)
	}
	return tag.RowsAffected(), nil
}
