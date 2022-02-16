package internal

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestWriteClientInfoToFile(t *testing.T) {
	type args struct {
		information *ClientInformation
		dest        string
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "Should write client info to file", args: args{
			information: &ClientInformation{
				AccessTokenExpiresAt:    time.Time{},
				AccessToken:             "dummy",
				ClientId:                "dummy",
				ClientSecret:            "dummy",
				ClientSecretExpiresAt:   "dummy",
				DeviceCode:              "dummy",
				VerificationUriComplete: "dummy",
			},
			dest: createTempFile(),
		}},
		{name: "Should write client info to file even when the parent folder doesn't exist", args: args{
			information: &ClientInformation{
				AccessTokenExpiresAt:    time.Time{},
				AccessToken:             "dummy",
				ClientId:                "dummy",
				ClientSecret:            "dummy",
				ClientSecretExpiresAt:   "dummy",
				DeviceCode:              "dummy",
				VerificationUriComplete: "dummy",
			},
			dest: createTempFolder() + "/non-existent/token.json",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			defer os.RemoveAll(tt.args.dest)

			WriteClientInfoToFile(tt.args.information, tt.args.dest)

			_, err := os.Open(tt.args.dest)
			if err != nil {
				t.Errorf("Something went wrong while opening the file: %q,", err)
			}

			got := ClientInformation{}
			content, _ := ioutil.ReadFile(tt.args.dest)
			err = json.Unmarshal(content, &got)
			if err != nil {
				t.Error(err)
			}

			want := *tt.args.information
			if !reflect.DeepEqual(got, want) {
				t.Errorf("File content is not equal. Got: %q, want: %q", got, want)
			}
		})
	}
}

func createTempFolder() string {
	temp, err := os.MkdirTemp("", "write-client-info-test")
	check(err)
	return temp
}

func createTempFile() string {
	file, err := os.CreateTemp("", "write-client-info-test")
	check(err)
	return file.Name()
}
