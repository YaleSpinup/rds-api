package rds

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
)

type mockRdsService struct {
	rdsiface.RDSAPI
}

func (m *mockRdsService) DescribeDBClusters(input *rds.DescribeDBClustersInput) (*rds.DescribeDBClustersOutput, error) {
	dbName := *input.DBClusterIdentifier

	if dbName == "unknown" || dbName == "instance" {
		return &rds.DescribeDBClustersOutput{}, nil
	}

	dbClusters := make([]*rds.DBCluster, 1)
	dbClusters[0] = &rds.DBCluster{
		DBClusterArn: aws.String("arn:aws:rds:us-east-1:123456789012:db:" + dbName),
	}
	return &rds.DescribeDBClustersOutput{
		DBClusters: dbClusters,
	}, nil
}

func (m *mockRdsService) DescribeDBInstances(input *rds.DescribeDBInstancesInput) (*rds.DescribeDBInstancesOutput, error) {
	dbName := *input.DBInstanceIdentifier

	if dbName == "unknown" || dbName == "cluster" {
		return &rds.DescribeDBInstancesOutput{}, nil
	}

	dbInstances := make([]*rds.DBInstance, 1)
	dbInstances[0] = &rds.DBInstance{
		DBInstanceArn: aws.String("arn:aws:rds:us-east-1:123456789012:db:" + dbName),
	}

	return &rds.DescribeDBInstancesOutput{
		DBInstances: dbInstances,
	}, nil
}

func TestDetermineArn(t *testing.T) {
	mc := Client{
		Service: &mockRdsService{},
	}

	got, err := mc.DetermineArn("cluster")
	t.Log(got)
	if err != nil {
		t.Fatalf("Expected error nil, got: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Expected single ARN, got: %v", len(got))
	}

	got, err = mc.DetermineArn("instance")
	t.Log(got)
	if err != nil {
		t.Fatalf("Expected error nil, got: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Expected single ARN, got: %v", len(got))
	}

	got, err = mc.DetermineArn("both")
	t.Log(got)
	if err != nil {
		t.Fatalf("Expected error nil, got: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Expected two ARNs, got: %v", len(got))
	}

	got, err = mc.DetermineArn("unknown")
	t.Log(got)
	if err == nil {
		t.Fatalf("Expected error, got: nil")
	}
}
