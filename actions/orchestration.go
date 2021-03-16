package actions

import (
	"errors"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gobuffalo/buffalo"
)

// databaseCreate orchestrates the creation of a repository from the DatabaseCreateInput
// It will create a database instance as specified by the `Instance` hash parameters.
// If a `Cluster` hash is also given, it will first create an RDS cluster and the instance next.
// If only `Cluster` is specified, it will create an RDS cluster (usually for Aurora serverless)
func (o *rdsOrchestrator) databaseCreate(c buffalo.Context, input *DatabaseCreateInput) (*DatabaseCreateOutput, error) {
	log.Printf("creating database with input %+v", input)

	var clusterOutput *rds.CreateDBClusterOutput
	var instanceOutput *rds.CreateDBInstanceOutput
	var err error

	// create rds cluster first, if specified
	if input.Cluster != nil {
		// set default subnet group
		if input.Cluster.DBSubnetGroupName == nil {
			input.Cluster.DBSubnetGroupName = aws.String(o.client.DefaultSubnetGroup)
		}
		// set default cluster parameter group
		if input.Cluster.DBClusterParameterGroupName == nil {
			pgFamily, pgErr := o.client.DetermineParameterGroupFamily(input.Cluster.Engine, input.Cluster.EngineVersion)
			if pgErr != nil {
				log.Println(pgErr.Error())
				return nil, pgErr
			}
			log.Println("Determined ParameterGroupFamily based on Engine:", pgFamily)
			cPg, ok := o.client.DefaultDBClusterParameterGroupName[pgFamily]
			if !ok {
				log.Println("No matching DefaultDBClusterParameterGroupName found in config, using AWS default PG")
			} else {
				log.Println("Using DefaultDBClusterParameterGroupName:", cPg)
				input.Cluster.DBClusterParameterGroupName = aws.String(cPg)
			}
		}

		input.Cluster.Tags = normalizeTags(input.Cluster.Tags)
		if clusterOutput, err = o.client.Service.CreateDBClusterWithContext(c, input.Cluster); err != nil {
			return nil, ErrCode("failed to create database cluster", err)
		}

		log.Println("Created RDS cluster", clusterOutput)
	}

	// create rds instance, if specified
	if input.Instance != nil {
		// set default subnet group
		if input.Instance.DBSubnetGroupName == nil {
			input.Instance.DBSubnetGroupName = aws.String(o.client.DefaultSubnetGroup)
		}
		// set default parameter group
		if input.Instance.DBParameterGroupName == nil {
			pgFamily, pgErr := o.client.DetermineParameterGroupFamily(input.Instance.Engine, input.Instance.EngineVersion)
			if pgErr != nil {
				log.Println(pgErr.Error())
				return nil, pgErr
			}
			log.Println("Determined ParameterGroupFamily based on Engine:", pgFamily)
			if pg, ok := o.client.DefaultDBParameterGroupName[pgFamily]; ok {
				log.Println("Using DefaultDBParameterGroupName:", pg)
				input.Instance.DBParameterGroupName = aws.String(pg)
			}
		}

		input.Instance.Tags = normalizeTags(input.Instance.Tags)
		if instanceOutput, err = o.client.Service.CreateDBInstanceWithContext(c, input.Instance); err != nil {
			if input.Cluster != nil {
				// if this instance was in a new cluster, delete the cluster
				log.Println("Deleting cluster", *input.Cluster.DBClusterIdentifier)
				clusterInput := &rds.DeleteDBClusterInput{
					DBClusterIdentifier: input.Cluster.DBClusterIdentifier,
					SkipFinalSnapshot:   aws.Bool(true),
				}
				if _, errc := o.client.Service.DeleteDBClusterWithContext(c, clusterInput); errc != nil {
					log.Println("Failed to delete cluster", errc.Error())
				} else {
					log.Println("Successfully requested deletion of cluster", *input.Cluster.DBClusterIdentifier)
				}
			}
			return nil, ErrCode("failed to create database instance", err)
		}

		log.Println("Created RDS instance", instanceOutput)
	}

	return &DatabaseCreateOutput{clusterOutput, instanceOutput}, nil
}

