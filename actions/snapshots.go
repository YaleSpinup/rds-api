package actions

import (
	"log"
	"strconv"

	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gobuffalo/buffalo"
)

// SnapshotsPost creates a manual snapshot for a given database instance or cluster
func SnapshotsPost(c buffalo.Context) error {
	req := SnapshotCreateRequest{}
	if err := c.Bind(&req); err != nil {
		log.Println(err)
		return c.Error(400, err)
	}

	if req.SnapshotIdentifier == "" {
		return c.Error(400, errors.New("Bad request: specify SnapshotIdentifier in request"))
	}

	rdsClient, ok := RDS[c.Param("account")]
	if !ok {
		return c.Error(400, errors.New("Bad request: unknown account "+c.Param("account")))
	}

	log.Printf("creating snapshot for %s", c.Param("db"))

	output := struct {
		DBClusterSnapshot *rds.DBClusterSnapshot `json:"DBClusterSnapshot,omitempty"`
		DBSnapshot        *rds.DBSnapshot        `json:"DBSnapshot,omitempty"`
	}{}

	clusterSnapshotOutput, err := rdsClient.Service.CreateDBClusterSnapshotWithContext(c, &rds.CreateDBClusterSnapshotInput{
		DBClusterIdentifier:         aws.String(c.Param("db")),
		DBClusterSnapshotIdentifier: aws.String(req.SnapshotIdentifier),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != rds.ErrCodeDBClusterNotFoundFault {
				return c.Error(400, aerr)
			}
		}
	} else {
		output.DBClusterSnapshot = clusterSnapshotOutput.DBClusterSnapshot
	}

	if output.DBClusterSnapshot == nil {
		// this is not a cluster database, just try to back up the instance
		instanceSnapshotOutput, err := rdsClient.Service.CreateDBSnapshotWithContext(c, &rds.CreateDBSnapshotInput{
			DBInstanceIdentifier: aws.String(c.Param("db")),
			DBSnapshotIdentifier: aws.String(req.SnapshotIdentifier),
		})
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				if aerr.Code() != rds.ErrCodeDBInstanceNotFoundFault {
					return c.Error(400, aerr)
				}
			}
		} else {
			output.DBSnapshot = instanceSnapshotOutput.DBSnapshot
		}
	}

	if output.DBClusterSnapshot == nil && output.DBSnapshot == nil {
		return c.Error(404, errors.New("Database not found"))
	}

	return c.Render(200, r.JSON(output))
}

// SnapshotsList gets a list of snapshots for a given database instance or cluster
func SnapshotsList(c buffalo.Context) error {
	rdsClient, ok := RDS[c.Param("account")]
	if !ok {
		return c.Error(400, errors.New("Bad request: unknown account "+c.Param("account")))
	}

	log.Printf("getting snapshots for %s", c.Param("db"))

	clusterSnapshotsOutput, err := rdsClient.Service.DescribeDBClusterSnapshotsWithContext(c, &rds.DescribeDBClusterSnapshotsInput{
		DBClusterIdentifier: aws.String(c.Param("db")),
	})
	if err != nil {
		return handleError(c, err)
	}

	instanceSnapshotsOutput, err := rdsClient.Service.DescribeDBSnapshotsWithContext(c, &rds.DescribeDBSnapshotsInput{
		DBInstanceIdentifier: aws.String(c.Param("db")),
	})
	if err != nil {
		return handleError(c, err)
	}

	if len(clusterSnapshotsOutput.DBClusterSnapshots) == 0 && len(instanceSnapshotsOutput.DBSnapshots) == 0 {
		return c.Error(404, errors.New("No snapshots found"))
	}

	var items int
	if clusterSnapshotsOutput.DBClusterSnapshots != nil {
		items = len(clusterSnapshotsOutput.DBClusterSnapshots)
	} else {
		items = len(instanceSnapshotsOutput.DBSnapshots)
	}

	output := struct {
		DBClusterSnapshots []*rds.DBClusterSnapshot `json:"DBClusterSnapshots,omitempty"`
		DBSnapshots        []*rds.DBSnapshot        `json:"DBSnapshots,omitempty"`
	}{
		clusterSnapshotsOutput.DBClusterSnapshots,
		instanceSnapshotsOutput.DBSnapshots,
	}

	c.Response().Header().Set("X-Items", strconv.Itoa(items))
	return c.Render(200, r.JSON(output))
}

