package main

import (
	"github.com/aws/aws-sdk-go/service/sso"
	"github.com/aws/aws-sdk-go/service/ssooidc"
	"reflect"
	"testing"
	"time"
)

func TestClientInformation_isExpired(t *testing.T) {
	type fields struct {
		AccessTokenExpiresAt time.Time
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "Should recognize a non-expired token",
			fields: fields{
				AccessTokenExpiresAt: time.Now().Add(time.Hour*8 - time.Minute*5),
			},
			want: false,
		},
		{
			name: "Should recognize an expired token",
			fields: fields{
				AccessTokenExpiresAt: time.Now().Add(-8 * time.Hour),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ati := ClientInformation{
				AccessTokenExpiresAt: tt.fields.AccessTokenExpiresAt,
			}
			if got := ati.isExpired(); got != tt.want {
				t.Errorf("isExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_handleOutdatedAccessToken(t *testing.T) {
	type args struct {
		clientInformation ClientInformation
		oidcClient        *ssooidc.SSOOIDC
	}
	tests := []struct {
		name string
		args args
		want ClientInformation
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handleOutdatedAccessToken(tt.args.clientInformation, tt.args.oidcClient); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleOutdatedAccessToken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_readClientInformation(t *testing.T) {
	type args struct {
		file string
	}
	tests := []struct {
		name    string
		args    args
		want    ClientInformation
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readClientInformation(tt.args.file)
			if (err != nil) != tt.wantErr {
				t.Errorf("readClientInformation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readClientInformation() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_registerClient(t *testing.T) {
	type args struct {
		oidc *ssooidc.SSOOIDC
	}
	tests := []struct {
		name string
		args args
		want *ClientInformation
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := registerClient(tt.args.oidc); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("registerClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_retrieveAccountInfo(t *testing.T) {
	type args struct {
		clientInformation ClientInformation
		ssoClient         *sso.SSO
	}
	tests := []struct {
		name    string
		args    args
		want    *sso.AccountInfo
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := retrieveAccountInfo(tt.args.clientInformation, tt.args.ssoClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("retrieveAccountInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("retrieveAccountInfo() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_retrieveRoleInfo(t *testing.T) {
	type args struct {
		accountInfo       *sso.AccountInfo
		clientInformation ClientInformation
		ssoClient         *sso.SSO
	}
	tests := []struct {
		name    string
		args    args
		want    *sso.RoleInfo
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := retrieveRoleInfo(tt.args.accountInfo, tt.args.clientInformation, tt.args.ssoClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("retrieveRoleInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("retrieveRoleInfo() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_retrieveToken(t *testing.T) {
	type args struct {
		client      *ssooidc.SSOOIDC
		input       ssooidc.CreateTokenInput
		information *ClientInformation
	}
	tests := []struct {
		name string
		args args
		want *ClientInformation
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := retrieveToken(tt.args.client, tt.args.input, tt.args.information); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("retrieveToken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_startDeviceAuthorization(t *testing.T) {
	type args struct {
		oidc *ssooidc.SSOOIDC
		rco  *ssooidc.RegisterClientOutput
	}
	tests := []struct {
		name string
		args args
		want *ssooidc.StartDeviceAuthorizationOutput
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := startDeviceAuthorization(tt.args.oidc, tt.args.rco); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("startDeviceAuthorization() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_tryToRetrieveToken(t *testing.T) {
	type args struct {
		client *ssooidc.SSOOIDC
		input  ssooidc.CreateTokenInput
		info   *ClientInformation
	}
	tests := []struct {
		name string
		args args
		want *ClientInformation
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tryToRetrieveToken(tt.args.client, tt.args.input, tt.args.info); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("tryToRetrieveToken() = %v, want %v", got, tt.want)
			}
		})
	}
}
