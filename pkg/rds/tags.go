package rds

import (
	"errors"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
)

// DetermineArn returns the ARN for an RDS instance or cluster given the database name
// It could return 2 ARNs if a cluster and instance with the same name exist
func (cl Client) DetermineArn(db *string) ([]string, error) {
	arns := []string{}

	log.Println("Trying to determine ARN for", *db)

	// search clusters for the given db name
	clustersOutput, _ := cl.Service.DescribeDBClusters(&rds.DescribeDBClustersInput{
		DBClusterIdentifier: db,
	})

	// search instances for the given db name
	instancesOutput, _ := cl.Service.DescribeDBInstances(&rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: db,
	})

	if clustersOutput == nil && instancesOutput == nil {
		return nil, errors.New("Unable to determine ARN for database " + *db)
	}

	if len(clustersOutput.DBClusters) > 0 {
		for _, cluster := range clustersOutput.DBClusters {
			arns = append(arns, aws.StringValue(cluster.DBClusterArn))
		}
	}

	if len(instancesOutput.DBInstances) > 0 {
		for _, instance := range instancesOutput.DBInstances {
			arns = append(arns, aws.StringValue(instance.DBInstanceArn))
		}
	}

	if len(arns) == 0 {
		return nil, errors.New("Unable to determine ARN for database " + *db)
	}

	return arns, nil
}
