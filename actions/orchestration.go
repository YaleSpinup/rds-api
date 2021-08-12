package actions

import (
	"errors"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gobuffalo/buffalo"
)

// databaseRestore orchestrates the creation of a database from a snapshot in the DatabaseCreateInput
func (o *rdsOrchestrator) databaseRestore(c buffalo.Context, req *DatabaseCreateRequest) (*DatabaseResponse, error) {
	log.Printf("creating database from snapshot request %+v", req)

	resp := &DatabaseResponse{}

	// restore a database cluster
	if req.Cluster != nil {
		snapshotId := aws.StringValue(req.Cluster.SnapshotIdentifier)
		if snapshotId == "" {
			return nil, errors.New("empty snapshot identifier")
		}

		if req.Cluster.DBClusterIdentifier == nil {
			return nil, errors.New("empty DBClusterIdentifier")
		}

		snapshotsOutput, err := o.client.Service.DescribeDBClusterSnapshotsWithContext(c, &rds.DescribeDBClusterSnapshotsInput{
			DBClusterSnapshotIdentifier: aws.String(snapshotId),
		})
		if err != nil {
			return nil, err
		}
		if len(snapshotsOutput.DBClusterSnapshots) > 1 {
			return nil, errors.New("unexpected number of snapshots")
		}

		snapshot := snapshotsOutput.DBClusterSnapshots[0]

		log.Printf("got snapshot info: %+v", snapshot)

		if aws.StringValue(snapshot.EngineMode) != "serverless" {
			// provisioned and other non-serverless clusters need an instance
			// check that required input was provided
			if req.Instance == nil {
				return nil, errors.New("missing Instance parameters for this cluster, engine mode " + *snapshot.EngineMode)
			}

			if req.Instance.DBInstanceClass == nil {
				return nil, errors.New("empty DBInstanceClass, required for engine mode " + *snapshot.EngineMode)
			}
		}

		log.Printf("restoring database cluster from snapshot %s", snapshotId)

		req.Cluster.Tags = normalizeTags(req.Cluster.Tags)

		// set default subnet group
		if req.Cluster.DBSubnetGroupName == nil {
			req.Cluster.DBSubnetGroupName = aws.String(o.client.DefaultSubnetGroup)
		}

		// set default cluster parameter group
		if req.Cluster.DBClusterParameterGroupName == nil {
			pgFamily, pgErr := o.client.DetermineParameterGroupFamily(snapshot.Engine, snapshot.EngineVersion)
			if pgErr != nil {
				log.Println(pgErr.Error())
				return nil, pgErr
			}
			log.Println("determined ParameterGroupFamily based on Engine:", pgFamily)
			cPg, ok := o.client.DefaultDBClusterParameterGroupName[pgFamily]
			if !ok {
				log.Println("no matching DefaultDBClusterParameterGroupName found in config, using AWS default PG")
			} else {
				log.Println("using DefaultDBClusterParameterGroupName:", cPg)
				req.Cluster.DBClusterParameterGroupName = aws.String(cPg)
			}
		}

		input := &rds.RestoreDBClusterFromSnapshotInput{
			CopyTagsToSnapshot:          aws.Bool(true),
			DBClusterIdentifier:         req.Cluster.DBClusterIdentifier,
			DBClusterParameterGroupName: req.Cluster.DBClusterParameterGroupName,
			DBSubnetGroupName:           req.Cluster.DBSubnetGroupName,
			EnableCloudwatchLogsExports: req.Cluster.EnableCloudwatchLogsExports,
			Engine:                      snapshot.Engine,
			EngineMode:                  snapshot.EngineMode,
			Port:                        req.Cluster.Port,
			SnapshotIdentifier:          aws.String(snapshotId),
			Tags:                        toRDSTags(req.Cluster.Tags),
			VpcSecurityGroupIds:         req.Cluster.VpcSecurityGroupIds,
		}

		if req.Cluster.ScalingConfiguration != nil {
			input.ScalingConfiguration = &rds.ScalingConfiguration{
				AutoPause:             req.Cluster.ScalingConfiguration.AutoPause,
				MaxCapacity:           req.Cluster.ScalingConfiguration.MaxCapacity,
				MinCapacity:           req.Cluster.ScalingConfiguration.MinCapacity,
				SecondsUntilAutoPause: req.Cluster.ScalingConfiguration.SecondsUntilAutoPause,
				TimeoutAction:         req.Cluster.ScalingConfiguration.TimeoutAction,
			}
		}

		log.Printf("restoring database cluster: %+v", *input)

		output, err := o.client.Service.RestoreDBClusterFromSnapshotWithContext(c, input)
		if err != nil {
			return nil, ErrCode("failed to create database cluster from snapshot", err)
		}

		log.Printf("created RDS cluster from snapshot: %+v", output.DBCluster)

		resp.Cluster = output.DBCluster

		// create instance in the cluster, if not serverless (e.g. provisioned)
		if aws.StringValue(snapshot.EngineMode) != "serverless" {
			log.Printf("cluster engine mode is %s, creating database instance ...", *snapshot.EngineMode)

			input := &rds.CreateDBInstanceInput{
				AutoMinorVersionUpgrade: aws.Bool(true),
				CopyTagsToSnapshot:      aws.Bool(true),
				DBClusterIdentifier:     req.Cluster.DBClusterIdentifier,
				DBInstanceClass:         req.Instance.DBInstanceClass,
				DBInstanceIdentifier:    req.Cluster.DBClusterIdentifier,
				Engine:                  snapshot.Engine,
				PubliclyAccessible:      aws.Bool(false),
				StorageEncrypted:        aws.Bool(true),
				Tags:                    toRDSTags(req.Cluster.Tags),
			}

			instanceOutput, err := o.client.Service.CreateDBInstanceWithContext(c, input)
			if err != nil {
				// delete the cluster to clean up
				log.Println("error creating instance, deleting cluster", *req.Cluster.DBClusterIdentifier)
				clusterInput := &rds.DeleteDBClusterInput{
					DBClusterIdentifier: req.Cluster.DBClusterIdentifier,
					SkipFinalSnapshot:   aws.Bool(true),
				}
				if _, errc := o.client.Service.DeleteDBClusterWithContext(c, clusterInput); errc != nil {
					log.Println("failed to delete cluster", errc.Error())
				} else {
					log.Println("successfully requested deletion of cluster", *req.Cluster.DBClusterIdentifier)
				}

				return nil, ErrCode("failed to create database instance", err)
			}

			log.Println("created RDS instance", instanceOutput)

			resp.Instance = instanceOutput.DBInstance
		}

		return resp, nil
	}

	// restore a database instance
	if req.Instance != nil && req.Cluster == nil {
		snapshotId := aws.StringValue(req.Instance.SnapshotIdentifier)
		if snapshotId == "" {
			return nil, errors.New("empty snapshot identifier")
		}

		if req.Instance.DBInstanceIdentifier == nil {
			return nil, errors.New("empty DBInstanceIdentifier")
		}

		// get information about the snapshot
		snapshotsOutput, err := o.client.Service.DescribeDBSnapshotsWithContext(c, &rds.DescribeDBSnapshotsInput{
			DBSnapshotIdentifier: aws.String(snapshotId),
		})
		if err != nil {
			return nil, err
		}
		if len(snapshotsOutput.DBSnapshots) > 1 {
			return nil, errors.New("unexpected number of snapshots")
		}

		snapshot := snapshotsOutput.DBSnapshots[0]

		req.Instance.Tags = normalizeTags(req.Instance.Tags)

		// set default subnet group
		if req.Instance.DBSubnetGroupName == nil {
			req.Instance.DBSubnetGroupName = aws.String(o.client.DefaultSubnetGroup)
		}

		// set default parameter group
		if req.Instance.DBParameterGroupName == nil {
			pgFamily, pgErr := o.client.DetermineParameterGroupFamily(snapshot.Engine, snapshot.EngineVersion)
			if pgErr != nil {
				log.Println(pgErr.Error())
				return nil, pgErr
			}
			log.Println("determined ParameterGroupFamily based on Engine:", pgFamily)
			if pg, ok := o.client.DefaultDBParameterGroupName[pgFamily]; ok {
				log.Println("using DefaultDBParameterGroupName:", pg)
				req.Instance.DBParameterGroupName = aws.String(pg)
			}
		}

		input := &rds.RestoreDBInstanceFromDBSnapshotInput{
			AutoMinorVersionUpgrade:     aws.Bool(true),
			CopyTagsToSnapshot:          aws.Bool(true),
			DBInstanceIdentifier:        req.Instance.DBInstanceIdentifier,
			DBParameterGroupName:        req.Instance.DBParameterGroupName,
			DBSnapshotIdentifier:        aws.String(snapshotId),
			DBSubnetGroupName:           req.Instance.DBSubnetGroupName,
			EnableCloudwatchLogsExports: req.Instance.EnableCloudwatchLogsExports,
			MultiAZ:                     req.Instance.MultiAZ,
			Port:                        req.Instance.Port,
			PubliclyAccessible:          aws.Bool(false),
			Tags:                        toRDSTags(req.Instance.Tags),
			VpcSecurityGroupIds:         req.Instance.VpcSecurityGroupIds,
		}

		log.Printf("restoring database instance: %+v", *input)

		output, err := o.client.Service.RestoreDBInstanceFromDBSnapshotWithContext(c, input)
		if err != nil {
			return nil, ErrCode("failed to create database instance from snapshot", err)
		}

		log.Printf("created RDS instance from snapshot: %+v", output.DBInstance)

		resp.Instance = output.DBInstance
		return resp, nil
	}

	return nil, errors.New("invalid request")
}

