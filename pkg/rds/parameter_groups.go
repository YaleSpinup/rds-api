package rds

import (
	"errors"

	"github.com/aws/aws-sdk-go/service/rds"
)

// DetermineParameterGroupFamily returns the DBParameterGroupFamily based on the
// given database Engine and EngineVersion
// e.g. given engine "postgres" and engineVersion "10.5" it will return "postgres10"
func (cl Client) DetermineParameterGroupFamily(engine, engineVersion *string) (string, error) {
	input := &rds.DescribeDBEngineVersionsInput{
		Engine:        engine,
		EngineVersion: engineVersion,
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
