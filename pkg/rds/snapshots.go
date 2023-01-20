package rds

import (
	"fmt"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gobuffalo/buffalo"
)

func (r *Client) DescribeDBClusterSnaphot(ctx buffalo.Context, snapshotId string) (*rds.DBClusterSnapshot, error) {
	clusterSnapshotsOutput, err := r.Service.DescribeDBClusterSnapshotsWithContext(ctx, &rds.DescribeDBClusterSnapshotsInput{
		DBClusterSnapshotIdentifier: aws.String(snapshotId),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == rds.ErrCodeDBClusterSnapshotNotFoundFault {
				msg := fmt.Sprintf("cluster with snapshot id %s not found", snapshotId)
				return nil, apierror.New(apierror.ErrNotFound, msg, err)
			}
		}
		return nil, err
	}
	if len(clusterSnapshotsOutput.DBClusterSnapshots) != 1 {
		msg := fmt.Sprintf("expected 1 snapshot but found %d, snapshot id: %s", len(clusterSnapshotsOutput.DBClusterSnapshots), snapshotId)
		return nil, apierror.New(apierror.ErrInternalError, msg, err)
	}
	return clusterSnapshotsOutput.DBClusterSnapshots[0], nil
}

func (r *Client) DescribeDBSnaphot(ctx buffalo.Context, snapshotId string) (*rds.DBSnapshot, error) {
	instanceSnapshotsOutput, err := r.Service.DescribeDBSnapshotsWithContext(ctx, &rds.DescribeDBSnapshotsInput{
		DBSnapshotIdentifier: aws.String(snapshotId),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != rds.ErrCodeDBSnapshotNotFoundFault {
				msg := fmt.Sprintf("instance with snapshot id %s not found", snapshotId)
				return nil, apierror.New(apierror.ErrNotFound, msg, err)
			}
		}
	}
	if len(instanceSnapshotsOutput.DBSnapshots) != 1 {
		msg := fmt.Sprintf("expected 1 snapshot but found %d, snapshot id: %s", len(instanceSnapshotsOutput.DBSnapshots), snapshotId)
		return nil, apierror.New(apierror.ErrInternalError, msg, err)
	}
	return instanceSnapshotsOutput.DBSnapshots[0], nil
}

type SnapshotInfo struct {
	Engine, EngineVersion string
}

func (r *Client) GetSnapshotInfo(c buffalo.Context, snapshotId string) (*SnapshotInfo, error) {
	clusterSnapshot, err := r.DescribeDBClusterSnaphot(c, snapshotId)
	if err != nil {
		instanceSnapshot, err := r.DescribeDBSnaphot(c, snapshotId)
		if err != nil {
			return nil, err
		}
		return &SnapshotInfo{
			Engine:        aws.StringValue(instanceSnapshot.Engine),
			EngineVersion: aws.StringValue(instanceSnapshot.EngineVersion)}, nil
	}
	return &SnapshotInfo{
		Engine:        aws.StringValue(clusterSnapshot.Engine),
		EngineVersion: aws.StringValue(clusterSnapshot.EngineVersion)}, nil

}

func (r *Client) DescribeDBEngineVersions(ctx buffalo.Context, engine, engineVersion string) ([]*rds.DBEngineVersion, error) {
	dbEngineVersions, err := r.Service.DescribeDBEngineVersionsWithContext(ctx, &rds.DescribeDBEngineVersionsInput{
		Engine:        aws.String(engine),
		EngineVersion: aws.String(engineVersion),
	})
	if err != nil {
		return nil, err
	}
	return dbEngineVersions.DBEngineVersions, nil
}
