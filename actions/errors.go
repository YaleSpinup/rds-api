package actions

import (
	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func ErrCode(msg string, err error) error {
	log.Debugf("processing error code with message '%s' and error '%s'", msg, err)

	if aerr, ok := errors.Cause(err).(awserr.Error); ok {
		switch aerr.Code() {
		case
			// ErrCodeInsufficientDBClusterCapacityFault for service response error code
			// "InsufficientDBClusterCapacityFault".
			//
			// The DB cluster doesn't have enough capacity for the current operation.
			rds.ErrCodeInsufficientDBClusterCapacityFault,

			// ErrCodeInsufficientDBInstanceCapacityFault for service response error code
			// "InsufficientDBInstanceCapacity".
			//
			// The specified DB instance class isn't available in the specified Availability
			// Zone.
			rds.ErrCodeInsufficientDBInstanceCapacityFault,

			// ErrCodeInsufficientStorageClusterCapacityFault for service response error code
			// "InsufficientStorageClusterCapacity".
			//
			// There is insufficient storage available for the current action. You might
			// be able to resolve this error by updating your subnet group to use different
			// Availability Zones that have more storage available.
			rds.ErrCodeInsufficientStorageClusterCapacityFault,

			// ErrCodeInvalidDBClusterEndpointStateFault for service response error code
			// "InvalidDBClusterEndpointStateFault".
			//
			// The requested operation can't be performed on the endpoint while the endpoint
			// is in this state.
			rds.ErrCodeInvalidDBClusterEndpointStateFault,

			// ErrCodeInvalidDBClusterSnapshotStateFault for service response error code
			// "InvalidDBClusterSnapshotStateFault".
			//
			// The supplied value isn't a valid DB cluster snapshot state.
			rds.ErrCodeInvalidDBClusterSnapshotStateFault,

			// ErrCodeInvalidDBClusterStateFault for service response error code
			// "InvalidDBClusterStateFault".
			//
			// The requested operation can't be performed while the cluster is in this state.
			rds.ErrCodeInvalidDBClusterStateFault,

			// ErrCodeInvalidDBInstanceAutomatedBackupStateFault for service response error code
			// "InvalidDBInstanceAutomatedBackupState".
			//
			// The automated backup is in an invalid state. For example, this automated
			// backup is associated with an active instance.
			rds.ErrCodeInvalidDBInstanceAutomatedBackupStateFault,

			// ErrCodeInvalidDBInstanceStateFault for service response error code
			// "InvalidDBInstanceState".
			//
			// The DB instance isn't in a valid state.
			rds.ErrCodeInvalidDBInstanceStateFault,

			// ErrCodeInvalidDBParameterGroupStateFault for service response error code
			// "InvalidDBParameterGroupState".
			//
			// The DB parameter group is in use or is in an invalid state. If you are attempting
			// to delete the parameter group, you can't delete it when the parameter group
			// is in this state.
			rds.ErrCodeInvalidDBParameterGroupStateFault,

			// ErrCodeInvalidDBProxyStateFault for service response error code
			// "InvalidDBProxyStateFault".
			//
			// The requested operation can't be performed while the proxy is in this state.
			rds.ErrCodeInvalidDBProxyStateFault,

			// ErrCodeInvalidDBSecurityGroupStateFault for service response error code
			// "InvalidDBSecurityGroupState".
			//
			// The state of the DB security group doesn't allow deletion.
			rds.ErrCodeInvalidDBSecurityGroupStateFault,

			// ErrCodeInvalidDBSnapshotStateFault for service response error code
			// "InvalidDBSnapshotState".
			//
			// The state of the DB snapshot doesn't allow deletion.
			rds.ErrCodeInvalidDBSnapshotStateFault,

			// ErrCodeInvalidDBSubnetGroupFault for service response error code
			// "InvalidDBSubnetGroupFault".
			//
			// The DBSubnetGroup doesn't belong to the same VPC as that of an existing cross-region
			// read replica of the same source instance.
			rds.ErrCodeInvalidDBSubnetGroupFault,

			// ErrCodeInvalidDBSubnetGroupStateFault for service response error code
			// "InvalidDBSubnetGroupStateFault".
			//
			// The DB subnet group cannot be deleted because it's in use.
			rds.ErrCodeInvalidDBSubnetGroupStateFault,

			// ErrCodeInvalidDBSubnetStateFault for service response error code
			// "InvalidDBSubnetStateFault".
			//
			// The DB subnet isn't in the available state.
			rds.ErrCodeInvalidDBSubnetStateFault,

			// ErrCodeInvalidEventSubscriptionStateFault for service response error code
			// "InvalidEventSubscriptionState".
			//
			// This error can occur if someone else is modifying a subscription. You should
			// retry the action.
			rds.ErrCodeInvalidEventSubscriptionStateFault,

			// ErrCodeInvalidExportOnlyFault for service response error code
			// "InvalidExportOnly".
			//
			// The export is invalid for exporting to an Amazon S3 bucket.
			rds.ErrCodeInvalidExportOnlyFault,

			// ErrCodeInvalidExportSourceStateFault for service response error code
			// "InvalidExportSourceState".
			//
			// The state of the export snapshot is invalid for exporting to an Amazon S3
			// bucket.
			rds.ErrCodeInvalidExportSourceStateFault,

			// ErrCodeInvalidExportTaskStateFault for service response error code
			// "InvalidExportTaskStateFault".
			//
			// You can't cancel an export task that has completed.
			rds.ErrCodeInvalidExportTaskStateFault,

			// ErrCodeInvalidGlobalClusterStateFault for service response error code
			// "InvalidGlobalClusterStateFault".
			//
			// The global cluster is in an invalid state and can't perform the requested
			// operation.
			rds.ErrCodeInvalidGlobalClusterStateFault,

			// ErrCodeInvalidOptionGroupStateFault for service response error code
			// "InvalidOptionGroupStateFault".
			//
			// The option group isn't in the available state.
			rds.ErrCodeInvalidOptionGroupStateFault,

			// ErrCodeInvalidRestoreFault for service response error code
			// "InvalidRestoreFault".
			//
			// Cannot restore from VPC backup to non-VPC DB instance.
			rds.ErrCodeInvalidRestoreFault,

			// ErrCodeInvalidS3BucketFault for service response error code
			// "InvalidS3BucketFault".
			//
			// The specified Amazon S3 bucket name can't be found or Amazon RDS isn't authorized
			// to access the specified Amazon S3 bucket. Verify the SourceS3BucketName and
			// S3IngestionRoleArn values and try again.
			rds.ErrCodeInvalidS3BucketFault,

			// ErrCodeInvalidVPCNetworkStateFault for service response error code
			// "InvalidVPCNetworkStateFault".
			//
			// The DB subnet group doesn't cover all Availability Zones after it's created
			// because of users' change.
			rds.ErrCodeInvalidVPCNetworkStateFault,

			// ErrCodeKMSKeyNotAccessibleFault for service response error code
			// "KMSKeyNotAccessibleFault".
			//
			// An error occurred accessing an AWS KMS key.
			rds.ErrCodeKMSKeyNotAccessibleFault,

			// ErrCodePointInTimeRestoreNotEnabledFault for service response error code
			// "PointInTimeRestoreNotEnabled".
			//
			// SourceDBInstanceIdentifier refers to a DB instance with BackupRetentionPeriod
			// equal to 0.
			rds.ErrCodePointInTimeRestoreNotEnabledFault,

			// ErrCodeProvisionedIopsNotAvailableInAZFault for service response error code
			// "ProvisionedIopsNotAvailableInAZFault".
			//
			// Provisioned IOPS not available in the specified Availability Zone.
			rds.ErrCodeProvisionedIopsNotAvailableInAZFault,

			// ErrCodeStorageTypeNotSupportedFault for service response error code
			// "StorageTypeNotSupported".
			//
			// Storage of the StorageType specified can't be associated with the DB instance.
			rds.ErrCodeStorageTypeNotSupportedFault:

			return apierror.New(apierror.ErrInternalError, msg, err)
		case
			// ErrCodeAuthorizationAlreadyExistsFault for service response error code
			// "AuthorizationAlreadyExists".
			//
			// The specified CIDR IP range or Amazon EC2 security group is already authorized
			// for the specified DB security group.
			rds.ErrCodeAuthorizationAlreadyExistsFault,

			// ErrCodeCustomAvailabilityZoneAlreadyExistsFault for service response error code
			// "CustomAvailabilityZoneAlreadyExists".
			//
			// CustomAvailabilityZoneName is already used by an existing custom Availability
			// Zone.
			rds.ErrCodeCustomAvailabilityZoneAlreadyExistsFault,

			// ErrCodeDBClusterAlreadyExistsFault for service response error code
			// "DBClusterAlreadyExistsFault".
			//
			// The user already has a DB cluster with the given identifier.
			rds.ErrCodeDBClusterAlreadyExistsFault,

			// ErrCodeDBClusterEndpointAlreadyExistsFault for service response error code
			// "DBClusterEndpointAlreadyExistsFault".
			//
			// The specified custom endpoint can't be created because it already exists.
			rds.ErrCodeDBClusterEndpointAlreadyExistsFault,

			// ErrCodeDBClusterRoleAlreadyExistsFault for service response error code
			// "DBClusterRoleAlreadyExists".
			//
			// The specified IAM role Amazon Resource Name (ARN) is already associated with
			// the specified DB cluster.
			rds.ErrCodeDBClusterRoleAlreadyExistsFault,

			// ErrCodeDBClusterSnapshotAlreadyExistsFault for service response error code
			// "DBClusterSnapshotAlreadyExistsFault".
			//
			// The user already has a DB cluster snapshot with the given identifier.
			rds.ErrCodeDBClusterSnapshotAlreadyExistsFault,

			// ErrCodeDBInstanceAlreadyExistsFault for service response error code
			// "DBInstanceAlreadyExists".
			//
			// The user already has a DB instance with the given identifier.
			rds.ErrCodeDBInstanceAlreadyExistsFault,

			// ErrCodeDBInstanceRoleAlreadyExistsFault for service response error code
			// "DBInstanceRoleAlreadyExists".
			//
			// The specified RoleArn or FeatureName value is already associated with the
			// DB instance.
			rds.ErrCodeDBInstanceRoleAlreadyExistsFault,

			// ErrCodeDBParameterGroupAlreadyExistsFault for service response error code
			// "DBParameterGroupAlreadyExists".
			//
			// A DB parameter group with the same name exists.
			rds.ErrCodeDBParameterGroupAlreadyExistsFault,

			// ErrCodeDBProxyAlreadyExistsFault for service response error code
			// "DBProxyTargetExistsFault".
			//
			// The specified proxy name must be unique for all proxies owned by your AWS
			// account in the specified AWS Region.
			rds.ErrCodeDBProxyAlreadyExistsFault,

			// ErrCodeDBSecurityGroupAlreadyExistsFault for service response error code
			// "DBSecurityGroupAlreadyExists".
			//
			// A DB security group with the name specified in DBSecurityGroupName already
			// exists.
			rds.ErrCodeDBSecurityGroupAlreadyExistsFault,

			// ErrCodeDBSnapshotAlreadyExistsFault for service response error code
			// "DBSnapshotAlreadyExists".
			//
			// DBSnapshotIdentifier is already used by an existing snapshot.
			rds.ErrCodeDBSnapshotAlreadyExistsFault,

			// ErrCodeDBSubnetGroupAlreadyExistsFault for service response error code
			// "DBSubnetGroupAlreadyExists".
			//
			// DBSubnetGroupName is already used by an existing DB subnet group.
			rds.ErrCodeDBSubnetGroupAlreadyExistsFault,

			// ErrCodeExportTaskAlreadyExistsFault for service response error code
			// "ExportTaskAlreadyExists".
			//
			// You can't start an export task that's already running.
			rds.ErrCodeExportTaskAlreadyExistsFault,

			// ErrCodeGlobalClusterAlreadyExistsFault for service response error code
			// "GlobalClusterAlreadyExistsFault".
			//
			// The GlobalClusterIdentifier already exists. Choose a new global database
			// identifier (unique name) to create a new global database cluster.
			rds.ErrCodeGlobalClusterAlreadyExistsFault,

			// ErrCodeOptionGroupAlreadyExistsFault for service response error code
			// "OptionGroupAlreadyExistsFault".
			//
			// The option group you are trying to create already exists.
			rds.ErrCodeOptionGroupAlreadyExistsFault,

			// ErrCodeReservedDBInstanceAlreadyExistsFault for service response error code
			// "ReservedDBInstanceAlreadyExists".
			//
			// User already has a reservation with the given identifier.
			rds.ErrCodeReservedDBInstanceAlreadyExistsFault,

			// ErrCodeSubnetAlreadyInUse for service response error code
			// "SubnetAlreadyInUse".
			//
			// The DB subnet is already in use in the Availability Zone.
			rds.ErrCodeSubnetAlreadyInUse,

			// ErrCodeSubscriptionAlreadyExistFault for service response error code
			// "SubscriptionAlreadyExist".
			//
			// The supplied subscription name already exists.
			rds.ErrCodeSubscriptionAlreadyExistFault:

			return apierror.New(apierror.ErrConflict, msg, aerr)
		case
			// ErrCodeDBClusterNotFoundFault for service response error code
			// "DBClusterNotFoundFault".
			//
			// DBClusterIdentifier doesn't refer to an existing DB cluster.
			rds.ErrCodeDBClusterNotFoundFault,

			// ErrCodeDBClusterSnapshotNotFoundFault for service response error code
			// "DBClusterSnapshotNotFoundFault".
			//
			// DBClusterSnapshotIdentifier doesn't refer to an existing DB cluster snapshot.
			rds.ErrCodeDBClusterSnapshotNotFoundFault,

			// ErrCodeDBInstanceAutomatedBackupNotFoundFault for service response error code
			// "DBInstanceAutomatedBackupNotFound".
			//
			// No automated backup for this DB instance was found.
			rds.ErrCodeDBInstanceAutomatedBackupNotFoundFault,

			// ErrCodeDBInstanceNotFoundFault for service response error code
			// "DBInstanceNotFound".
			//
			// DBInstanceIdentifier doesn't refer to an existing DB instance.
			rds.ErrCodeDBInstanceNotFoundFault,

			// ErrCodeGlobalClusterNotFoundFault for service response error code
			// "GlobalClusterNotFoundFault".
			//
			// The GlobalClusterIdentifier doesn't refer to an existing global database
			// cluster.
			rds.ErrCodeGlobalClusterNotFoundFault,

			// ErrCodeReservedDBInstanceNotFoundFault for service response error code
			// "ReservedDBInstanceNotFound".
			//
			// The specified reserved DB Instance not found.
			rds.ErrCodeReservedDBInstanceNotFoundFault,

			// ErrCodeReservedDBInstancesOfferingNotFoundFault for service response error code
			// "ReservedDBInstancesOfferingNotFound".
			//
			// Specified offering does not exist.
			rds.ErrCodeReservedDBInstancesOfferingNotFoundFault,

			// ErrCodeResourceNotFoundFault for service response error code
			// "ResourceNotFoundFault".
			//
			// The specified resource ID was not found.
			rds.ErrCodeResourceNotFoundFault:

			return apierror.New(apierror.ErrNotFound, msg, aerr)
		case
			// ErrCodeAuthorizationQuotaExceededFault for service response error code
			// "AuthorizationQuotaExceeded".
			//
			// The DB security group authorization quota has been reached.
			rds.ErrCodeAuthorizationQuotaExceededFault,

			// ErrCodeCustomAvailabilityZoneQuotaExceededFault for service response error code
			// "CustomAvailabilityZoneQuotaExceeded".
			//
			// You have exceeded the maximum number of custom Availability Zones.
			rds.ErrCodeCustomAvailabilityZoneQuotaExceededFault,

			// ErrCodeDBClusterEndpointQuotaExceededFault for service response error code
			// "DBClusterEndpointQuotaExceededFault".
			//
			// The cluster already has the maximum number of custom endpoints.
			rds.ErrCodeDBClusterEndpointQuotaExceededFault,

			// ErrCodeDBClusterQuotaExceededFault for service response error code
			// "DBClusterQuotaExceededFault".
			//
			// The user attempted to create a new DB cluster and the user has already reached
			// the maximum allowed DB cluster quota.
			rds.ErrCodeDBClusterQuotaExceededFault,

			// ErrCodeDBClusterRoleQuotaExceededFault for service response error code
			// "DBClusterRoleQuotaExceeded".
			//
			// You have exceeded the maximum number of IAM roles that can be associated
			// with the specified DB cluster.
			rds.ErrCodeDBClusterRoleQuotaExceededFault,

			// ErrCodeDBInstanceAutomatedBackupQuotaExceededFault for service response error code
			// "DBInstanceAutomatedBackupQuotaExceeded".
			//
			// The quota for retained automated backups was exceeded. This prevents you
			// from retaining any additional automated backups. The retained automated backups
			// quota is the same as your DB Instance quota.
			rds.ErrCodeDBInstanceAutomatedBackupQuotaExceededFault,

			// ErrCodeDBInstanceRoleQuotaExceededFault for service response error code
			// "DBInstanceRoleQuotaExceeded".
			//
			// You can't associate any more AWS Identity and Access Management (IAM) roles
			// with the DB instance because the quota has been reached.
			rds.ErrCodeDBInstanceRoleQuotaExceededFault,

			// ErrCodeDBParameterGroupQuotaExceededFault for service response error code
			// "DBParameterGroupQuotaExceeded".
			//
			// The request would result in the user exceeding the allowed number of DB parameter
			// groups.
			rds.ErrCodeDBParameterGroupQuotaExceededFault,

			// ErrCodeDBSecurityGroupQuotaExceededFault for service response error code
			// "QuotaExceeded.DBSecurityGroup".
			//
			// The request would result in the user exceeding the allowed number of DB security
			// groups.
			rds.ErrCodeDBSecurityGroupQuotaExceededFault,

			// ErrCodeDBSubnetGroupQuotaExceededFault for service response error code
			// "DBSubnetGroupQuotaExceeded".
			//
			// The request would result in the user exceeding the allowed number of DB subnet
			// groups.
			rds.ErrCodeDBSubnetGroupQuotaExceededFault,

			// ErrCodeDBSubnetQuotaExceededFault for service response error code
			// "DBSubnetQuotaExceededFault".
			//
			// The request would result in the user exceeding the allowed number of subnets
			// in a DB subnet groups.
			rds.ErrCodeDBSubnetQuotaExceededFault,

			// ErrCodeEventSubscriptionQuotaExceededFault for service response error code
			// "EventSubscriptionQuotaExceeded".
			//
			// You have reached the maximum number of event subscriptions.
			rds.ErrCodeEventSubscriptionQuotaExceededFault,

			// ErrCodeGlobalClusterQuotaExceededFault for service response error code
			// "GlobalClusterQuotaExceededFault".
			//
			// The number of global database clusters for this account is already at the
			// maximum allowed.
			rds.ErrCodeGlobalClusterQuotaExceededFault,

			// ErrCodeInstanceQuotaExceededFault for service response error code
			// "InstanceQuotaExceeded".
			//
			// The request would result in the user exceeding the allowed number of DB instances.
			rds.ErrCodeInstanceQuotaExceededFault,

			// ErrCodeOptionGroupQuotaExceededFault for service response error code
			// "OptionGroupQuotaExceededFault".
			//
			// The quota of 20 option groups was exceeded for this AWS account.
			rds.ErrCodeOptionGroupQuotaExceededFault,

			// ErrCodeReservedDBInstanceQuotaExceededFault for service response error code
			// "ReservedDBInstanceQuotaExceeded".
			//
			// Request would exceed the user's DB Instance quota.
			rds.ErrCodeReservedDBInstanceQuotaExceededFault,

			// ErrCodeSharedSnapshotQuotaExceededFault for service response error code
			// "SharedSnapshotQuotaExceeded".
			//
			// You have exceeded the maximum number of accounts that you can share a manual
			// DB snapshot with.
			rds.ErrCodeSharedSnapshotQuotaExceededFault,

			// ErrCodeSnapshotQuotaExceededFault for service response error code
			// "SnapshotQuotaExceeded".
			//
			// The request would result in the user exceeding the allowed number of DB snapshots.
			rds.ErrCodeSnapshotQuotaExceededFault,

			// ErrCodeStorageQuotaExceededFault for service response error code
			// "StorageQuotaExceeded".
			//
			// The request would result in the user exceeding the allowed amount of storage
			// available across all DB instances.
			rds.ErrCodeStorageQuotaExceededFault:

			return apierror.New(apierror.ErrLimitExceeded, msg, aerr)
		default:
			m := msg + ": " + aerr.Message()
			return apierror.New(apierror.ErrBadRequest, m, aerr)
		}
	}

	return apierror.New(apierror.ErrInternalError, msg, err)
}
