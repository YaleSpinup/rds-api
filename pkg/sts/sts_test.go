package sts

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/service/sts/stsiface"
)

// mockSTSClient is a fake sts client
type mockSTSClient struct {
	stsiface.STSAPI
	t   *testing.T
	err error
}

func newMockSTSClient(t *testing.T, err error) stsiface.STSAPI {
	return &mockSTSClient{
		t:   t,
		err: err,
	}
}

func TestNewSession(t *testing.T) {
	client := New()
	to := reflect.TypeOf(client).String()
	if to != "sts.STS" {
		t.Errorf("expected type to be 'sts.STS', got %s", to)
	}
}
