# rds-api

This API provides simple restful API access to Amazon's RDS service.

## Usage

This API uses the standard format for input and output of parameters as defined by the AWS SDK (for reference see https://docs.aws.amazon.com/sdk-for-go/api/service/rds/).

### Config

You can define multiple _accounts_ in your `config.json` file which are mapped to endpoints by the API and allow RDS instances to be created in different AWS accounts. See [example config](config/config.example.json)

In each account you can optionally define certain defaults that will be used if those parameters are not specified in the POST request when creating a database:
  - `defaultSubnetGroup` - the subnet group that will be used if one is not given
  - `defaultDBParameterGroupName` - map of ParameterGroupFamily to ParameterGroupName's
  - `defaultDBClusterParameterGroupName` - map of ParameterGroupFamily to ClusterParameterGroupName's

_Note that any default parameters need to refer to existing resources (groups), i.e. they need to be created separately outside of this API._

### Authentication

Authentication is accomplished via a pre-shared key (hashed string) in the `X-Auth-Token` header.

### Creating a database

You can specify both database cluster and instance information in the POST to create just an instance or a cluster and a member instance. 

For example, to create a single Postgres database instance:

```
POST http://127.0.0.1:3000/v1/rds/{account}
{
   "Instance":{
      "AllocatedStorage":20,
      "AutoMinorVersionUpgrade":true,
      "BackupRetentionPeriod":0,
      "DBInstanceClass":"db.t2.small",
      "DBInstanceIdentifier":"mypostgres",
      "Engine":"postgres",
      "MasterUserPassword":"MyPassword",
      "MasterUsername":"MyUser",
      "PubliclyAccessible":false,
      "StorageEncrypted":true
   }
}
```

All instance creation parameters are listed at https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#CreateDBInstanceInput

To create an Aurora cluster with one database instance:

```
POST http://127.0.0.1:3000/v1/rds/{account}
{
   "Cluster":{
      "AutoMinorVersionUpgrade":true,
      "BackupRetentionPeriod":1,
      "DBClusterIdentifier":"myaurora",
      "Engine":"aurora",
      "MasterUserPassword":"MyPassword",
      "MasterUsername":"MyUser",
      "StorageEncrypted":true,
      "VpcSecurityGroupIds":[
         "sg-12345678"
      ]
   },
   "Instance":{
      "DBClusterIdentifier":"myaurora",
      "DBInstanceClass":"db.t2.small",
      "DBInstanceIdentifier":"myaurora-1",
      "Engine":"aurora",
      "PubliclyAccessible":false
   }
}
```

All cluster creation parameters are listed at https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#CreateDBClusterInput

To create an Aurora serverless cluster you need to pass the `EngineMode` parameter and can optionally specify custom `ScalingConfiguration`:

```
POST http://127.0.0.1:3000/v1/rds/{account}
{
   "Cluster":{
      "AutoMinorVersionUpgrade":true,
      "BackupRetentionPeriod":1,
      "DBClusterIdentifier":"myserverless",
      "Engine":"aurora-mysql",
      "EngineMode":"serverless",
      "MasterUserPassword":"MyPassword",
      "MasterUsername":"MyUser",
      "ScalingConfiguration": {
         "AutoPause": true,
         "MaxCapacity": 4,
         "MinCapacity": 1,
         "SecondsUntilAutoPause": 300
      },
      "StorageEncrypted":true,
      "VpcSecurityGroupIds":[
         "sg-12345678"
      ]
   }
}
```

### Getting details about a database

To get details about a specific database instance or cluster:

```
GET http://127.0.0.1:3000/v1/rds/{account}/mypostgres
```
```
{
  "DBInstances": [
    {
      "AllocatedStorage": 20,
      "AutoMinorVersionUpgrade": true,
      "AvailabilityZone": "us-east-1d",
      "BackupRetentionPeriod": 0,
      "DBClusterIdentifier": null,
      "DBInstanceClass": "db.t2.micro",
      "DBInstanceIdentifier": "mypostgres",
      "DBInstanceStatus": "available",
      "Endpoint": {
        "Address": "mypostgres.c8ukc5s0qnag.us-east-1.rds.amazonaws.com",
        "HostedZoneId": "Z3R2ITVGPH62AM",
        "Port": 5432
      },
      ...
    }
  ]
}
```

To get details about _all_ database instances in the given account (to list both database instances _and_ clusters you can add `all=true` query parameter):

```
GET http://127.0.0.1:3000/v1/rds/{account}[?all=true]
```

### Modifying database parameters

You can specify either cluster or instance parameters in the PUT to modify a cluster or an instance.

For example, to change the master password for an Aurora cluster:

```
PUT http://127.0.0.1:3000/v1/rds/{account}/myaurora
{
   "Cluster": {
      "MasterUserPassword": "EXAMPLE",
      "ApplyImmediately": true
   }
}
```

### Updating tags for a database

You can pass a list of tags (Key/Value pairs) to add or updated on the given database. If there is an RDS cluster and instance with the same name, the tags for both will be updated.

```
PUT http://127.0.0.1:3000/v1/rds/{account}/myaurora
{
   "Tags": [
      {
         "Key": "NewTag",
         "Value": "new"
      }
   ]
}
```

### Deleting a database

By default, a final snapshot is _not_ created when deleting a database instance. You can override that by adding `snapshot=true` query parameter.

```
DELETE http://127.0.0.1:3000/v1/rds/{account}/mypostgres[?snapshot=true]
```

The API will check if the database instance belongs to a cluster and will automatically delete the cluster if this is the last member.

### Stopping and starting a database/cluster

```
PUT http://127.0.0.1:3000/v1/rds/{account}/myaurora/power
{
   "state": "stop|start"
}
```

### Getting a list of snapshots for a database/cluster

This will return list of snapshots (with details) for the specified database in either `DBClusterSnapshots` or `DBSnapshots`, depending if it's a cluster or an instance.
It will also set an `X-Items` header containing the total number of snapshots in the list.

```
GET http://127.0.0.1:3000/v1/rds/{account}/mydbcluster/snapshots
```
```
{
    "DBClusterSnapshots": [
        {
            ...
            "DBClusterIdentifier": "mydbcluster",
            "DBClusterSnapshotIdentifier": "rds:mydbcluster-2021-07-21-08-37",
            ...
        },
        {
            ...
            "DBClusterIdentifier": "mydbcluster",
            "DBClusterSnapshotIdentifier": "rds:mydbcluster-2021-07-22-08-37",
            ...
        },
        ...
```

or

```
GET http://127.0.0.1:3000/v1/rds/{account}/mydbinstance/snapshots
```
```
{
    "DBSnapshots": [
        {
            ...
            "DBInstanceIdentifier": "mydbinstance",
            "DBSnapshotIdentifier": "rds:mydbinstance-2021-07-21-05-20",
            ...
        },
        {
            ...
            "DBInstanceIdentifier": "mydbinstance",
            "DBSnapshotIdentifier": "rds:mydbinstance-2021-07-22-05-20",
            ...
        },
        ...
```

### Getting information about a specific snapshot

This will return details about a snapshot in either `DBClusterSnapshot` or `DBSnapshot`, depending if it's a cluster or an instance snapshot.

```
GET http://127.0.0.1:3000/v1/rds/{account}/snapshots/rds:mydbinstance-2021-07-22-05-20
```
```
{
    "DBSnapshot": {
        "AllocatedStorage": 20,
        "AvailabilityZone": "us-east-1d",
        "DBInstanceIdentifier": "mydbinstance",
        "DBSnapshotArn": "arn:aws:rds:us-east-1:01234567890:snapshot:rds:mydbinstance-2021-07-22-05-20",
        "DBSnapshotIdentifier": "rds:mydbinstance-2021-07-22-05-20",
        "DbiResourceId": "db-DH55SGP1L9S4DHZ6564CAO7PUQ",
        "Encrypted": true,
        "Engine": "postgres",
        "EngineVersion": "10.15",
        "IAMDatabaseAuthenticationEnabled": false,
        "InstanceCreateTime": "2021-05-07T18:46:08.571Z",
        "Iops": null,
        "KmsKeyId": "arn:aws:kms:us-east-1:01234567890:key/32c76e50-8fab-5e15-cba4-eef7f4a042f7",
        "LicenseModel": "postgresql-license",
        "MasterUsername": "test",
        "OptionGroupName": "default:postgres-10",
        "PercentProgress": 100,
        "Port": 5432,
        "ProcessorFeatures": null,
        "SnapshotCreateTime": "2021-07-22T05:20:00.000Z",
        "SnapshotType": "automated",
        "SourceDBSnapshotIdentifier": null,
        "SourceRegion": null,
        "Status": "available",
        "StorageType": "gp2",
        "TdeCredentialArn": null,
        "Timezone": null,
        "VpcId": "vpc-01234567"
    }
}
```

## Development

- Install Buffalo framework (v0.13+): https://gobuffalo.io/en/docs/installation
- Run `buffalo setup` to install required dependencies
- Run `buffalo dev` to start the app locally
- Run `buffalo tests -v` to run all tests

To build a container locally for testing:
```
$ cd docker/
$ docker-compose up -d
$ curl http://localhost:8088/v1/rds/ping
pong
# do your work, then shut it down
$ docker-compose down
```

## Authors

Tenyo Grozev <tenyo.grozev@yale.edu>

[Powered by Buffalo](http://gobuffalo.io)
