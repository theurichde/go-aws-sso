package sso

import (
	"encoding/json"
	"gopkg.in/ini.v1"
	"os"
	"reflect"
	"testing"
	"time"
)

func Test_isFileExisting(t *testing.T) {

	tempFile, _ := os.CreateTemp("", "used-for-testing")
	t.Run("True if file exists", func(t *testing.T) {
		got := isFileOrFolderExisting(tempFile.Name())
		if got != true {
			t.Errorf("isFileExisting() = %v, want %v", got, true)
		}
	})

	t.Run("False if file does not exist", func(t *testing.T) {
		got := isFileOrFolderExisting("/tmp/not-existing-file.name")
		if got != false {
			t.Errorf("isFileExisting() = %v, want %v", got, true)
		}
	})
}

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

			WriteStructToFile(tt.args.information, tt.args.dest)

			_, err := os.Open(tt.args.dest)
			if err != nil {
				t.Errorf("Something went wrong while opening the file: %q,", err)
			}

			got := ClientInformation{}
			content, _ := os.ReadFile(tt.args.dest)
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

func TestWriteAWSCredentialsFile(t *testing.T) {

	tmp, _ := os.CreateTemp("", "testCredentials")
	defer os.Remove(tmp.Name())
	CredentialsFilePath = tmp.Name()
	init, _ := ini.Load(CredentialsFilePath)
	initTemplate := CredentialsFileTemplate{
		AwsAccessKeyId:     "dummyAwsAccessKeyId",
		AwsSecretAccessKey: "dummyAwsSecretAccessKey",
		AwsSessionToken:    "dummyAwsSessionToken",
		Region:             "dummyRegion",
	}

	profile := "testProfile"
	section, _ := init.NewSection(profile)
	section.ReflectFrom(&initTemplate)

	want := CredentialsFileTemplate{
		CredentialProcess: "dummy credential process",
		Region:            "eu-central-1",
	}

	t.Run("Already existing section with attributes should be completely replaced by new or adapted section", func(t *testing.T) {

		WriteAWSCredentialsFile(&want, profile)

		gotIni, _ := ini.Load(CredentialsFilePath)
		gotSection, _ := gotIni.GetSection(profile)
		got := CredentialsFileTemplate{}
		gotSection.MapTo(&got)

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got: %q, want: %q", got, want)
		}
	})
}
