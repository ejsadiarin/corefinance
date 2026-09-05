package helper

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

func ToPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func StrVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ToPgUUID(id *string) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{Valid: false}
	}
	parsed, err := uuid.Parse(*id)
	if err != nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}
}

func ToUUID(id *string) uuid.UUID {
	if id == nil {
		return uuid.Nil
	}
	parsed, err := uuid.Parse(*id)
	if err != nil {
		return uuid.Nil
	}
	return parsed
}

func ToPgDate(date string) pgtype.Date {
	if date == "" {
		return pgtype.Date{Valid: false}
	}
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return pgtype.Date{Valid: false}
	}
	return pgtype.Date{Time: t, Valid: true}
}

func ToPgDatePtr(date *string) pgtype.Date {
	if date == nil {
		return pgtype.Date{Valid: false}
	}
	return ToPgDate(*date)
}

func ToPgUUIDFromUUID(id uuid.UUID) pgtype.UUID {
	if id == uuid.Nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: id, Valid: true}
}

func ToDecimal(v any) decimal.Decimal {
	if v == nil {
		return decimal.Zero
	}
	switch val := v.(type) {
	case decimal.Decimal:
		return val
	case float64:
		return decimal.NewFromFloat(val)
	case int64:
		return decimal.NewFromInt(val)
	case int:
		return decimal.NewFromInt(int64(val))
	default:
		return decimal.Zero
	}
}
