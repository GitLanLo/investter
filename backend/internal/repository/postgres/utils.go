package postgres

import (
	"database/sql"
	"github.com/lib/pq"
)

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullFloat(f float64) sql.NullFloat64 {
	if f == 0 {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: f, Valid: true}
}

func stringArray(arr []string) pq.StringArray {
	if arr == nil {
		return pq.StringArray{}
	}
	return pq.StringArray(arr)
}
