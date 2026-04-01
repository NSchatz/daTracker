package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	DisplayName  string    `json:"display_name"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Circle struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	InviteCode string    `json:"invite_code"`
	CreatedBy  uuid.UUID `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
}

type CircleMember struct {
	CircleID    uuid.UUID `json:"circle_id"`
	UserID      uuid.UUID `json:"user_id"`
	Role        string    `json:"role"`
	JoinedAt    time.Time `json:"joined_at"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
}

type Location struct {
	ID           int64     `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	Lat          float64   `json:"lat"`
	Lng          float64   `json:"lng"`
	Speed        *float32  `json:"speed,omitempty"`
	BatteryLevel *int16    `json:"battery_level,omitempty"`
	Accuracy     *float32  `json:"accuracy,omitempty"`
	RecordedAt   time.Time `json:"recorded_at"`
}

type LocationInput struct {
	Lat          float64   `json:"lat"`
	Lng          float64   `json:"lng"`
	Speed        *float32  `json:"speed,omitempty"`
	BatteryLevel *int16    `json:"battery_level,omitempty"`
	Accuracy     *float32  `json:"accuracy,omitempty"`
	RecordedAt   time.Time `json:"recorded_at"`
}

type Geofence struct {
	ID           uuid.UUID `json:"id"`
	CircleID     uuid.UUID `json:"circle_id"`
	Name         string    `json:"name"`
	Lat          float64   `json:"lat"`
	Lng          float64   `json:"lng"`
	RadiusMeters float32   `json:"radius_meters"`
	CreatedBy    uuid.UUID `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
}
