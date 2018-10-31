package actions

import (
	"errors"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gobuffalo/buffalo"
)

// DatabaseCreateInput is the input for creating a new database
// The Instance part is required and defines the database instance properties
// The Cluster is optional if the created database instance belongs to a new cluster
type DatabaseCreateInput struct {
	// https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#CreateDBClusterInput
	Cluster *rds.CreateDBClusterInput
	// https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#CreateDBInstanceInput
	Instance *rds.CreateDBInstanceInput
}

// DatabasesList gets a list of databases for a given account
// If the `all=true` parameter is passed it will return a list of clusters in addition to instances.
func DatabasesList(c buffalo.Context) error {
	// if all param is given, we'll return information about both instances and clusters
	// otherwise, only database instances will be returned
	all := false
	if b, err := strconv.ParseBool(c.Param("all")); err == nil {
		all = b
	}

	rdsClient, ok := RDS[c.Param("account")]
	if !ok {
		return c.Error(400, errors.New("Bad request: unknown account "+c.Param("account")))
	}

	var clustersOutput *rds.DescribeDBClustersOutput
	var instancesOutput *rds.DescribeDBInstancesOutput
	var err error

	if all {
		if clustersOutput, err = rdsClient.Service.DescribeDBClustersWithContext(c, &rds.DescribeDBClustersInput{}); err != nil {
			log.Println(err.Error())
		}
	}

	instancesOutput, err = rdsClient.Service.DescribeDBInstancesWithContext(c, &rds.DescribeDBInstancesInput{})
	if err != nil {
		log.Println(err.Error())
		if aerr, ok := err.(awserr.Error); ok {
			return c.Error(400, aerr)
		}
		return err
	}

	output := struct {
		*rds.DescribeDBClustersOutput
		*rds.DescribeDBInstancesOutput
	}{
		clustersOutput,
		instancesOutput,
	}
	return c.Render(200, r.JSON(output))
}

// DatabasesGet gets details about a specific database
// If the `all=true` parameter is passed it will return a list of clusters in addition to instances.
func DatabasesGet(c buffalo.Context) error {
	// if all param is given, we'll return information about both instances and clusters
	// otherwise, only database instances will be searched
	all := false
	if b, err := strconv.ParseBool(c.Param("all")); err == nil {
		all = b
	}

	rdsClient, ok := RDS[c.Param("account")]
	if !ok {
		return c.Error(400, errors.New("Bad request: unknown account "+c.Param("account")))
	}

	var clustersOutput *rds.DescribeDBClustersOutput
	var instancesOutput *rds.DescribeDBInstancesOutput
	var err error
	clusterNotFound := true

	if all {
		// search clusters for the given db name
		clusterNotFound = false
		clustersInput := &rds.DescribeDBClustersInput{
			DBClusterIdentifier: aws.String(c.Param("db")),
		}
		if clustersOutput, err = rdsClient.Service.DescribeDBClustersWithContext(c, clustersInput); err != nil {
			log.Println(err.Error())
			if aerr, ok := err.(awserr.Error); ok {
				if aerr.Code() == rds.ErrCodeDBClusterNotFoundFault {
					clusterNotFound = true
				}
			}
		}
	}

	// search instances for the given db name
	instancesInput := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(c.Param("db")),
	}
	if instancesOutput, err = rdsClient.Service.DescribeDBInstancesWithContext(c, instancesInput); err != nil {
		log.Println(err.Error())
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == rds.ErrCodeDBInstanceNotFoundFault {
				if clusterNotFound {
					return c.Error(404, aerr)
				}
			} else {
				return c.Error(400, aerr)
			}
		}
		if clusterNotFound {
			return err
		}
	}

	output := struct {
		*rds.DescribeDBClustersOutput
		*rds.DescribeDBInstancesOutput
	}{
		clustersOutput,
		instancesOutput,
	}
	return c.Render(200, r.JSON(output))
}

// DatabasesPost creates a database in a given account
// It will create a database instance as specified by the `Instance` hash parameters.
// If a `Cluster` hash is also given, it will first create an RDS cluster and the instance next.
func DatabasesPost(c buffalo.Context) error {
	input := DatabaseCreateInput{}
	if err := c.Bind(&input); err != nil {
		log.Println(err)
		return c.Error(400, err)
	}
	if input.Instance == nil {
		return c.Error(400, errors.New("Bad request"))
	}

	rdsClient, ok := RDS[c.Param("account")]
	if !ok {
		return c.Error(400, errors.New("Bad request: unknown account "+c.Param("account")))
	}

	var clusterOutput *rds.CreateDBClusterOutput
	var instanceOutput *rds.CreateDBInstanceOutput
	var err error

	// create rds cluster first, if specified
	if input.Cluster != nil {
		if input.Cluster.DBSubnetGroupName == nil {
			input.Cluster.DBSubnetGroupName = aws.String(rdsClient.DefaultSubnetGroup)
		}
		if clusterOutput, err = rdsClient.Service.CreateDBClusterWithContext(c, input.Cluster); err != nil {
			log.Println(err.Error())
			if aerr, ok := err.(awserr.Error); ok {
				return c.Error(400, aerr)
			}
			return err
		}
		log.Println("Created RDS cluster", clusterOutput)
	}

	// create rds instance
	if input.Instance.DBSubnetGroupName == nil {
		input.Instance.DBSubnetGroupName = aws.String(rdsClient.DefaultSubnetGroup)
	}
	if instanceOutput, err = rdsClient.Service.CreateDBInstanceWithContext(c, input.Instance); err != nil {
		log.Println(err.Error())
		if input.Cluster != nil {
			// if this instance was in a new cluster, delete the cluster
			log.Println("Deleting cluster", *input.Cluster.DBClusterIdentifier)
			clusterInput := &rds.DeleteDBClusterInput{
				DBClusterIdentifier: input.Cluster.DBClusterIdentifier,
				SkipFinalSnapshot:   aws.Bool(true),
			}
			if _, errc := rdsClient.Service.DeleteDBClusterWithContext(c, clusterInput); errc != nil {
				log.Println("Failed to delete cluster", errc.Error())
			} else {
				log.Println("Successfully requested deletion of cluster", *input.Cluster.DBClusterIdentifier)
			}
		}
		if aerr, ok := err.(awserr.Error); ok {
			return c.Error(400, aerr)
		}
		return err
	}
	log.Println("Created RDS instance", instanceOutput)

	output := struct {
		*rds.CreateDBClusterOutput
		*rds.CreateDBInstanceOutput
	}{
		clusterOutput,
		instanceOutput,
	}

	return c.Render(200, r.JSON(output))
}

