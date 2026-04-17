package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PlanTier int

const (
	FreePlan PlanTier = iota
	ProPlan
	EnterprisePlan
)

type User struct {
	ID                uuid.UUID `db:"id"`
	Email             string    `db:"email"`
	Username          string    `db:"username"`
	PasswordHash      []byte    `db:"password_hash"`
	FullName          string    `db:"full_name"`
	AvatarURL         string    `db:"avatar_url"`
	IsActive          bool      `db:"is_active"`
	IsVerified        bool      `db:"is_verified"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
	LastLoginAt       time.Time `db:"last_login_at"`
	MonthlyLinkLimit  int64     `db:"monthly_link_limit"`
	CurrentMonthLinks int64     `db:"current_month_links"`
	PlanTier          PlanTier  `db:"plan_tier"`
}

type UserModel struct {
	Pool *pgxpool.Pool
	Ctx  context.Context
}

func (m *UserModel) GetByID(id int64) (*User, error) {
	stmt := `SELECT id, username, email, password_hash, created_at
             FROM users WHERE id = $1`

	rows, err := m.Pool.Query(m.Ctx, stmt, id)
	if err != nil {
		return nil, err
	}

	// CollectOneRow is the pgx version of sqlx's Get()
	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[User])
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (m *UserModel) GetAll(ctx context.Context) ([]User, error) {
	query := `SELECT id, username, email, password_hash, created_at
              FROM users`

	rows, err := m.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	// CollectRows is the pgx version of sqlx's Select()
	return pgx.CollectRows(rows, pgx.RowToStructByName[User])
}
