package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/service/sso"
	"github.com/valyala/fasttemplate"
	"io/ioutil"
	"os"
	"path"
)

func ProcessCredentialsTemplate(credentials *sso.GetRoleCredentialsOutput, profile string) string {
	template := `[{{profile}}]
aws_access_key_id = {{access_key_id}}
aws_secret_access_key = {{secret_access_key}}
aws_session_token = {{session_token}}
output = json
region = eu-central-1`

	engine := fasttemplate.New(template, "{{", "}}")
	filledTemplate := engine.ExecuteString(map[string]interface{}{
		"profile":           profile,
		"access_key_id":     *credentials.RoleCredentials.AccessKeyId,
		"secret_access_key": *credentials.RoleCredentials.SecretAccessKey,
		"session_token":     *credentials.RoleCredentials.SessionToken,
	})
	return filledTemplate
}

func WriteAWSCredentialsFile(template string) {

	homeDir, err := os.UserHomeDir()
	check(err)

	credentialsFile := homeDir + "/.aws/credentials"
	if !IsFileOrFolderExisting(credentialsFile) {
		err = os.MkdirAll(homeDir+"/.aws", 0777)
		check(err)
		_, err = os.OpenFile(credentialsFile, os.O_CREATE, 0644)
		check(err)
	}
	err = ioutil.WriteFile(credentialsFile, []byte(template), 0644)
	check(err)
}

func IsFileOrFolderExisting(target string) bool {
	if _, err := os.Stat(target); err == nil {
		return true
	} else if os.IsNotExist(err) {
		return false
	} else {
		message := fmt.Sprintf("Could not determine if file or folder %q exists or not. Exiting.", target)
		panic(message)
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