// DatabasesDelete deletes a database in a given account
// It will delete the database instance with the given {db} name and will also delete the associated cluster
// if the instance belongs to a cluster and is the last remaining member.
// If the snapshot=true parameter is given, it will create a final snapshot of the instance/cluster.
func DatabasesDelete(c buffalo.Context) error {
	// if snapshot param is given, a final snapshot will be created before deleting
	snapshot := false
	if b, err := strconv.ParseBool(c.Param("snapshot")); err == nil {
		snapshot = b
	}

	rdsClient, ok := RDS[c.Param("account")]
	if !ok {
		return c.Error(400, errors.New("Bad request: unknown account "+c.Param("account")))
	}

	var clusterOutput *rds.DeleteDBClusterOutput
	var instanceOutput *rds.DeleteDBInstanceOutput
	var err error
	var clusterName *string

	// first, let's determine if the given database instance belongs to a cluster
	i, err := rdsClient.Service.DescribeDBInstancesWithContext(c, &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(c.Param("db")),
	})
	if err != nil {
		log.Println(err.Error())
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == rds.ErrCodeDBInstanceNotFoundFault {
				return c.Error(404, aerr)
			}
			return c.Error(400, aerr)
		}
		return err
	}

	if len(i.DBInstances) > 1 {
		return c.Error(400, errors.New("Unexpected number of DBInstances"))
	}
	if i.DBInstances[0].DBClusterIdentifier != nil {
		clusterName = i.DBInstances[0].DBClusterIdentifier
	}

	instanceInput := &rds.DeleteDBInstanceInput{
		DBInstanceIdentifier: aws.String(c.Param("db")),
		SkipFinalSnapshot:    aws.Bool(true),
	}

	if snapshot && clusterName == nil {
		log.Printf("Deleting database %s and creating final snapshot", c.Param("db"))
		instanceInput.FinalDBSnapshotIdentifier = aws.String("final-" + c.Param("db"))
		instanceInput.SkipFinalSnapshot = aws.Bool(false)
	} else {
		log.Printf("Deleting database %s without creating final snapshot", c.Param("db"))
	}

	if instanceOutput, err = rdsClient.Service.DeleteDBInstanceWithContext(c, instanceInput); err != nil {
		log.Println(err.Error())
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == rds.ErrCodeDBInstanceNotFoundFault {
				return c.Error(404, aerr)
			}
			return c.Error(400, aerr)
		}
		return err
	}
	log.Println("Successfully requested deletion of database instance", c.Param("db"), instanceOutput)

	// check if this database instance was part of a cluster
	// and delete the cluster (if this was the last member instance)
	if clusterName != nil {
		clusterInput := &rds.DeleteDBClusterInput{
			DBClusterIdentifier: clusterName,
			SkipFinalSnapshot:   aws.Bool(true),
		}

		if snapshot {
			log.Printf("Trying to delete associated database cluster %s with final snapshot", *clusterName)
			clusterInput.FinalDBSnapshotIdentifier = aws.String("final-" + *clusterName)
			clusterInput.SkipFinalSnapshot = aws.Bool(false)
		} else {
			log.Printf("Trying to delete associated database cluster %s", *clusterName)
		}

		// the cluster deletion will fail if there are still member instances in the cluster
		if clusterOutput, err = rdsClient.Service.DeleteDBClusterWithContext(c, clusterInput); err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				if aerr.Code() == rds.ErrCodeDBClusterNotFoundFault {
					log.Println(rds.ErrCodeDBClusterNotFoundFault, aerr.Error())
				} else {
					log.Println(aerr.Error())
				}
			} else {
				log.Println(err.Error())
			}
		} else {
			log.Println("Successfully requested deletion of database cluster", *clusterName, clusterOutput)
		}
	}

	output := struct {
		*rds.DeleteDBClusterOutput
		*rds.DeleteDBInstanceOutput
	}{
		clusterOutput,
		instanceOutput,
	}

	return c.Render(200, r.JSON(output))
}
