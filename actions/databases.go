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

// DatabaseModifyInput is the input for modifying an existing database
type DatabaseModifyInput struct {
	// https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#ModifyDBClusterInput
	Cluster *rds.ModifyDBClusterInput
	// https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#ModifyDBInstanceInput
	Instance *rds.ModifyDBInstanceInput
	Tags     []*rds.Tag
}

// DatabaseStateInput is the input for changing the database state
type DatabaseStateInput struct {
	State string
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
// If only `Cluster` is specified, it will create an RDS cluster (usually for Aurora serverless)
func DatabasesPost(c buffalo.Context) error {
	input := DatabaseCreateInput{}
	if err := c.Bind(&input); err != nil {
		log.Println(err)
		return c.Error(400, err)
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
		// set default subnet group
		if input.Cluster.DBSubnetGroupName == nil {
			input.Cluster.DBSubnetGroupName = aws.String(rdsClient.DefaultSubnetGroup)
		}
		// set default cluster parameter group
		if input.Cluster.DBClusterParameterGroupName == nil {
			pgFamily, pgErr := rdsClient.DetermineParameterGroupFamily(input.Cluster.Engine, input.Cluster.EngineVersion)
			if pgErr != nil {
				log.Println(pgErr.Error())
				return c.Error(400, pgErr)
			}
			log.Println("Determined ParameterGroupFamily based on Engine:", pgFamily)
			cPg, ok := rdsClient.DefaultDBClusterParameterGroupName[pgFamily]
			if !ok {
				log.Println("No matching DefaultDBClusterParameterGroupName found in config, using AWS default PG")
			} else {
				log.Println("Using DefaultDBClusterParameterGroupName:", cPg)
				input.Cluster.DBClusterParameterGroupName = aws.String(cPg)
			}
		}

		input.Cluster.Tags = normalizeTags(input.Cluster.Tags)
		if clusterOutput, err = rdsClient.Service.CreateDBClusterWithContext(c, input.Cluster); err != nil {
			log.Println(err.Error())
			if aerr, ok := err.(awserr.Error); ok {
				return c.Error(400, aerr)
			}
			return err
		}
		log.Println("Created RDS cluster", clusterOutput)
	}

	// create rds instance, if specified
	if input.Instance != nil {
		// set default subnet group
		if input.Instance.DBSubnetGroupName == nil {
			input.Instance.DBSubnetGroupName = aws.String(rdsClient.DefaultSubnetGroup)
		}
		// set default parameter group
		if input.Instance.DBParameterGroupName == nil {
			pgFamily, pgErr := rdsClient.DetermineParameterGroupFamily(input.Instance.Engine, input.Instance.EngineVersion)
			if pgErr != nil {
				log.Println(pgErr.Error())
				return c.Error(400, pgErr)
			}
			log.Println("Determined ParameterGroupFamily based on Engine:", pgFamily)
			pg, ok := rdsClient.DefaultDBParameterGroupName[pgFamily]
			if !ok {
				log.Println("No matching DefaultDBParameterGroupName found in config, using AWS default PG")
			} else {
				log.Println("Using DefaultDBParameterGroupName:", pg)
				input.Instance.DBParameterGroupName = aws.String(pg)
			}
		}

		input.Instance.Tags = normalizeTags(input.Instance.Tags)
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
	}

	output := struct {
		*rds.CreateDBClusterOutput
		*rds.CreateDBInstanceOutput
	}{
		clusterOutput,
		instanceOutput,
	}

	return c.Render(200, r.JSON(output))
}

// DatabasesPut modifies a database in a given account
// Either Cluster or Instance input parameters can be specified for a request
// Tags list can be given with any key/value tags to add/update
func DatabasesPut(c buffalo.Context) error {
	input := DatabaseModifyInput{}
	if err := c.Bind(&input); err != nil {
		log.Println(err)
		return c.Error(400, err)
	}
	if input.Cluster == nil && input.Instance == nil && input.Tags == nil {
		return c.Error(400, errors.New("Bad request"))
	}

	if input.Cluster != nil && input.Instance != nil {
		return c.Error(400, errors.New("Bad request: cannot specify both Cluster and Instance"))
	}

	rdsClient, ok := RDS[c.Param("account")]
	if !ok {
		return c.Error(400, errors.New("Bad request: unknown account "+c.Param("account")))
	}

	var clusterOutput *rds.ModifyDBClusterOutput
	var instanceOutput *rds.ModifyDBInstanceOutput
	var err error

	if input.Cluster != nil {
		input.Cluster.DBClusterIdentifier = aws.String(c.Param("db"))
		if clusterOutput, err = rdsClient.Service.ModifyDBClusterWithContext(c, input.Cluster); err != nil {
			log.Println(err.Error())
			if aerr, ok := err.(awserr.Error); ok {
				return c.Error(400, aerr)
			}
			return err
		}
		log.Println("Modified RDS cluster", clusterOutput)
	}

	if input.Instance != nil {
		input.Instance.DBInstanceIdentifier = aws.String(c.Param("db"))
		if instanceOutput, err = rdsClient.Service.ModifyDBInstanceWithContext(c, input.Instance); err != nil {
			log.Println(err.Error())
			if aerr, ok := err.(awserr.Error); ok {
				return c.Error(400, aerr)
			}
			return err
		}
		log.Println("Modified RDS instance", instanceOutput)
	}

	if input.Tags != nil {
		log.Println("Updating tags for "+c.Param("db"), input.Tags)

		// determine ARN(s) for this RDS resource
		arns, err := rdsClient.DetermineArn(c.Param("db"))
		if err != nil {
			log.Println(err)
			return c.Error(400, err)
		}

		normalizedTags := normalizeTags(input.Tags)

		// update tags for all RDS resources with matching ARNs
		for _, arn := range arns {
			if _, err = rdsClient.Service.AddTagsToResourceWithContext(c, &rds.AddTagsToResourceInput{
				ResourceName: aws.String(arn),
				Tags:         normalizedTags,
			}); err != nil {
				return c.Error(400, err)
			}
			log.Println("Updated tags for RDS resource", arn)
		}
	}

	output := struct {
		*rds.ModifyDBClusterOutput
		*rds.ModifyDBInstanceOutput
	}{
		clusterOutput,
		instanceOutput,
	}

	return c.Render(200, r.JSON(output))
}

// DatabasesPutState stops or starts a database in a given account
func DatabasesPutState(c buffalo.Context) error {
	input := DatabaseStateInput{}
	if err := c.Bind(&input); err != nil {
		log.Println(err)
		return c.Error(400, err)
	}

	rdsClient, ok := RDS[c.Param("account")]
	if !ok {
		return c.Error(400, errors.New("Bad request: unknown account "+c.Param("account")))
	}

	id := c.Param("db")
	if id == "" {
		return c.Error(400, errors.New("Bad request: missing database identifier"))
	}

	switch input.State {
	case "start":
		if err := rdsClient.StartDatabase(c, id); err != nil {
			return c.Error(400, err)
		}
	case "stop":
		if err := rdsClient.StopDatabase(c, id); err != nil {
			return c.Error(400, err)
		}
	default:
		return c.Error(400, errors.New("Invalid state.  Valid states are 'stop' or 'start'."))
	}

	return c.Render(200, r.JSON("OK"))
}

// DatabasesDelete deletes a database in a given account
// It will delete the database instance with the given {db} name and will also delete the associated cluster
//  if the instance belongs to a cluster and is the last remaining member.
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
	instanceNotFound := false

	// first, let's determine if the given database instance belongs to a cluster
	i, err := rdsClient.Service.DescribeDBInstancesWithContext(c, &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(c.Param("db")),
	})
	if err != nil {
		log.Println(err.Error())
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == rds.ErrCodeDBInstanceNotFoundFault {
				log.Printf("No matching database instance found: %s", c.Param("db"))
				instanceNotFound = true
			} else {
				return c.Error(400, aerr)
			}
		}
	}

	if i != nil && !instanceNotFound {
		if len(i.DBInstances) > 1 {
			return c.Error(400, errors.New("Unexpected number of DBInstances"))
		}
		if len(i.DBInstances) < 1 {
			return c.Error(400, errors.New("No DBInstances found"))
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
	}

	// check if this database instance was part of a cluster
	// and delete the cluster (if this was the last member instance)
	if clusterName != nil && !instanceNotFound {
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

	// delete cluster (with no associated instances)
	if instanceNotFound {
		clusterName = aws.String(c.Param("db"))
		clusterInput := &rds.DeleteDBClusterInput{
			DBClusterIdentifier: clusterName,
			SkipFinalSnapshot:   aws.Bool(true),
		}

		if snapshot {
			log.Printf("Trying to delete database cluster %s with final snapshot", *clusterName)
			clusterInput.FinalDBSnapshotIdentifier = aws.String("final-" + *clusterName)
			clusterInput.SkipFinalSnapshot = aws.Bool(false)
		} else {
			log.Printf("Trying to delete database cluster %s", *clusterName)
		}

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

// normalizeTags strips the org from the given tags and ensures it is set to the API org
func normalizeTags(tags []*rds.Tag) []*rds.Tag {
	normalizedTags := []*rds.Tag{}
	for _, t := range tags {
		if aws.StringValue(t.Key) == "spinup:org" || aws.StringValue(t.Key) == "yale:org" {
			continue
		}
		normalizedTags = append(normalizedTags, t)
	}

	normalizedTags = append(normalizedTags,
		&rds.Tag{
			Key:   aws.String("spinup:org"),
			Value: aws.String(AppConfig.Org),
		})

	return normalizedTags
}
