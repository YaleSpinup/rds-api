package rds

import (
	"errors"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gobuffalo/buffalo"
)

// StopDatabase stops an RDS database instance or cluster.
// SQL server in Multi-AZ configuration is not supported.
// Note: this operation can take a long time and clusters/instances stopped for more than 7
// days will be automatically started so patches can be applied.
// https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/aurora-cluster-stop-start.html
// https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_StopInstance.html
func (r *Client) StopDatabase(ctx buffalo.Context, id string) error {
	if id == "" {
		return errors.New("database identifier cannot be empty")
	}

	log.Printf("Stopping database with identifier %s", id)

	if _, err := r.Service.StopDBClusterWithContext(ctx, &rds.StopDBClusterInput{
		DBClusterIdentifier: aws.String(id),
	}); err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != rds.ErrCodeDBClusterNotFoundFault {
				return err
			}
		}
	} else {
		return nil
	}

	_, err := r.Service.StopDBInstanceWithContext(ctx, &rds.StopDBInstanceInput{
		DBInstanceIdentifier: aws.String(id),
	})

	return err
}

// StartDatabase starts an RDS database instance or cluster
func (r *Client) StartDatabase(ctx buffalo.Context, id string) error {
	if id == "" {
		return errors.New("database identifier cannot be empty")
	}

	log.Printf("Starting database with identifier %s", id)

	if _, err := r.Service.StartDBClusterWithContext(ctx, &rds.StartDBClusterInput{
		DBClusterIdentifier: aws.String(id),
	}); err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != rds.ErrCodeDBClusterNotFoundFault {
				return err
			}
		}
	} else {
		return nil
	}

	_, err := r.Service.StartDBInstanceWithContext(ctx, &rds.StartDBInstanceInput{
		DBInstanceIdentifier: aws.String(id),
	})

	return err
}
