# rds-api

This API provides simple restful API access to Amazon's RDS service.

## Usage

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
      "DBInstanceClass":"db.t2.micro",
      "DBInstanceIdentifier":"mypostgres",
      "Engine":"postgres",
      "MasterUserPassword":"MyPassword",
      "MasterUsername":"MyUser",
      "PubliclyAccessible":false,
      "StorageEncrypted":false
   }
}
```

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
      "StorageEncrypted":false,
      "VpcSecurityGroupIds":[
         "sg-12345678"
      ]
   },
   "Instance":{
      "DBClusterIdentifier":"myaurora",
      "DBInstanceClass":"db.t2.small",
      "DBInstanceIdentifier":"myaurora-1",
      "Engine":"aurora",
      "PubliclyAccessible":false,
      "StorageEncrypted":false
   }
}
```

### Getting details about a database

To get details about a specific database instance:

```
GET http://127.0.0.1:3000/v1/rds/{account}/mypostgres
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
      ...
    }
  ]
}
```

To get details about _all_ database instances in the given account (to list both database instances and clusters you can add `all=true` query parameter):

```
GET http://127.0.0.1:3000/v1/rds/{account}[?all=true]
```

### Deleting a database

By default, a final snapshot is not created when deleting a database instance. You can request that by adding `snapshot=true` query parameter.

```
DELETE http://127.0.0.1:3000/v1/rds/{account}/mypostgres[?snapshot=true]
```

## Development

- Install Buffalo framework (v0.12+): https://gobuffalo.io/en/docs/installation
- Run `buffalo dev` to start the app locally
- Run `buffalo tests -v` to run all tests

## Authors

Tenyo Grozev <tenyo.grozev@yale.edu>

[Powered by Buffalo](http://gobuffalo.io)