// databaseCreate orchestrates the creation of a database from the DatabaseCreateInput
// It will create a database instance as specified by the `Instance` hash parameters.
// If a `Cluster` hash is also given, it will first create an RDS cluster and the instance next.
// If only `Cluster` is specified, it will create an RDS cluster (usually for Aurora serverless)
func (o *rdsOrchestrator) databaseCreate(c buffalo.Context, req *DatabaseCreateRequest) (*DatabaseResponse, error) {
	log.Printf("creating database from request %+v", req)

	var clusterOutput *rds.CreateDBClusterOutput
	var instanceOutput *rds.CreateDBInstanceOutput
	var cluster *rds.DBCluster
	var instance *rds.DBInstance
	var err error

	// create rds cluster first, if specified
	if req.Cluster != nil {
		req.Cluster.Tags = normalizeTags(req.Cluster.Tags)

		// set default storage encryption
		if req.Cluster.StorageEncrypted == nil {
			req.Cluster.StorageEncrypted = aws.Bool(true)
		}

		// set default subnet group
		if req.Cluster.DBSubnetGroupName == nil {
			req.Cluster.DBSubnetGroupName = aws.String(o.client.DefaultSubnetGroup)
		}

		// set default cluster parameter group
		if req.Cluster.DBClusterParameterGroupName == nil {
			pgFamily, pgErr := o.client.DetermineParameterGroupFamily(req.Cluster.Engine, req.Cluster.EngineVersion)
			if pgErr != nil {
				log.Println(pgErr.Error())
				return nil, pgErr
			}
			log.Println("determined ParameterGroupFamily based on Engine:", pgFamily)
			cPg, ok := o.client.DefaultDBClusterParameterGroupName[pgFamily]
			if !ok {
				log.Println("no matching DefaultDBClusterParameterGroupName found in config, using AWS default PG")
			} else {
				log.Println("using DefaultDBClusterParameterGroupName:", cPg)
				req.Cluster.DBClusterParameterGroupName = aws.String(cPg)
			}
		}

		input := &rds.CreateDBClusterInput{
			BackupRetentionPeriod:       req.Cluster.BackupRetentionPeriod,
			CopyTagsToSnapshot:          aws.Bool(true),
			DBClusterIdentifier:         req.Cluster.DBClusterIdentifier,
			DBClusterParameterGroupName: req.Cluster.DBClusterParameterGroupName,
			DBSubnetGroupName:           req.Cluster.DBSubnetGroupName,
			EnableCloudwatchLogsExports: req.Cluster.EnableCloudwatchLogsExports,
			Engine:                      req.Cluster.Engine,
			EngineMode:                  req.Cluster.EngineMode,
			EngineVersion:               req.Cluster.EngineVersion,
			MasterUserPassword:          req.Cluster.MasterUserPassword,
			MasterUsername:              req.Cluster.MasterUsername,
			Port:                        req.Cluster.Port,
			StorageEncrypted:            req.Cluster.StorageEncrypted,
			Tags:                        toRDSTags(req.Cluster.Tags),
			VpcSecurityGroupIds:         req.Cluster.VpcSecurityGroupIds,
		}

		if req.Cluster.ScalingConfiguration != nil {
			input.ScalingConfiguration = &rds.ScalingConfiguration{
				AutoPause:             req.Cluster.ScalingConfiguration.AutoPause,
				MaxCapacity:           req.Cluster.ScalingConfiguration.MaxCapacity,
				MinCapacity:           req.Cluster.ScalingConfiguration.MinCapacity,
				SecondsUntilAutoPause: req.Cluster.ScalingConfiguration.SecondsUntilAutoPause,
				TimeoutAction:         req.Cluster.ScalingConfiguration.TimeoutAction,
			}
		}

		if clusterOutput, err = o.client.Service.CreateDBClusterWithContext(c, input); err != nil {
			return nil, ErrCode("failed to create database cluster", err)
		}

		log.Println("created RDS cluster", clusterOutput)
		cluster = clusterOutput.DBCluster
	}

	// create rds instance, if specified
	if req.Instance != nil {
		req.Instance.Tags = normalizeTags(req.Instance.Tags)

		// set default storage encryption
		if req.Instance.StorageEncrypted == nil {
			req.Instance.StorageEncrypted = aws.Bool(true)
		}

		// set default subnet group
		if req.Instance.DBSubnetGroupName == nil {
			req.Instance.DBSubnetGroupName = aws.String(o.client.DefaultSubnetGroup)
		}

		// set default parameter group
		if req.Instance.DBParameterGroupName == nil {
			pgFamily, pgErr := o.client.DetermineParameterGroupFamily(req.Instance.Engine, req.Instance.EngineVersion)
			if pgErr != nil {
				log.Println(pgErr.Error())
				return nil, pgErr
			}
			log.Println("determined ParameterGroupFamily based on Engine:", pgFamily)
			if pg, ok := o.client.DefaultDBParameterGroupName[pgFamily]; ok {
				log.Println("using DefaultDBParameterGroupName:", pg)
				req.Instance.DBParameterGroupName = aws.String(pg)
			}
		}

		input := &rds.CreateDBInstanceInput{
			AllocatedStorage:            req.Instance.AllocatedStorage,
			AutoMinorVersionUpgrade:     aws.Bool(true),
			BackupRetentionPeriod:       req.Instance.BackupRetentionPeriod,
			CopyTagsToSnapshot:          aws.Bool(true),
			DBClusterIdentifier:         req.Instance.DBClusterIdentifier,
			DBInstanceClass:             req.Instance.DBInstanceClass,
			DBInstanceIdentifier:        req.Instance.DBInstanceIdentifier,
			DBParameterGroupName:        req.Instance.DBParameterGroupName,
			DBSubnetGroupName:           req.Instance.DBSubnetGroupName,
			EnableCloudwatchLogsExports: req.Instance.EnableCloudwatchLogsExports,
			Engine:                      req.Instance.Engine,
			EngineVersion:               req.Instance.EngineVersion,
			MasterUserPassword:          req.Instance.MasterUserPassword,
			MasterUsername:              req.Instance.MasterUsername,
			MultiAZ:                     req.Instance.MultiAZ,
			Port:                        req.Instance.Port,
			PubliclyAccessible:          aws.Bool(false),
			StorageEncrypted:            req.Instance.StorageEncrypted,
			Tags:                        toRDSTags(req.Instance.Tags),
			VpcSecurityGroupIds:         req.Instance.VpcSecurityGroupIds,
		}

		if instanceOutput, err = o.client.Service.CreateDBInstanceWithContext(c, input); err != nil {
			if req.Cluster != nil {
				// if this instance was in a new cluster, delete the cluster
				log.Println("deleting cluster", *req.Cluster.DBClusterIdentifier)
				clusterInput := &rds.DeleteDBClusterInput{
					DBClusterIdentifier: req.Cluster.DBClusterIdentifier,
					SkipFinalSnapshot:   aws.Bool(true),
				}
				if _, errc := o.client.Service.DeleteDBClusterWithContext(c, clusterInput); errc != nil {
					log.Println("failed to delete cluster", errc.Error())
				} else {
					log.Println("successfully requested deletion of cluster", *req.Cluster.DBClusterIdentifier)
				}
			}
			return nil, ErrCode("failed to create database instance", err)
		}

		log.Println("created RDS instance", instanceOutput)
		instance = instanceOutput.DBInstance
	}

	return &DatabaseResponse{
		Cluster:  cluster,
		Instance: instance,
	}, nil
}

