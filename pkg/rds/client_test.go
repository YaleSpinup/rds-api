package rds

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
)

// mockRDSClient is a fake ec2 client
type mockRDSClient struct {
	rdsiface.RDSAPI
	t   *testing.T
	err error
}

func newmockRDSClient(t *testing.T, err error) rdsiface.RDSAPI {
	return &mockRDSClient{
		t:   t,
		err: err,
	}
}
