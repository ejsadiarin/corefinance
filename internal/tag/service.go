package tag

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
)

type Service struct {
	queries *db.Queries
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		queries: db.New(pool),
	}
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (db.Tag, error) {
	return s.queries.CreateTag(ctx, db.CreateTagParams{
		UserID: userID,
		Name:   req.Name,
		Color:  toPgText(req.Color),
	})
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]db.Tag, error) {
	return s.queries.ListTags(ctx, userID)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID, userID uuid.UUID) (db.Tag, error) {
	return s.queries.GetTag(ctx, db.GetTagParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req UpdateRequest) (db.Tag, error) {
	return s.queries.UpdateTag(ctx, db.UpdateTagParams{
		ID:     id,
		UserID: userID,
		Name:   req.Name,
		Color:  toPgText(req.Color),
	})
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.queries.DeleteTag(ctx, db.DeleteTagParams{
		ID:     id,
		UserID: userID,
	})
}

func toPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}