// databaseModify modifies database parameters and tags
// Either Cluster or Instance input parameters can be specified for a request
// Tags list can be given with any key/value tags to add/update
func (o *rdsOrchestrator) databaseModify(c buffalo.Context, id string, input *DatabaseModifyInput) (*DatabaseModifyOutput, error) {
	log.Printf("modifying database %s with input %+v", id, input)

	var clusterOutput *rds.ModifyDBClusterOutput
	var instanceOutput *rds.ModifyDBInstanceOutput
	var err error

	if input.Cluster != nil {
		input.Cluster.DBClusterIdentifier = aws.String(id)

		// set default cluster parameter group when upgrading engine version
		if input.Cluster.DBClusterParameterGroupName == nil && input.Cluster.EngineVersion != nil {
			// get information about the existing cluster to determine the engine type
			describeClusterOutput, err := o.client.Service.DescribeDBClustersWithContext(c, &rds.DescribeDBClustersInput{})
			if err == nil && describeClusterOutput != nil {
				pgFamily, pgErr := o.client.DetermineParameterGroupFamily(describeClusterOutput.DBClusters[0].Engine, input.Cluster.EngineVersion)
				if pgErr != nil {
					log.Println(pgErr.Error())
					return nil, pgErr
				}
				log.Println("Determined ParameterGroupFamily based on Engine:", pgFamily)
				cPg, ok := o.client.DefaultDBClusterParameterGroupName[pgFamily]
				if !ok {
					log.Println("No matching DefaultDBClusterParameterGroupName found in config, using AWS default PG")
				} else {
					log.Println("Using DefaultDBClusterParameterGroupName:", cPg)
					input.Cluster.DBClusterParameterGroupName = aws.String(cPg)
				}
			}
		}

		if clusterOutput, err = o.client.Service.ModifyDBClusterWithContext(c, input.Cluster); err != nil {
			return nil, ErrCode("failed to modify database cluster", err)
		}

		log.Println("Modified RDS cluster", clusterOutput)
	}

	if input.Instance != nil {
		input.Instance.DBInstanceIdentifier = aws.String(id)

		// set default instance parameter group when upgrading engine version
		if input.Instance.DBParameterGroupName == nil && input.Instance.EngineVersion != nil {
			// get information about the existing instance to determine the engine type
			describeInstanceOutput, err := o.client.Service.DescribeDBInstancesWithContext(c, &rds.DescribeDBInstancesInput{})
			if err == nil && describeInstanceOutput != nil {
				pgFamily, pgErr := o.client.DetermineParameterGroupFamily(describeInstanceOutput.DBInstances[0].Engine, input.Instance.EngineVersion)
				if pgErr != nil {
					log.Println(pgErr.Error())
					return nil, pgErr
				}
				log.Println("Determined ParameterGroupFamily based on Engine:", pgFamily)
				if pg, ok := o.client.DefaultDBParameterGroupName[pgFamily]; ok {
					log.Println("Using DefaultDBParameterGroupName:", pg)
					input.Instance.DBParameterGroupName = aws.String(pg)
				}
			}
		}

		if instanceOutput, err = o.client.Service.ModifyDBInstanceWithContext(c, input.Instance); err != nil {
			return nil, ErrCode("failed to modify database instance", err)
		}

		log.Println("Modified RDS instance", instanceOutput)
	}

	if input.Tags != nil {
		log.Println("Updating tags for "+id, input.Tags)

		// determine ARN(s) for this RDS resource
		arns, err := o.client.DetermineArn(id)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		normalizedTags := normalizeTags(input.Tags)

		// update tags for all RDS resources with matching ARNs
		for _, arn := range arns {
			if _, err = o.client.Service.AddTagsToResourceWithContext(c, &rds.AddTagsToResourceInput{
				ResourceName: aws.String(arn),
				Tags:         normalizedTags,
			}); err != nil {
				return nil, ErrCode("failed to add tags to database", err)
			}
			log.Println("Updated tags for RDS resource", arn)
		}
	}

	return &DatabaseModifyOutput{clusterOutput, instanceOutput}, nil
}

// databaseDelete deletes a database
// It will delete the database instance with the given {db} name and will also delete the associated cluster
//  if the instance belongs to a cluster and is the last remaining member.
// If snapshot is true, it will create a final snapshot of the instance/cluster.
func (o *rdsOrchestrator) databaseDelete(c buffalo.Context, id string, snapshot bool) (*DatabaseDeleteOutput, error) {
	log.Printf("deleting database %s (snapshot: %t)", id, snapshot)

	var clusterOutput *rds.DeleteDBClusterOutput
	var instanceOutput *rds.DeleteDBInstanceOutput
	var err error
	var clusterName *string
	var instanceNotFound bool

	// first, let's determine if the given database instance belongs to a cluster
	describeInstanceOutput, err := o.client.Service.DescribeDBInstancesWithContext(c, &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(id),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == rds.ErrCodeDBInstanceNotFoundFault {
				log.Printf("No matching database instance found: %s", id)
				instanceNotFound = true
			} else {
				return nil, ErrCode("failed to describe database instance", err)
			}
		}
	}

	// if a db instance exists, delete it
	if describeInstanceOutput != nil && !instanceNotFound {
		if len(describeInstanceOutput.DBInstances) > 1 {
			return nil, errors.New("Unexpected number of DBInstances")
		}
		if len(describeInstanceOutput.DBInstances) < 1 {
			return nil, errors.New("No DBInstances found")
		}
		if describeInstanceOutput.DBInstances[0].DBClusterIdentifier != nil {
			clusterName = describeInstanceOutput.DBInstances[0].DBClusterIdentifier
		}

		instanceInput := &rds.DeleteDBInstanceInput{
			DBInstanceIdentifier: aws.String(id),
			SkipFinalSnapshot:    aws.Bool(true),
		}

		if snapshot && clusterName == nil {
			log.Printf("Deleting database %s and creating final snapshot", id)
			instanceInput.FinalDBSnapshotIdentifier = aws.String("final-" + id)
			instanceInput.SkipFinalSnapshot = aws.Bool(false)
		} else {
			log.Printf("Deleting database %s without creating final snapshot", id)
		}

		if instanceOutput, err = o.client.Service.DeleteDBInstanceWithContext(c, instanceInput); err != nil {
			return nil, ErrCode("failed to delete database instance", err)
		}

		log.Println("Successfully requested deletion of database instance", id, instanceOutput)
	}

	// check if this db instance was part of a cluster
	// and delete the cluster, if this was the last member instance
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
		if clusterOutput, err = o.client.Service.DeleteDBClusterWithContext(c, clusterInput); err != nil {
			return nil, ErrCode("failed to delete database cluster", err)
		}

		log.Println("Successfully requested deletion of database cluster", *clusterName, clusterOutput)
	}

	// delete cluster (with no associated instances)
	if instanceNotFound {
		clusterName = aws.String(id)
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

		if clusterOutput, err = o.client.Service.DeleteDBClusterWithContext(c, clusterInput); err != nil {
			return nil, ErrCode("failed to delete database cluster", err)
		}

		log.Println("Successfully requested deletion of database cluster", *clusterName, clusterOutput)
	}

	return &DatabaseDeleteOutput{clusterOutput, instanceOutput}, nil
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
