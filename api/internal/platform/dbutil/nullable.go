package dbutil

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// NullableUUID 把 UUID 转为可空数据库类型，空值表示未设置。
func NullableUUID(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: value, Valid: value != uuid.Nil}
}

// OptionalUUID 把可空数据库类型还原为 UUID，空值返回 uuid.Nil。
func OptionalUUID(value pgtype.UUID) uuid.UUID {
	if !value.Valid {
		return uuid.Nil
	}
	return uuid.UUID(value.Bytes)
}

// NullableInt64 把整数转为可空数据库类型。
func NullableInt64(value int64) pgtype.Int8 {
	return pgtype.Int8{Int64: value, Valid: true}
}
