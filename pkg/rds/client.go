package rds

import (
	"log"

	"github.com/YaleSpinup/rds-api/pkg/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
)

// Client struct contains the initialized RDS service and other RDS-related parameters
type Client struct {
	Service            rdsiface.RDSAPI
	DefaultSubnetGroup string
}

// NewSession creates an AWS session for RDS and returns an RDSClient
func NewSession(c common.Account) Client {
	log.Printf("Creating new session with key id %s in region %s", c.Akid, c.Region)
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(c.Akid, c.Secret, ""),
		Region:      aws.String(c.Region),
	}))

	return Client{
		Service:            rds.New(sess),
		DefaultSubnetGroup: c.DefaultSubnetGroup,
	}
}
