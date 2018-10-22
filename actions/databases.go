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

// DatabasesGet gets a list of databases for a given account
//   If a {db} param is given it will return information about the specific database instance, otherwise
//   all database instances will be returned.
//   If the `all=true` parameter is passed it will return a list of clusters in addition to instances.
func DatabasesGet(c buffalo.Context) error {
	// if all param is given, we'll return information about both instances and clusters
	// otherwise, only database instances will be searched
	all := false
	if b, err := strconv.ParseBool(c.Param("all")); err == nil {
		all = b
	}

	rdsClient := RDS[c.Param("account")]
	var outputClusters *rds.DescribeDBClustersOutput
	var outputInstances *rds.DescribeDBInstancesOutput
	var err error
	clusterNotFound := true

	if all {
		clusterNotFound = false
		inputClusters := &rds.DescribeDBClustersInput{
			DBClusterIdentifier: aws.String(c.Param("db")),
		}

		outputClusters, err = rdsClient.Service.DescribeDBClusters(inputClusters)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case rds.ErrCodeDBClusterNotFoundFault:
					log.Println(rds.ErrCodeDBClusterNotFoundFault, aerr.Error())
					clusterNotFound = true
				default:
					log.Println(aerr.Error())
				}
			}
			log.Println(err.Error())
		}
	}

	inputInstances := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(c.Param("db")),
	}

	outputInstances, err = rdsClient.Service.DescribeDBInstances(inputInstances)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case rds.ErrCodeDBInstanceNotFoundFault:
				log.Println(rds.ErrCodeDBInstanceNotFoundFault, aerr.Error())
				if clusterNotFound {
					return c.Error(404, aerr)
				}
			default:
				log.Println(aerr.Error())
				return c.Error(400, aerr)
			}
		} else {
			log.Println(err.Error())
			return err
		}
	}

	output := struct {
		*rds.DescribeDBClustersOutput
		*rds.DescribeDBInstancesOutput
	}{
		outputClusters,
		outputInstances,
	}
	return c.Render(200, r.JSON(output))
}

// DatabasesPost creates a database in a given account
//   It will create a database instance as specified by the `Instance` hash parameters.
//   If a `Cluster` hash is also given, it will first create an RDS cluster and the instance next.
func DatabasesPost(c buffalo.Context) error {
	type DatabaseCreateInput struct {
		// https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#CreateDBClusterInput
		Cluster *rds.CreateDBClusterInput
		// https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#CreateDBInstanceInput
		Instance *rds.CreateDBInstanceInput
	}
	input := DatabaseCreateInput{}
	if err := c.Bind(&input); err != nil {
		log.Println(err)
		return c.Error(400, err)
	}
	if input.Instance == nil {
		return c.Error(400, errors.New("Bad request"))
	}

	rdsClient := RDS[c.Param("account")]

	// create rds cluster first, if specified
	if input.Cluster != nil {
		if input.Cluster.DBSubnetGroupName == nil {
			input.Cluster.DBSubnetGroupName = aws.String(rdsClient.DefaultSubnet)
		}

		resultCluster, err := rdsClient.Service.CreateDBCluster(input.Cluster)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				log.Println(aerr.Error())
				return c.Error(400, aerr)
			}
			log.Println(err.Error())
			return err
		}
		log.Println("Created RDS cluster", resultCluster)
	}

	// create rds instance
	if input.Instance.DBSubnetGroupName == nil {
		input.Instance.DBSubnetGroupName = aws.String(rdsClient.DefaultSubnet)
	}

	resultInstance, err := rdsClient.Service.CreateDBInstance(input.Instance)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			log.Println(aerr.Error())
			return c.Error(400, aerr)
		}
		log.Println(err.Error())
		return err
	}
	log.Println("Created RDS instance", resultInstance)

	return c.Render(200, r.JSON(resultInstance))
}

// DatabasesDelete deletes a database in a given account
//   It will delete the database instance with the given {db} name and will also delete the associated cluster
//   if the instance belongs to a cluster and is the last remaining member.
//   If the snapshot=true parameter is given, it will create a final snapshot of the instance/cluster.
func DatabasesDelete(c buffalo.Context) error {
	// if snapshot param is given, a final snapshot will be created before deleting
	snapshot := false
	if b, err := strconv.ParseBool(c.Param("snapshot")); err == nil {
		snapshot = b
	}

	rdsClient := RDS[c.Param("account")]
	var inputInstance *rds.DeleteDBInstanceInput
	var inputCluster *rds.DeleteDBClusterInput

	// first, let's determine if the given database instance belongs to a cluster
	var clusterName *string
	i, err := rdsClient.Service.DescribeDBInstances(&rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(c.Param("db")),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case rds.ErrCodeDBInstanceNotFoundFault:
				log.Println(rds.ErrCodeDBInstanceNotFoundFault, aerr.Error())
				return c.Error(404, aerr)
			default:
				log.Println(aerr.Error())
				return c.Error(400, aerr)
			}
		} else {
			log.Println(err.Error())
			return err
		}
	}
	if i.DBInstances[0].DBClusterIdentifier != nil {
		clusterName = i.DBInstances[0].DBClusterIdentifier
	}

	if snapshot && clusterName == nil {
		log.Printf("Deleting database %s and creating final snapshot", c.Param("db"))
		inputInstance = &rds.DeleteDBInstanceInput{
			DBInstanceIdentifier:      aws.String(c.Param("db")),
			FinalDBSnapshotIdentifier: aws.String("final-" + c.Param("db")),
			SkipFinalSnapshot:         aws.Bool(false),
		}
	} else {
		log.Printf("Deleting database %s without creating final snapshot", c.Param("db"))
		inputInstance = &rds.DeleteDBInstanceInput{
			DBInstanceIdentifier: aws.String(c.Param("db")),
			SkipFinalSnapshot:    aws.Bool(true),
		}
	}

	outputInstance, err := rdsClient.Service.DeleteDBInstance(inputInstance)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case rds.ErrCodeDBInstanceNotFoundFault:
				log.Println(rds.ErrCodeDBInstanceNotFoundFault, aerr.Error())
				return c.Error(404, aerr)
			default:
				log.Println(aerr.Error())
				return c.Error(400, aerr)
			}
		}
		log.Println(err.Error())
		return err
	}

	// check if this database instance was part of a cluster
	// and delete the cluster (if this was the last member instance)
	if clusterName != nil {
		if snapshot {
			log.Printf("Trying to delete associated database cluster %s with final snapshot", *clusterName)
			inputCluster = &rds.DeleteDBClusterInput{
				DBClusterIdentifier:       clusterName,
				FinalDBSnapshotIdentifier: aws.String("final-" + *clusterName),
				SkipFinalSnapshot:         aws.Bool(false),
			}
		} else {
			log.Printf("Trying to delete associated database cluster %s", *clusterName)
			inputCluster = &rds.DeleteDBClusterInput{
				DBClusterIdentifier: clusterName,
				SkipFinalSnapshot:   aws.Bool(true),
			}
		}

		// the cluster deletion will fail if there are still member instances in the cluster
		outputCluster, err := rdsClient.Service.DeleteDBCluster(inputCluster)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case rds.ErrCodeDBClusterNotFoundFault:
					log.Println(rds.ErrCodeDBClusterNotFoundFault, aerr.Error())
				default:
					log.Println(aerr.Error())
				}
			} else {
				log.Println(err.Error())
			}
		} else {
			log.Println("Successfully requested deletion of database cluster", *clusterName, outputCluster)
		}
	}

	return c.Render(200, r.JSON(outputInstance))
}
