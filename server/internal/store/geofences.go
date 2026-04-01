package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/model"
)

func (s *Store) CreateGeofence(ctx context.Context, circleID uuid.UUID, name string, lat, lng float64, radiusMeters float32, createdBy uuid.UUID) (*model.Geofence, error) {
	var g model.Geofence
	err := s.pool.QueryRow(ctx,
		`INSERT INTO geofences (circle_id, name, center, radius_meters, created_by)
		 VALUES ($1, $2, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, $5, $6)
		 RETURNING id, circle_id, name, ST_Y(center::geometry), ST_X(center::geometry), radius_meters, created_by, created_at`,
		circleID, name, lng, lat, radiusMeters, createdBy,
	).Scan(&g.ID, &g.CircleID, &g.Name, &g.Lat, &g.Lng, &g.RadiusMeters, &g.CreatedBy, &g.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create geofence: %w", err)
	}
	return &g, nil
}

func (s *Store) GetGeofences(ctx context.Context, circleID uuid.UUID) ([]model.Geofence, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, circle_id, name, ST_Y(center::geometry), ST_X(center::geometry), radius_meters, created_by, created_at
		 FROM geofences WHERE circle_id = $1`,
		circleID,
	)
	if err != nil {
		return nil, fmt.Errorf("get geofences: %w", err)
	}
	defer rows.Close()

	var geofences []model.Geofence
	for rows.Next() {
		var g model.Geofence
		if err := rows.Scan(&g.ID, &g.CircleID, &g.Name, &g.Lat, &g.Lng, &g.RadiusMeters, &g.CreatedBy, &g.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan geofence: %w", err)
		}
		geofences = append(geofences, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate geofences: %w", err)
	}
	return geofences, nil
}

func (s *Store) UpdateGeofence(ctx context.Context, id uuid.UUID, name string, lat, lng float64, radiusMeters float32) (*model.Geofence, error) {
	var g model.Geofence
	err := s.pool.QueryRow(ctx,
		`UPDATE geofences
		 SET name = $2, center = ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, radius_meters = $5
		 WHERE id = $1
		 RETURNING id, circle_id, name, ST_Y(center::geometry), ST_X(center::geometry), radius_meters, created_by, created_at`,
		id, name, lng, lat, radiusMeters,
	).Scan(&g.ID, &g.CircleID, &g.Name, &g.Lat, &g.Lng, &g.RadiusMeters, &g.CreatedBy, &g.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("update geofence: %w", err)
	}
	return &g, nil
}

func (s *Store) DeleteGeofence(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM geofences WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete geofence: %w", err)
	}
	return nil
}

func (s *Store) FindContainingGeofences(ctx context.Context, circleID uuid.UUID, lat, lng float64) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id FROM geofences
		 WHERE circle_id = $1
		   AND ST_DWithin(center, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, radius_meters)`,
		circleID, lng, lat,
	)
	if err != nil {
		return nil, fmt.Errorf("find containing geofences: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan geofence id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate geofence ids: %w", err)
	}
	return ids, nil
}
