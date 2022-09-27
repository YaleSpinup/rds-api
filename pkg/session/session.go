package session

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	log "github.com/sirupsen/logrus"
)

// Session is a wrapper around the aws session service
type Session struct {
	Session     *session.Session
	RoleName    string
	ExternalID  string
	credentials *credentials.Credentials
	region      string
}

type SessionOption func(*Session)

// New creates a new AWS session with options
func New(opts ...SessionOption) Session {
	log.Info("creating new aws session...")

	s := Session{}

	for _, opt := range opts {
		opt(&s)
	}

	config := aws.Config{
		Credentials: s.credentials,
		Region:      aws.String(s.region),
	}

	sess := session.Must(session.NewSession(&config))
	s.Session = sess

	return s
}

func WithCredentials(key, secret, token string) SessionOption {
	return func(s *Session) {
		log.Debugf("setting credentials with key id %s", key)
		s.credentials = credentials.NewStaticCredentials(key, secret, token)
	}
}

func WithRegion(region string) SessionOption {
	return func(s *Session) {
		log.Debugf("setting region to %s", region)
		s.region = region
	}
}

func WithExternalID(extId string) SessionOption {
	return func(s *Session) {
		log.Debugf("setting external ID to %s", extId)
		s.ExternalID = extId
	}
}

func WithExternalRoleName(role string) SessionOption {
	return func(s *Session) {
		log.Debugf("setting region to %s", role)
		s.RoleName = role
	}
}