// databaseModify modifies database parameters and tags
// Either Cluster or Instance input parameters can be specified for a request
// Tags list can be given with any key/value tags to add/update
func (o *rdsOrchestrator) databaseModify(c buffalo.Context, id string, input *DatabaseModifyInput) (*DatabaseResponse, error) {
	log.Printf("modifying database %s with input %+v", id, input)

	var clusterOutput *rds.ModifyDBClusterOutput
	var instanceOutput *rds.ModifyDBInstanceOutput
	var cluster *rds.DBCluster
	var instance *rds.DBInstance
	var err error

	if input.Cluster != nil {
		input.Cluster.DBClusterIdentifier = aws.String(id)

		// set default cluster parameter group when upgrading engine version
		if input.Cluster.DBClusterParameterGroupName == nil && input.Cluster.EngineVersion != nil {
			// get information about the existing cluster to determine the engine type
			describeClusterOutput, err := o.client.Service.DescribeDBClustersWithContext(c, &rds.DescribeDBClustersInput{
				DBClusterIdentifier: aws.String(id),
			})
			if err == nil && describeClusterOutput != nil {
				pgFamily, pgErr := o.client.DetermineParameterGroupFamily(describeClusterOutput.DBClusters[0].Engine, input.Cluster.EngineVersion)
				if pgErr != nil {
					log.Println(pgErr.Error())
					return nil, pgErr
				}
				log.Println("determined ParameterGroupFamily based on Engine:", pgFamily)
				cPg, ok := o.client.DefaultDBClusterParameterGroupName[pgFamily]
				if !ok {
					log.Println("no matching DefaultDBClusterParameterGroupName found in config, using AWS default PG")
				} else {
					log.Println("using DefaultDBClusterParameterGroupName:", cPg)
					input.Cluster.DBClusterParameterGroupName = aws.String(cPg)
				}
			}
		}

		if clusterOutput, err = o.client.Service.ModifyDBClusterWithContext(c, input.Cluster); err != nil {
			return nil, ErrCode("failed to modify database cluster", err)
		}

		log.Println("modified RDS cluster", clusterOutput)
		cluster = clusterOutput.DBCluster
	}

	if input.Instance != nil {
		input.Instance.DBInstanceIdentifier = aws.String(id)

		// set default instance parameter group when upgrading engine version
		if input.Instance.DBParameterGroupName == nil && input.Instance.EngineVersion != nil {
			// get information about the existing instance to determine the engine type
			describeInstanceOutput, err := o.client.Service.DescribeDBInstancesWithContext(c, &rds.DescribeDBInstancesInput{
				DBInstanceIdentifier: aws.String(id),
			})
			if err == nil && describeInstanceOutput != nil {
				pgFamily, pgErr := o.client.DetermineParameterGroupFamily(describeInstanceOutput.DBInstances[0].Engine, input.Instance.EngineVersion)
				if pgErr != nil {
					log.Println(pgErr.Error())
					return nil, pgErr
				}
				log.Println("determined ParameterGroupFamily based on Engine:", pgFamily)
				if pg, ok := o.client.DefaultDBParameterGroupName[pgFamily]; ok {
					log.Println("using DefaultDBParameterGroupName:", pg)
					input.Instance.DBParameterGroupName = aws.String(pg)
				}
			}
		}

		if instanceOutput, err = o.client.Service.ModifyDBInstanceWithContext(c, input.Instance); err != nil {
			return nil, ErrCode("failed to modify database instance", err)
		}

		log.Println("modified RDS instance", instanceOutput)
		instance = instanceOutput.DBInstance
	}

	if input.Tags != nil {
		log.Println("updating tags for "+id, input.Tags)

		// determine ARN(s) for this RDS resource
		arns, err := o.client.DetermineArn(id)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		normalizedTags := normalizeTags(fromRDSTags(input.Tags))

		// update tags for all RDS resources with matching ARNs
		for _, arn := range arns {
			if _, err = o.client.Service.AddTagsToResourceWithContext(c, &rds.AddTagsToResourceInput{
				ResourceName: aws.String(arn),
				Tags:         toRDSTags(normalizedTags),
			}); err != nil {
				return nil, ErrCode("failed to add tags to database", err)
			}
			log.Println("updated tags for RDS resource", arn)
		}
	}

	return &DatabaseResponse{
		Cluster:  cluster,
		Instance: instance,
	}, nil
}

