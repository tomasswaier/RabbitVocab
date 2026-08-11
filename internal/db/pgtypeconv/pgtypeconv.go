package pgtypeconv

import "github.com/jackc/pgx/v5/pgtype"

func TextFromPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func PtrFromText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}
