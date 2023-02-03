package internal

import (
	"encoding/json"
	"errors"
	"github.com/aws/aws-sdk-go/service/sso"
	"github.com/valyala/fasttemplate"
	"go.uber.org/zap"
	"os"
	"path"
)

var CredentialsFilePath = GetCredentialsFilePath()

func ProcessPersistedCredentialsTemplate(credentials *sso.GetRoleCredentialsOutput, profile string) string {
	template := `[{{profile}}]
aws_access_key_id = {{access_key_id}}
aws_secret_access_key = {{secret_access_key}}
aws_session_token = {{session_token}}
output = json
region = eu-central-1
`

	engine := fasttemplate.New(template, "{{", "}}")
	filledTemplate := engine.ExecuteString(map[string]interface{}{
		"profile":           profile,
		"access_key_id":     *credentials.RoleCredentials.AccessKeyId,
		"secret_access_key": *credentials.RoleCredentials.SecretAccessKey,
		"session_token":     *credentials.RoleCredentials.SessionToken,
	})
	return filledTemplate
}

func ProcessCredentialProcessTemplate(accountId string, roleName string, profile string, region string) string {
	template := `[{{profile}}]
credential_process = go-aws-sso assume -a {{accountId}} -n {{roleName}}
region = {{region}}
`

	engine := fasttemplate.New(template, "{{", "}}")
	filledTemplate := engine.ExecuteString(map[string]interface{}{
		"profile":   profile,
		"region":    region,
		"accountId": accountId,
		"roleName":  roleName,
	})
	return filledTemplate
}

func GetCredentialsFilePath() string {
	homeDir, err := os.UserHomeDir()
	check(err)
	return homeDir + "/.aws/credentials"
}

func WriteAWSCredentialsFile(template string) {
	if !IsFileOrFolderExisting(CredentialsFilePath) {
		dir := path.Dir(CredentialsFilePath)
		err := os.MkdirAll(dir, 0755)
		check(err)
		f, err := os.OpenFile(CredentialsFilePath, os.O_CREATE, 0644)
		check(err)
		defer f.Close()
	}
	err := ioutil.WriteFile(CredentialsFilePath, []byte(template), 0644)
	check(err)
}

// IsFileOrFolderExisting
// Checks either or not a target file is existing.
// Returns true if the target exists, otherwise false.
func IsFileOrFolderExisting(target string) bool {
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
	if IsFileOrFolderExisting(file) {
		clientInformation := ClientInformation{}
		content, _ := ioutil.ReadFile(ClientInfoFileDestination())
		err := json.Unmarshal(content, &clientInformation)
		check(err)
		return clientInformation, nil
	}
	return ClientInformation{}, errors.New("no ClientInformation exist")
}

func WriteStructToFile(payload interface{}, dest string) {
	targetDir := path.Dir(dest)
	if !IsFileOrFolderExisting(targetDir) {
		err := os.MkdirAll(targetDir, 0700)
		check(err)
	}
	file, err := json.MarshalIndent(payload, "", " ")
	check(err)
	_ = ioutil.WriteFile(dest, file, 0600)
}
