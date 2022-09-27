package sts

import (
	"context"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	log "github.com/sirupsen/logrus"
)

type STS struct {
	DefaultDuration int64
	session         *session.Session
	Service         stsiface.STSAPI
	Org             string
}

type STSOption func(*STS)

func New(opts ...STSOption) STS {
	s := STS{
		DefaultDuration: 900,
	}

	for _, opt := range opts {
		opt(&s)
	}

	if s.session != nil {
		s.Service = sts.New(s.session)
	}

	return s
}

func WithSession(sess *session.Session) STSOption {
	return func(s *STS) {
		log.Debug("using aws session")
		s.session = sess
	}
}

func WithDefaultSessionDuration(t int64) STSOption {
	return func(s *STS) {
		log.Debugf("setting default session duration to %d", t)
		s.DefaultDuration = t
	}
}

// AssumeRole assumes the passed role with the given input
// NB: the combined size of the inlinePolicy and the policy within the policyArns passed is 2048 characters.
func (s *STS) AssumeRole(ctx context.Context, input *sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error) {
	if input.RoleArn == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("assuming role '%s' with session name '%s'", aws.StringValue(input.RoleArn), aws.StringValue(input.RoleSessionName))

	log.Debugf("assuming role %s with input %+v", aws.StringValue(input.RoleArn), input)

	out, err := s.Service.AssumeRoleWithContext(ctx, input)
	if err != nil {
		return nil, err
	}

	log.Debugf("got output from sts assume role (%s): %+v", aws.StringValue(input.RoleArn), out)

	return out, nil
}
