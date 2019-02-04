package rds

import (
	"errors"

	"github.com/aws/aws-sdk-go/service/rds"
)

// DetermineParameterGroupFamily returns the DBParameterGroupFamily based on the
// given database Engine and EngineVersion
func (cl Client) DetermineParameterGroupFamily(e, ev *string) (string, error) {
	input := &rds.DescribeDBEngineVersionsInput{
		Engine:        e,
		EngineVersion: ev,
	}

	evResult, err := cl.Service.DescribeDBEngineVersions(input)
	if err != nil {
		return "", err
	}
	if len(evResult.DBEngineVersions) == 0 {
		return "", errors.New("Unable to find any matching database engine/version")
	}

	return *evResult.DBEngineVersions[0].DBParameterGroupFamily, nil
}
