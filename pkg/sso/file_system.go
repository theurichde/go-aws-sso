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

type ProfileTemplate struct {
	aws_access_key_id     string
	aws_secret_access_key string
	aws_session_token     string
	output                string
	profile               string
	region                string
	accountId             string
	roleName              string
}

func ProcessPersistedCredentialsTemplate(credentials *sso.GetRoleCredentialsOutput, profile string, region string) ProfileTemplate {
	profileTemplate := ProfileTemplate{
		profile:               profile,
		region:                region,
		aws_access_key_id:     *credentials.RoleCredentials.AccessKeyId,
		aws_secret_access_key: *credentials.RoleCredentials.SecretAccessKey,
		aws_session_token:     *credentials.RoleCredentials.SessionToken,
		output:                "json",
	}

	return profileTemplate
}

func ProcessCredentialProcessTemplate(accountId string, roleName string, profile string, region string) ProfileTemplate {
	profileTemplate := ProfileTemplate{
		profile:   profile,
		region:    region,
		accountId: accountId,
		roleName:  roleName,
	}
	return profileTemplate
}

func GetCredentialsFilePath() string {
	homeDir, err := os.UserHomeDir()
	check(err)
	return homeDir + "/.aws/credentials"
}

func WriteAWSCredentialsFile(template ProfileTemplate, profile string, persist bool) {
	if !isFileOrFolderExisting(CredentialsFilePath) {
		dir := path.Dir(CredentialsFilePath)
		err := os.MkdirAll(dir, 0755)
		check(err)
		f, err := os.OpenFile(CredentialsFilePath, os.O_CREATE, 0644)
		check(err)
		defer f.Close()
	}

	cfg, err := ini.Load(CredentialsFilePath)
	check(err)

	sec, err := cfg.GetSection(profile)
	if err == nil {
		if persist {
			cfg.DeleteSection(sec.Name())
			sec, err = cfg.NewSection(profile)
			check(err)
			addPersistedProfileKeys(template, sec)
		} else {
			cfg.DeleteSection(sec.Name())
			sec, err = cfg.NewSection(profile)
			check(err)
			addProfileKeys(template, sec)
		}
	} else {
		sec, err := cfg.NewSection(profile)
		check(err)

		if persist {
			addPersistedProfileKeys(template, sec)
		} else {
			addProfileKeys(template, sec)
		}
	}
	cfg.SaveTo(CredentialsFilePath)
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

func addProfileKeys(template ProfileTemplate, sec *ini.Section) {
	_, err := sec.NewKey("credential_process", fmt.Sprintf("go-aws-sso assume -a %s -n %s", template.accountId, template.roleName))
	check(err)
	_, err = sec.NewKey("region", template.region)
	check(err)
}

func addPersistedProfileKeys(template ProfileTemplate, sec *ini.Section) {
	_, err := sec.NewKey("aws_access_key_id", template.aws_access_key_id)
	check(err)
	_, err = sec.NewKey("aws_secret_access_key", template.aws_secret_access_key)
	check(err)
	_, err = sec.NewKey("aws_session_token", template.aws_session_token)
	check(err)
	_, err = sec.NewKey("output", "json")
	check(err)
	_, err = sec.NewKey("region", template.region)
	check(err)
}

func check(err error) {
	if err != nil {
		zap.S().Fatalf("Something went wrong: %q", err)
	}
}
