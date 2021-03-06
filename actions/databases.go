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

// DatabaseCreateInput is the input for creating a new database
// The Instance part is required and defines the database instance properties
// The Cluster is optional if the created database instance belongs to a new cluster
type DatabaseCreateInput struct {
	// https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#CreateDBClusterInput
	Cluster *rds.CreateDBClusterInput
	// https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#CreateDBInstanceInput
	Instance *rds.CreateDBInstanceInput
}

// DatabaseCreateOutput is the output from creating a new database
type DatabaseCreateOutput struct {
	// https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#CreateDBClusterOutput
	Cluster *rds.CreateDBClusterOutput
	// https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#CreateDBInstanceOutput
	Instance *rds.CreateDBInstanceOutput
}

// DatabaseDeleteOutput is the output from deleting a database
type DatabaseDeleteOutput struct {
	// https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#DeleteDBClusterOutput
	Cluster *rds.DeleteDBClusterOutput
	// https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#DeleteDBInstanceOutput
	Instance *rds.DeleteDBInstanceOutput
}

// DatabaseModifyInput is the input for modifying an existing database
type DatabaseModifyInput struct {
	// https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#ModifyDBClusterInput
	Cluster *rds.ModifyDBClusterInput
	// https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#ModifyDBInstanceInput
	Instance *rds.ModifyDBInstanceInput
	Tags     []*rds.Tag
}

// DatabaseModifyOutput is the output from modifying an existing database
type DatabaseModifyOutput struct {
	// https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#ModifyDBClusterOutput
	Cluster *rds.ModifyDBClusterOutput
	// https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#ModifyDBInstanceOutput
	Instance *rds.ModifyDBInstanceOutput
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
		return handleError(c, err)
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
func DatabasesPost(c buffalo.Context) error {
	input := DatabaseCreateInput{}
	if err := c.Bind(&input); err != nil {
		log.Println(err)
		return c.Error(400, err)
	}

	if input.Cluster == nil && input.Instance == nil {
		return c.Error(400, errors.New("Bad request: specify Cluster or Instance in request"))
	}

	rdsClient, ok := RDS[c.Param("account")]
	if !ok {
		return c.Error(400, errors.New("Bad request: unknown account "+c.Param("account")))
	}

	orch := &rdsOrchestrator{
		client: rdsClient,
	}

	resp, err := orch.databaseCreate(c, &input)
	if err != nil {
		return handleError(c, err)
	}

	return c.Render(200, r.JSON(resp))
}

// DatabasesPut modifies a database in a given account
func DatabasesPut(c buffalo.Context) error {
	input := DatabaseModifyInput{}
	if err := c.Bind(&input); err != nil {
		log.Println(err)
		return c.Error(400, err)
	}

	if input.Cluster == nil && input.Instance == nil && input.Tags == nil {
		return c.Error(400, errors.New("Bad request: missing Cluster, Instance, Tags in request"))
	}

	if input.Cluster != nil && input.Instance != nil {
		return c.Error(400, errors.New("Bad request: cannot specify both Cluster and Instance"))
	}

	rdsClient, ok := RDS[c.Param("account")]
	if !ok {
		return c.Error(400, errors.New("Bad request: unknown account "+c.Param("account")))
	}

	orch := &rdsOrchestrator{
		client: rdsClient,
	}

	resp, err := orch.databaseModify(c, c.Param("db"), &input)
	if err != nil {
		return handleError(c, err)
	}

	return c.Render(200, r.JSON(resp))
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
func DatabasesDelete(c buffalo.Context) error {
	snapshot := false
	if b, err := strconv.ParseBool(c.Param("snapshot")); err == nil {
		snapshot = b
	}

	rdsClient, ok := RDS[c.Param("account")]
	if !ok {
		return c.Error(400, errors.New("Bad request: unknown account "+c.Param("account")))
	}

	orch := &rdsOrchestrator{
		client: rdsClient,
	}

	resp, err := orch.databaseDelete(c, c.Param("db"), snapshot)
	if err != nil {
		return handleError(c, err)
	}

	return c.Render(200, r.JSON(resp))
}
