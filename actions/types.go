package actions

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
)

type DatabaseCreateRequest struct {
	Cluster  *CreateDBClusterInput
	Instance *CreateDBInstanceInput
}

func (dcr DatabaseCreateRequest) String() string {
	s, _ := json.MarshalIndent(dcr, "", "\t")
	return string(s)
}

type SnapshotCreateRequest struct {
	SnapshotIdentifier string
}

type SnapshotModifyRequest struct {
	EngineVersion string
}

// CreateDBInstanceInput is the input for creating a new database instance
// based on https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#CreateDBInstanceInput
type CreateDBInstanceInput struct {
	AllocatedStorage            *int64
	BackupRetentionPeriod       *int64
	DBClusterIdentifier         *string
	DBInstanceClass             *string
	DBInstanceIdentifier        *string
	DBParameterGroupName        *string
	DBSubnetGroupName           *string
	EnableCloudwatchLogsExports []*string
	Engine                      *string
	EngineVersion               *string
	LicenseModel                *string
	MasterUserPassword          *string
	MasterUsername              *string
	MultiAZ                     *bool
	Port                        *int64
	SnapshotIdentifier          *string
	StorageEncrypted            *bool
	Tags                        []*Tag
	VpcSecurityGroupIds         []*string
}

// CreateDBClusterInput is the input for creating a new database cluster
// based on https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#CreateDBClusterInput
type CreateDBClusterInput struct {
	BackupRetentionPeriod       *int64
	DBClusterIdentifier         *string
	DBClusterParameterGroupName *string
	DBSubnetGroupName           *string
	EnableCloudwatchLogsExports []*string
	Engine                      *string
	EngineMode                  *string
	EngineVersion               *string
	MasterUserPassword          *string
	MasterUsername              *string
	Port                        *int64
	ScalingConfiguration        *ScalingConfiguration
	SnapshotIdentifier          *string
	StorageEncrypted            *bool
	Tags                        []*Tag
	VpcSecurityGroupIds         []*string
}

type ScalingConfiguration struct {
	AutoPause             *bool
	MaxCapacity           *int64
	MinCapacity           *int64
	SecondsUntilAutoPause *int64
	TimeoutAction         *string
}

type Tag struct {
	Key   *string
	Value *string
}

// DatabaseResponse is the output from database operations
type DatabaseResponse struct {
	// https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#DBCluster
	Cluster *rds.DBCluster
	// https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#DBInstance
	Instance *rds.DBInstance
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

// normalizeTags strips the org from the given tags and ensures it is set to the API org
func normalizeTags(tags []*Tag) []*Tag {
	normalizedTags := []*Tag{}
	for _, t := range tags {
		if aws.StringValue(t.Key) == "spinup:org" || aws.StringValue(t.Key) == "yale:org" {
			continue
		}
		normalizedTags = append(normalizedTags, t)
	}

	normalizedTags = append(normalizedTags,
		&Tag{
			Key:   aws.String("spinup:org"),
			Value: aws.String(Org),
		})

	return normalizedTags
}

// fromRDSTags converts from RDS tags to api Tags
func fromRDSTags(ecrTags []*rds.Tag) []*Tag {
	tags := make([]*Tag, 0, len(ecrTags))
	for _, t := range ecrTags {
		tags = append(tags, &Tag{
			Key:   t.Key,
			Value: t.Value,
		})
	}
	return tags
}

// toRDSTags converts from api Tags to RDS tags
func toRDSTags(tags []*Tag) []*rds.Tag {
	rdsTags := make([]*rds.Tag, 0, len(tags))
	for _, t := range tags {
		rdsTags = append(rdsTags, &rds.Tag{
			Key:   t.Key,
			Value: t.Value,
		})
	}
	return rdsTags
}
