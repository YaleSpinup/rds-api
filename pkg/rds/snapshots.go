package rds

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gobuffalo/buffalo"
)

func (r *Client) ModifyDBSnapshot(c buffalo.Context, snap, engineversion string) (*rds.DBSnapshot, error) {
	SnapshotUpgradeOutput, err := r.Service.ModifyDBSnapshotWithContext(c, &rds.ModifyDBSnapshotInput{
		DBSnapshotIdentifier: aws.String(snap),
		EngineVersion:        aws.String(engineversion),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == rds.ErrCodeDBClusterSnapshotNotFoundFault {
				return nil, nil
			}
			return nil, c.Error(400, aerr)
		}
		return nil, err
	}

	return SnapshotUpgradeOutput.DBSnapshot, nil
}
