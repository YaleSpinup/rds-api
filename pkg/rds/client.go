package rds

import (
	"github.com/YaleSpinup/rds-api/pkg/common"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
)

// Client struct contains the initialized RDS service and other RDS-related parameters
type Client struct {
	Service                            rdsiface.RDSAPI
	DefaultSubnetGroup                 string
	DefaultDBParameterGroupName        map[string]string
	DefaultDBClusterParameterGroupName map[string]string
}

// NewSession creates an AWS session for RDS and returns an RDSClient
func NewSession(sess *session.Session, c common.CommonConfig) *Client {
	return &Client{
		Service:                            rds.New(sess),
		DefaultSubnetGroup:                 c.DefaultSubnetGroup,
		DefaultDBParameterGroupName:        c.DefaultDBParameterGroupName,
		DefaultDBClusterParameterGroupName: c.DefaultDBClusterParameterGroupName,
	}
}