// SnapshotsGet returns information about a specific database snapshot
func SnapshotsGet(c buffalo.Context) error {
	rdsClient, ok := RDS[c.Param("account")]
	if !ok {
		return c.Error(400, errors.New("Bad request: unknown account "+c.Param("account")))
	}

	log.Printf("getting information about snapshot %s", c.Param("snap"))

	var clusterSnapshot *rds.DBClusterSnapshot
	var instanceSnapshot *rds.DBSnapshot

	clusterSnapshotsOutput, err := rdsClient.Service.DescribeDBClusterSnapshotsWithContext(c, &rds.DescribeDBClusterSnapshotsInput{
		DBClusterSnapshotIdentifier: aws.String(c.Param("snap")),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != rds.ErrCodeDBClusterSnapshotNotFoundFault {
				return c.Error(400, aerr)
			}
		}
	} else if len(clusterSnapshotsOutput.DBClusterSnapshots) > 1 {
		return c.Error(500, errors.New("Unexpected number of snapshots"))
	} else {
		clusterSnapshot = clusterSnapshotsOutput.DBClusterSnapshots[0]
	}

	instanceSnapshotsOutput, err := rdsClient.Service.DescribeDBSnapshotsWithContext(c, &rds.DescribeDBSnapshotsInput{
		DBSnapshotIdentifier: aws.String(c.Param("snap")),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != rds.ErrCodeDBSnapshotNotFoundFault {
				return c.Error(400, aerr)
			}
		}
	} else if len(instanceSnapshotsOutput.DBSnapshots) > 1 {
		return c.Error(500, errors.New("Unexpected number of snapshots"))
	} else {
		instanceSnapshot = instanceSnapshotsOutput.DBSnapshots[0]
	}

	if clusterSnapshot == nil && instanceSnapshot == nil {
		return c.Error(404, errors.New("Snapshot not found"))
	}

	output := struct {
		DBClusterSnapshot *rds.DBClusterSnapshot `json:"DBClusterSnapshot,omitempty"`
		DBSnapshot        *rds.DBSnapshot        `json:"DBSnapshot,omitempty"`
	}{
		clusterSnapshot,
		instanceSnapshot,
	}

	return c.Render(200, r.JSON(output))
}

// SnapshotsDelete deletes a specific database snapshot
func SnapshotsDelete(c buffalo.Context) error {
	rdsClient, ok := RDS[c.Param("account")]
	if !ok {
		return c.Error(400, errors.New("Bad request: unknown account "+c.Param("account")))
	}

	log.Printf("deleting snapshot %s", c.Param("snap"))

	output := struct {
		DBClusterSnapshot *rds.DBClusterSnapshot `json:"DBClusterSnapshot,omitempty"`
		DBSnapshot        *rds.DBSnapshot        `json:"DBSnapshot,omitempty"`
	}{}

	clusterSnapshotOutput, err := rdsClient.Service.DeleteDBClusterSnapshotWithContext(c, &rds.DeleteDBClusterSnapshotInput{
		DBClusterSnapshotIdentifier: aws.String(c.Param("snap")),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != rds.ErrCodeDBClusterSnapshotNotFoundFault {
				return c.Error(400, aerr)
			}
		}
	} else {
		output.DBClusterSnapshot = clusterSnapshotOutput.DBClusterSnapshot
	}

	if output.DBClusterSnapshot == nil {
		instanceSnapshotOutput, err := rdsClient.Service.DeleteDBSnapshotWithContext(c, &rds.DeleteDBSnapshotInput{
			DBSnapshotIdentifier: aws.String(c.Param("snap")),
		})
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				if aerr.Code() != rds.ErrCodeDBSnapshotNotFoundFault {
					return c.Error(400, aerr)
				}
			}
		} else {
			output.DBSnapshot = instanceSnapshotOutput.DBSnapshot
		}
	}

	if output.DBClusterSnapshot == nil && output.DBSnapshot == nil {
		return c.Error(404, errors.New("Snapshot not found"))
	}

	return c.Render(200, r.JSON(output))
}
