package rds

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gobuffalo/buffalo"
)

func (r *Client) ModifyDBSnapshot(c buffalo.Context, snap, engineversion string) (*rds.DBSnapshot, error) {
	snapshotUpgradeOutput, err := r.Service.ModifyDBSnapshotWithContext(c, &rds.ModifyDBSnapshotInput{
		DBSnapshotIdentifier: aws.String(snap),
		EngineVersion:        aws.String(engineversion),
	})
	if err != nil {
		return nil, err
	}
	return snapshotUpgradeOutput.DBSnapshot, nil
}
