package sso

import (
	"github.com/aws/aws-sdk-go/service/ssooidc"
	"github.com/aws/aws-sdk-go/service/ssooidc/ssooidciface"
	"reflect"
	"testing"
	"time"
)

type MockSSOOIDCClient struct {
	ssooidciface.SSOOIDCAPI
	CreateTokenOutput              ssooidc.CreateTokenOutput
	RegisterClientOutput           ssooidc.RegisterClientOutput
	StartDeviceAuthorizationOutput ssooidc.StartDeviceAuthorizationOutput
}

func (m MockSSOOIDCClient) CreateToken(_ *ssooidc.CreateTokenInput) (*ssooidc.CreateTokenOutput, error) {
	return &m.CreateTokenOutput, nil
}

func (m MockSSOOIDCClient) StartDeviceAuthorization(_ *ssooidc.StartDeviceAuthorizationInput) (*ssooidc.StartDeviceAuthorizationOutput, error) {
	return &m.StartDeviceAuthorizationOutput, nil
}

func (m MockSSOOIDCClient) RegisterClient(*ssooidc.RegisterClientInput) (*ssooidc.RegisterClientOutput, error) {
	return &m.RegisterClientOutput, nil
}

type mockTime struct {
	Timer
}

func (t mockTime) Now() time.Time {
	return time.Date(2021, 01, 01, 00, 00, 00, 00, &time.Location{})
}

func Test_retrieveToken(t *testing.T) {

	mockTime := mockTime{}
	at := "accessToken"
	want := ClientInformation{AccessToken: at, AccessTokenExpiresAt: mockTime.Now().Add(time.Hour*8 - time.Minute*5)}

	t.Run("foobar", func(t *testing.T) {
		mockClient := MockSSOOIDCClient{CreateTokenOutput: ssooidc.CreateTokenOutput{
			AccessToken: &at,
		}}

		got := retrieveToken(mockClient, mockTime, &ClientInformation{})

		if !reflect.DeepEqual(*got, want) {
			t.Errorf("retrieveToken() = got %v, want %v", *got, want)
		}

	})

}

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
