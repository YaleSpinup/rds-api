package rds

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/gobuffalo/buffalo"
)

var ctx = &buffalo.DefaultContext{}

func (m mockRDSClient) DescribeDBClusterSnapshotsWithContext(_ aws.Context, ri *rds.DescribeDBClusterSnapshotsInput, _ ...request.Option) (*rds.DescribeDBClusterSnapshotsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	if *ri.DBClusterSnapshotIdentifier == "" {
		return &rds.DescribeDBClusterSnapshotsOutput{}, nil
	}

	return &rds.DescribeDBClusterSnapshotsOutput{DBClusterSnapshots: []*rds.DBClusterSnapshot{{Engine: aws.String("postgres"), EngineVersion: aws.String("14.5")}}}, nil
}

func (m mockRDSClient) DescribeDBEngineVersionsWithContext(aws.Context, *rds.DescribeDBEngineVersionsInput, ...request.Option) (*rds.DescribeDBEngineVersionsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &rds.DescribeDBEngineVersionsOutput{DBEngineVersions: []*rds.DBEngineVersion{{Engine: aws.String("postgres"), EngineVersion: aws.String("14.5")}}}, nil
}

func (m mockRDSClient) DescribeDBSnapshotsWithContext(ctx aws.Context, input *rds.DescribeDBSnapshotsInput, opts ...request.Option) (*rds.DescribeDBSnapshotsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	if *input.DBSnapshotIdentifier == "" {
		return &rds.DescribeDBSnapshotsOutput{}, nil
	}
	return &rds.DescribeDBSnapshotsOutput{DBSnapshots: []*rds.DBSnapshot{{Engine: aws.String("postgres"), EngineVersion: aws.String("14.5")}}}, nil
}

func (m *mockRDSClient) ModifyDBSnapshotWithContext(_ aws.Context, r *rds.ModifyDBSnapshotInput, _ ...request.Option) (*rds.ModifyDBSnapshotOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &rds.ModifyDBSnapshotOutput{DBSnapshot: &rds.DBSnapshot{DBSnapshotIdentifier: r.DBSnapshotIdentifier, EngineVersion: r.EngineVersion}}, nil
}

func TestClient_GetSnapshotInfo(t *testing.T) {

	type fields struct {
		Service                            rdsiface.RDSAPI
		DefaultSubnetGroup                 string
		DefaultDBParameterGroupName        map[string]string
		DefaultDBClusterParameterGroupName map[string]string
	}
	type args struct {
		c          buffalo.Context
		snapshotId string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *SnapshotInfo
		wantErr bool
	}{
		{
			name:    "success case",
			args:    args{c: ctx, snapshotId: "rds:awesomedatabase-2022-11-24-03-29"},
			fields:  fields{Service: newmockRDSClient(t, nil)},
			want:    &SnapshotInfo{Engine: "postgres", EngineVersion: "14.5"},
			wantErr: false,
		},
		{
			name:    "aws error",
			args:    args{c: ctx, snapshotId: "rds:awesomedatabase-2022-11-24-03-29"},
			fields:  fields{Service: newmockRDSClient(t, awserr.New("Bad Request", "boom.", nil))},
			wantErr: true,
		},
		{
			name:    "aws db not found error",
			args:    args{c: ctx, snapshotId: "rds:awesomedatabase-2022-11-24-03-29"},
			fields:  fields{Service: newmockRDSClient(t, awserr.New(rds.ErrCodeDBClusterSnapshotNotFoundFault, "not found.", nil))},
			wantErr: true,
		},
		{
			name:    "nil input",
			args:    args{c: ctx},
			fields:  fields{Service: newmockRDSClient(t, nil)},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Client{
				Service:                            tt.fields.Service,
				DefaultSubnetGroup:                 tt.fields.DefaultSubnetGroup,
				DefaultDBParameterGroupName:        tt.fields.DefaultDBParameterGroupName,
				DefaultDBClusterParameterGroupName: tt.fields.DefaultDBClusterParameterGroupName,
			}
			got, err := r.GetSnapshotInfo(tt.args.c, tt.args.snapshotId)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.GetSnapshotInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.GetSnapshotInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_DescribeDBEngineVersions(t *testing.T) {
	type fields struct {
		Service                            rdsiface.RDSAPI
		DefaultSubnetGroup                 string
		DefaultDBParameterGroupName        map[string]string
		DefaultDBClusterParameterGroupName map[string]string
	}
	type args struct {
		ctx           buffalo.Context
		engine        string
		engineVersion string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*rds.DBEngineVersion
		wantErr bool
	}{
		{
			name:    "success case",
			args:    args{ctx: ctx, engine: "postgres", engineVersion: "14.5"},
			fields:  fields{Service: newmockRDSClient(t, nil)},
			want:    []*rds.DBEngineVersion{{Engine: aws.String("postgres"), EngineVersion: aws.String("14.5")}},
			wantErr: false,
		},
		{
			name:    "aws error",
			args:    args{ctx: ctx, engine: "postgres", engineVersion: "14.5"},
			fields:  fields{Service: newmockRDSClient(t, awserr.New("Bad Request", "boom.", nil))},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Client{
				Service:                            tt.fields.Service,
				DefaultSubnetGroup:                 tt.fields.DefaultSubnetGroup,
				DefaultDBParameterGroupName:        tt.fields.DefaultDBParameterGroupName,
				DefaultDBClusterParameterGroupName: tt.fields.DefaultDBClusterParameterGroupName,
			}
			got, err := r.DescribeDBEngineVersions(tt.args.ctx, tt.args.engine, tt.args.engineVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.DescribeDBEngineVersions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.DescribeDBEngineVersions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_ModifyDBSnapshot(t *testing.T) {
	type fields struct {
		Service                            rdsiface.RDSAPI
		DefaultSubnetGroup                 string
		DefaultDBParameterGroupName        map[string]string
		DefaultDBClusterParameterGroupName map[string]string
	}
	type args struct {
		ctx           buffalo.Context
		snap          string
		engineversion string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *rds.DBSnapshot
		wantErr bool
	}{
		{
			name:    "success case",
			args:    args{ctx: &buffalo.DefaultContext{}, snap: "1234", engineversion: "14.5"},
			fields:  fields{Service: newmockRDSClient(t, nil)},
			want:    &rds.DBSnapshot{DBSnapshotIdentifier: aws.String("1234"), EngineVersion: aws.String("14.5")},
			wantErr: false,
		},
		{
			name:    "aws error",
			args:    args{ctx: &buffalo.DefaultContext{}, snap: "1234", engineversion: "14.5"},
			fields:  fields{Service: newmockRDSClient(t, awserr.New("Bad Request", "boom.", nil))},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Client{
				Service:                            tt.fields.Service,
				DefaultSubnetGroup:                 tt.fields.DefaultSubnetGroup,
				DefaultDBParameterGroupName:        tt.fields.DefaultDBParameterGroupName,
				DefaultDBClusterParameterGroupName: tt.fields.DefaultDBClusterParameterGroupName,
			}
			got, err := r.ModifyDBSnapshot(tt.args.ctx, tt.args.snap, tt.args.engineversion)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.ModifyDBSnapshot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.ModifyDBSnapshot() = %v, want %v", got, tt.want)
			}
		})
	}
}
