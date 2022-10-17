package actions

import (
	"time"

	"github.com/YaleSpinup/rds-api/pkg/common"
	"github.com/YaleSpinup/rds-api/pkg/session"
	"github.com/patrickmn/go-cache"
)

type server struct {
	accountsMap   map[string]string
	defaultConfig common.CommonConfig
	org           string
	session       *session.Session
	sessionCache  *cache.Cache
}

func newServer(config common.Config) *server {
	sess := session.New(
		session.WithCredentials(config.Account.Akid, config.Account.Secret, ""),
		session.WithRegion(config.Account.Region),
		session.WithExternalID(config.Account.ExternalID),
		session.WithExternalRoleName(config.Account.Role),
	)
	return &server{
		accountsMap:   config.AccountsMap,
		defaultConfig: config.DefaultConfig,
		org:           config.Org,
		session:       &sess,
		sessionCache:  cache.New(600*time.Second, 900*time.Second),
	}
}

// if we have an entry for the account name, return the associated account number
func (s *server) mapAccountNumber(name string) string {
	if a, ok := s.accountsMap[name]; ok {
		return a
	}
	return name
}
