package sso

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/aws/aws-sdk-go/service/sso"
	"go.uber.org/zap"
	"gopkg.in/ini.v1"
)

var CredentialsFilePath = GetCredentialsFilePath()

type CredentialsFileTemplate struct {
	AwsAccessKeyId     string `ini:"aws_access_key_id,omitempty"`
	AwsSecretAccessKey string `ini:"aws_secret_access_key,omitempty"`
	AwsSessionToken    string `ini:"aws_session_token,omitempty"`
	CredentialProcess  string `ini:"credential_process,omitempty"`
	Output             string `ini:"output,omitempty"`
	Region             string `ini:"region,omitempty"`
}

func ProcessPersistedCredentialsTemplate(credentials *sso.GetRoleCredentialsOutput, region string) CredentialsFileTemplate {
	profileTemplate := CredentialsFileTemplate{
		AwsAccessKeyId:     *credentials.RoleCredentials.AccessKeyId,
		AwsSecretAccessKey: *credentials.RoleCredentials.SecretAccessKey,
		AwsSessionToken:    *credentials.RoleCredentials.SessionToken,
		Region:             region,
	}

	return profileTemplate
}

func ProcessCredentialProcessTemplate(accountId string, roleName string, region string) CredentialsFileTemplate {
	exeName, err := os.Executable()
	check(err)
	profileTemplate := CredentialsFileTemplate{
		CredentialProcess: fmt.Sprintf("%s assume -q -a %s -n %s", exeName, accountId, roleName),
		Region:            region,
	}
	return profileTemplate
}

func GetCredentialsFilePath() string {
	homeDir, err := os.UserHomeDir()
	check(err)
	return homeDir + "/.aws/credentials"
}

func WriteAWSCredentialsFile(template *CredentialsFileTemplate, profile string) {
	if !isFileOrFolderExisting(CredentialsFilePath) {
		createCredentialsFile()
	}
	writeIniFile(template, profile)
}

func createCredentialsFile() {
	dir := path.Dir(CredentialsFilePath)
	err := os.MkdirAll(dir, 0755)
	check(err)
	f, err := os.OpenFile(CredentialsFilePath, os.O_CREATE, 0644)
	check(err)
	defer f.Close()
}

func writeIniFile(template *CredentialsFileTemplate, profile string) {
	cfg, err := ini.Load(CredentialsFilePath)
	check(err)

	recreateSection(template, profile, cfg)

	zap.S().Debugf("Saving ini file to %s", CredentialsFilePath)
	cfg.SaveTo(CredentialsFilePath)
}

func recreateSection(template *CredentialsFileTemplate, profile string, cfg *ini.File) {
	zap.S().Debugf("Deleting profile [%s] in credentials file", profile)
	cfg.DeleteSection(profile)
	sec, err := cfg.NewSection(profile)
	check(err)
	zap.S().Debugf("Reflecting profile [%s] in credentials file", profile)
	err = sec.ReflectFrom(template)
}

// isFileOrFolderExisting
// Checks either or not a target file is existing.
// Returns true if the target exists, otherwise false.
func isFileOrFolderExisting(target string) bool {
	if _, err := os.Stat(target); err == nil {
		return true
	} else if os.IsNotExist(err) {
		return false
	} else {
		zap.S().Panicf("Could not determine if file or folder %s exists or not. Exiting.", target)
		return false
	}
}

func ReadClientInformation(file string) (ClientInformation, error) {
	if isFileOrFolderExisting(file) {
		clientInformation := ClientInformation{}
		content, _ := os.ReadFile(ClientInfoFileDestination())
		err := json.Unmarshal(content, &clientInformation)
		check(err)
		return clientInformation, nil
	}
	return ClientInformation{}, errors.New("no ClientInformation exist")
}

func WriteStructToFile(payload interface{}, dest string) {
	targetDir := path.Dir(dest)
	if !isFileOrFolderExisting(targetDir) {
		err := os.MkdirAll(targetDir, 0700)
		check(err)
	}
	file, err := json.MarshalIndent(payload, "", " ")
	check(err)
	_ = os.WriteFile(dest, file, 0600)
}

func check(err error) {
	if err != nil {
		zap.S().Fatalf("Something went wrong: %q", err)
	}
}
