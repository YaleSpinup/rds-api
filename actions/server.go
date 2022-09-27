package actions

import (
	"fmt"
	"time"

	"github.com/YaleSpinup/apierror"
	"github.com/YaleSpinup/rds-api/pkg/common"
	"github.com/YaleSpinup/rds-api/pkg/session"
	"github.com/patrickmn/go-cache"
)

type server struct {
	accounts     map[string]common.RdsAccount
	org          string
	session      *session.Session
	sessionCache *cache.Cache
}

func newServer(config common.Config) *server {
	sess := session.New(
		session.WithCredentials(config.Account.Akid, config.Account.Secret, ""),
		session.WithRegion(config.Account.Region),
		session.WithExternalID(config.Account.ExternalID),
		session.WithExternalRoleName(config.Account.Role),
	)

	return &server{
		accounts:     config.Accounts,
		org:          config.Org,
		session:      &sess,
		sessionCache: cache.New(600*time.Second, 900*time.Second),
	}
}

// if we have an entry for the account name, return the associated account number
func (s *server) mapAccountNumber(name string) (*common.RdsAccount, error) {
	if a, ok := s.accounts[name]; ok {
		return &a, nil
	}
	return nil, apierror.New(apierror.ErrBadRequest, fmt.Sprintf("unknown account %s", name), nil)
}
