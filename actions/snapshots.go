package actions

import (
	"fmt"
	"log"
	"strconv"

	"github.com/YaleSpinup/apierror"
	rdsapi "github.com/YaleSpinup/rds-api/pkg/rds"
	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gobuffalo/buffalo"
)

// SnapshotsPost creates a manual snapshot for a given database instance or cluster
func (s *server) SnapshotsPost(c buffalo.Context) error {
	req := SnapshotCreateRequest{}
	if err := c.Bind(&req); err != nil {
		log.Println(err)
		return c.Error(400, err)
	}

	if req.SnapshotIdentifier == "" {
		return c.Error(400, errors.New("Bad request: specify SnapshotIdentifier in request"))
	}

	accountId := s.mapAccountNumber(c.Param("account"))

	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, s.session.RoleName)
	policy, err := generatePolicy("rds:CreateDBSnapshot", "rds:CreateDBClusterSnapshot")
	if err != nil {
		return handleError(c, err)
	}
	session, err := s.assumeRole(
		c,
		s.session.ExternalID,
		role,
		policy,
		"arn:aws:iam::aws:policy/AmazonRDSReadOnlyAccess",
	)
	if err != nil {
		msg := fmt.Sprintf("failed to assume role in account: %s", accountId)
		return handleError(c, apierror.New(apierror.ErrForbidden, msg, err))
	}

	rdsClient := rdsapi.NewSession(session.Session, s.defaultConfig)

	orch := &rdsOrchestrator{
		client: rdsClient,
	}

	log.Printf("creating snapshot for %s", c.Param("db"))

	output := struct {
		DBClusterSnapshot *rds.DBClusterSnapshot `json:"DBClusterSnapshot,omitempty"`
		DBSnapshot        *rds.DBSnapshot        `json:"DBSnapshot,omitempty"`
	}{}

	clusterSnapshot, err := orch.clusterSnapshotCreate(c, c.Param("db"), req.SnapshotIdentifier)
	if err != nil {
		return err
	}
	output.DBClusterSnapshot = clusterSnapshot

	if clusterSnapshot == nil {
		// this is not a cluster database, just try to back up the instance
		instanceSnapshot, err := orch.instanceSnapshotCreate(c, c.Param("db"), req.SnapshotIdentifier)
		if err != nil {
			return err
		}
		output.DBSnapshot = instanceSnapshot
	}

	if output.DBClusterSnapshot == nil && output.DBSnapshot == nil {
		return c.Error(404, errors.New("Database not found"))
	}

	return c.Render(200, r.JSON(output))
}

// SnapshotsList gets a list of snapshots for a given database instance or cluster
func (s *server) SnapshotsList(c buffalo.Context) error {
	accountId := s.mapAccountNumber(c.Param("account"))

	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, s.session.RoleName)
	policy, err := generatePolicy("rds:DescribeDBClusterSnapshots", "rds:DescribeDBSnapshots")
	if err != nil {
		return handleError(c, err)
	}
	session, err := s.assumeRole(
		c,
		s.session.ExternalID,
		role,
		policy,
		"arn:aws:iam::aws:policy/AmazonRDSReadOnlyAccess",
	)
	if err != nil {
		msg := fmt.Sprintf("failed to assume role in account: %s", accountId)
		return handleError(c, apierror.New(apierror.ErrForbidden, msg, err))
	}

	rdsClient := rdsapi.NewSession(session.Session, s.defaultConfig)

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
func (s *server) SnapshotsGet(c buffalo.Context) error {
	accountId := s.mapAccountNumber(c.Param("account"))

	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, s.session.RoleName)
	policy, err := generatePolicy("rds:DescribeDBClusterSnapshots", "rds:DescribeDBSnapshots")
	if err != nil {
		return handleError(c, err)
	}
	session, err := s.assumeRole(
		c,
		s.session.ExternalID,
		role,
		policy,
		"arn:aws:iam::aws:policy/AmazonRDSReadOnlyAccess",
	)
	if err != nil {
		msg := fmt.Sprintf("failed to assume role in account: %s", accountId)
		return handleError(c, apierror.New(apierror.ErrForbidden, msg, err))
	}

	rdsClient := rdsapi.NewSession(session.Session, s.defaultConfig)

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
func (s *server) SnapshotsDelete(c buffalo.Context) error {
	accountId := s.mapAccountNumber(c.Param("account"))

	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, s.session.RoleName)
	policy, err := generatePolicy("rds:DeleteDBClusterSnapshot", "rds:DeleteDBSnapshot")
	if err != nil {
		return handleError(c, err)
	}
	session, err := s.assumeRole(
		c,
		s.session.ExternalID,
		role,
		policy,
		"arn:aws:iam::aws:policy/AmazonRDSReadOnlyAccess",
	)
	if err != nil {
		msg := fmt.Sprintf("failed to assume role in account: %s", accountId)
		return handleError(c, apierror.New(apierror.ErrForbidden, msg, err))
	}

	rdsClient := rdsapi.NewSession(session.Session, s.defaultConfig)

	orch := &rdsOrchestrator{
		client: rdsClient,
	}

	log.Printf("deleting snapshot %s", c.Param("snap"))

	output := struct {
		DBClusterSnapshot *rds.DBClusterSnapshot `json:"DBClusterSnapshot,omitempty"`
		DBSnapshot        *rds.DBSnapshot        `json:"DBSnapshot,omitempty"`
	}{}

	clusterSnapshot, err := orch.clusterSnapshotDelete(c, c.Param("snap"))
	if err != nil {
		return err
	}
	output.DBClusterSnapshot = clusterSnapshot

	if clusterSnapshot == nil {
		// this is not a cluster database, just try to back up the instance
		instanceSnapshot, err := orch.instanceSnapshotDelete(c, c.Param("snap"))
		if err != nil {
			return err
		}
		output.DBSnapshot = instanceSnapshot
	}

	if output.DBClusterSnapshot == nil && output.DBSnapshot == nil {
		return c.Error(404, errors.New("Snapshot not found"))
	}

	return c.Render(200, r.JSON(output))
}

func (s *server) SnapshotModify(c buffalo.Context) error {
	req := SnapshotModifyRequest{}
	if err := c.Bind(&req); err != nil {
		return c.Error(400, err)
	}

	accountId := s.mapAccountNumber(c.Param("account"))

	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, s.session.RoleName)
	policy, err := generatePolicy("rds:ModifyDBSnapshot")
	if err != nil {
		return handleError(c, err)
	}
	session, err := s.assumeRole(
		c,
		s.session.ExternalID,
		role,
		policy,
		"arn:aws:iam::aws:policy/AmazonRDSReadOnlyAccess",
	)
	if err != nil {
		msg := fmt.Sprintf("failed to assume role in account: %s", accountId)
		return handleError(c, apierror.New(apierror.ErrForbidden, msg, err))
	}

	rdsClient := rdsapi.NewSession(session.Session, s.defaultConfig)

	resp, err := rdsClient.ModifyDBSnapshot(c,  c.Param("snap"), req.EngineVersion)
	if err != nil {
		return handleError(c, err)
	}

	return c.Render(200, r.JSON(resp))
}