// databaseDelete deletes a database
// It will delete the database instance with the given {db} name and will also delete the associated cluster
//  if the instance belongs to a cluster and is the last remaining member.
// If snapshot is true, it will create a final snapshot of the instance/cluster.
func (o *rdsOrchestrator) databaseDelete(c buffalo.Context, id string, snapshot bool) (*DatabaseResponse, error) {
	log.Printf("deleting database %s (snapshot: %t)", id, snapshot)

	var clusterOutput *rds.DeleteDBClusterOutput
	var instanceOutput *rds.DeleteDBInstanceOutput
	var cluster *rds.DBCluster
	var instance *rds.DBInstance
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
				log.Printf("no matching database instance found: %s", id)
				instanceNotFound = true
			} else {
				return nil, ErrCode("failed to describe database instance", err)
			}
		}
	}

	// if a db instance exists, delete it
	if describeInstanceOutput != nil && !instanceNotFound {
		if len(describeInstanceOutput.DBInstances) > 1 {
			return nil, errors.New("unexpected number of DBInstances")
		}
		if len(describeInstanceOutput.DBInstances) < 1 {
			return nil, errors.New("no DBInstances found")
		}
		if describeInstanceOutput.DBInstances[0].DBClusterIdentifier != nil {
			clusterName = describeInstanceOutput.DBInstances[0].DBClusterIdentifier
		}

		instanceInput := &rds.DeleteDBInstanceInput{
			DBInstanceIdentifier: aws.String(id),
			SkipFinalSnapshot:    aws.Bool(true),
		}

		if snapshot && clusterName == nil {
			log.Printf("deleting database %s and creating final snapshot", id)
			instanceInput.FinalDBSnapshotIdentifier = aws.String("final-" + id)
			instanceInput.SkipFinalSnapshot = aws.Bool(false)
		} else {
			log.Printf("deleting database %s without creating final snapshot", id)
		}

		if instanceOutput, err = o.client.Service.DeleteDBInstanceWithContext(c, instanceInput); err != nil {
			return nil, ErrCode("failed to delete database instance", err)
		}

		log.Println("successfully requested deletion of database instance", id, instanceOutput)
		instance = instanceOutput.DBInstance
	}

	// check if this db instance was part of a cluster
	// and delete the cluster, if this was the last member instance
	if clusterName != nil && !instanceNotFound {
		clusterInput := &rds.DeleteDBClusterInput{
			DBClusterIdentifier: clusterName,
			SkipFinalSnapshot:   aws.Bool(true),
		}

		if snapshot {
			log.Printf("trying to delete associated database cluster %s with final snapshot", *clusterName)
			clusterInput.FinalDBSnapshotIdentifier = aws.String("final-" + *clusterName)
			clusterInput.SkipFinalSnapshot = aws.Bool(false)
		} else {
			log.Printf("trying to delete associated database cluster %s", *clusterName)
		}

		// the cluster deletion will fail if there are still member instances in the cluster
		if clusterOutput, err = o.client.Service.DeleteDBClusterWithContext(c, clusterInput); err != nil {
			return nil, ErrCode("failed to delete database cluster", err)
		}

		log.Println("successfully requested deletion of database cluster", *clusterName, clusterOutput)
		cluster = clusterOutput.DBCluster
	}

	// delete cluster (with no associated instances)
	if instanceNotFound {
		clusterName = aws.String(id)
		clusterInput := &rds.DeleteDBClusterInput{
			DBClusterIdentifier: clusterName,
			SkipFinalSnapshot:   aws.Bool(true),
		}

		if snapshot {
			log.Printf("trying to delete database cluster %s with final snapshot", *clusterName)
			clusterInput.FinalDBSnapshotIdentifier = aws.String("final-" + *clusterName)
			clusterInput.SkipFinalSnapshot = aws.Bool(false)
		} else {
			log.Printf("trying to delete database cluster %s", *clusterName)
		}

		if clusterOutput, err = o.client.Service.DeleteDBClusterWithContext(c, clusterInput); err != nil {
			return nil, ErrCode("failed to delete database cluster", err)
		}

		log.Println("successfully requested deletion of database cluster", *clusterName, clusterOutput)
		cluster = clusterOutput.DBCluster
	}

	return &DatabaseResponse{
		Cluster:  cluster,
		Instance: instance,
	}, nil
}

