package google

import (
	"errors"

	"github.com/bruli-lab/myCalendar/internal/domain/auth"
	"golang.org/x/oauth2"
)

const invalidGrant = "invalid_grant"

func asRefreshError(err error) error {
	var retrieveErr *oauth2.RetrieveError
	if errors.As(err, &retrieveErr) && retrieveErr.ErrorCode == invalidGrant {
		return auth.NewRefreshError("refresh token revoked or expired: " + retrieveErr.ErrorDescription)
	}
	return err
}
