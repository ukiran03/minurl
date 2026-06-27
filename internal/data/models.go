package data

import (
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

var PgxReqCtxTTL = 3 * time.Second // pgx query request context lifetime

type Models struct {
	MinUrls MinUrlModel
}

func NewModels(db *pgxpool.Pool, sfnid int) Models {
	return Models{
		MinUrls: MinUrlModel{
			SFNID: int64(sfnid),
			DB:    db,
			TTL:   PgxReqCtxTTL,
		},
	}
}
