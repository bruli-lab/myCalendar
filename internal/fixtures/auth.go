package fixtures

import (
	"time"

	"github.com/bruli-lab/myCalendar/internal/domain/auth"
)

type AuthBuilder struct {
	AccessToken  *string
	RefreshToken *string
	Expiry       *time.Time
	TokenType    *string
}

func (b AuthBuilder) Build() *auth.Auth {
	a, err := auth.New(
		setData("access-token", b.AccessToken),
		setData("refresh-token", b.RefreshToken),
		setData(time.Date(2026, time.November, 13, 9, 0, 0, 0, time.UTC), b.Expiry),
		setData("Bearer", b.TokenType),
	)
	if err != nil {
		panic(err)
	}
	return a
}
