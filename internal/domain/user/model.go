// internal/domain/user/model.go
package user

import (
	"database/sql"
)

// User adalah representasi data user di dalam database
type User struct {
	ID       int            `json:"id"`
	Fullname string         `json:"fullname"`
	Email    string         `json:"email"`
	Phone    string         `json:"phone"`
	Password string         `json:"-"`
	Photo    sql.NullString `json:"photo"`
}

// SignUpRequest adalah data yang kita harapkan dari request API
type SignUpRequest struct {
	Fullname string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}
