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

func (m *mockRDSClient) ModifyDBSnapshotWithContext(_ aws.Context, r *rds.ModifyDBSnapshotInput, _ ...request.Option) (*rds.ModifyDBSnapshotOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &rds.ModifyDBSnapshotOutput{DBSnapshot: &rds.DBSnapshot{DBSnapshotIdentifier: r.DBSnapshotIdentifier, EngineVersion: r.EngineVersion}}, nil
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