func (o *rdsOrchestrator) clusterSnapshotCreate(c buffalo.Context, cluster, snapshot string) (*rds.DBClusterSnapshot, error) {
	clusterSnapshotOutput, err := o.client.Service.CreateDBClusterSnapshotWithContext(c, &rds.CreateDBClusterSnapshotInput{
		DBClusterIdentifier:         aws.String(cluster),
		DBClusterSnapshotIdentifier: aws.String(snapshot),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == rds.ErrCodeDBClusterNotFoundFault {
				return nil, nil
			}
			return nil, c.Error(400, aerr)
		}
		return nil, err
	}

	return clusterSnapshotOutput.DBClusterSnapshot, nil
}

func (o *rdsOrchestrator) clusterSnapshotDelete(c buffalo.Context, snapshot string) (*rds.DBClusterSnapshot, error) {
	clusterSnapshotOutput, err := o.client.Service.DeleteDBClusterSnapshotWithContext(c, &rds.DeleteDBClusterSnapshotInput{
		DBClusterSnapshotIdentifier: aws.String(snapshot),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == rds.ErrCodeDBClusterSnapshotNotFoundFault {
				return nil, nil
			}
			return nil, c.Error(400, aerr)
		}
		return nil, err
	}

	return clusterSnapshotOutput.DBClusterSnapshot, nil
}

func (o *rdsOrchestrator) instanceSnapshotCreate(c buffalo.Context, instance, snapshot string) (*rds.DBSnapshot, error) {
	instanceSnapshotOutput, err := o.client.Service.CreateDBSnapshotWithContext(c, &rds.CreateDBSnapshotInput{
		DBInstanceIdentifier: aws.String(instance),
		DBSnapshotIdentifier: aws.String(snapshot),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == rds.ErrCodeDBInstanceNotFoundFault {
				return nil, nil
			}
			return nil, c.Error(400, aerr)
		}
		return nil, err
	}

	return instanceSnapshotOutput.DBSnapshot, nil
}

func (o *rdsOrchestrator) instanceSnapshotDelete(c buffalo.Context, snapshot string) (*rds.DBSnapshot, error) {
	instanceSnapshotOutput, err := o.client.Service.DeleteDBSnapshotWithContext(c, &rds.DeleteDBSnapshotInput{
		DBSnapshotIdentifier: aws.String(snapshot),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == rds.ErrCodeDBSnapshotNotFoundFault {
				return nil, nil
			}
			return nil, c.Error(400, aerr)
		}
		return nil, err
	}

	return instanceSnapshotOutput.DBSnapshot, nil
}
