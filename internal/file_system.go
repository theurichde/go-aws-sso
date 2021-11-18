package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/service/sso"
	"github.com/valyala/fasttemplate"
	"io/ioutil"
	"os"
)

func ProcessCredentialsTemplate(credentials *sso.GetRoleCredentialsOutput) string {
	template := `[default]
aws_access_key_id = {{access_key_id}}
aws_secret_access_key = {{secret_access_key}}
aws_session_token = {{session_token}}
output = json
region = eu-central-1`

	engine := fasttemplate.New(template, "{{", "}}")
	filledTemplate := engine.ExecuteString(map[string]interface{}{
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
	if !IsFileExisting(credentialsFile) {
		err = os.MkdirAll(homeDir+"/.aws", 0777)
		check(err)
		_, err = os.OpenFile(credentialsFile, os.O_CREATE, 0644)
		check(err)
	}
	err = ioutil.WriteFile(credentialsFile, []byte(template), 0644)
	check(err)
}

func IsFileExisting(file string) bool {
	if _, err := os.Stat(file); err == nil {
		return true
	} else if os.IsNotExist(err) {
		return false
	} else {
		message := fmt.Sprintf("Could not determine is file %q exists or not. Exiting.", file)
		panic(message)
	}
}

func WriteClientInfoToFile(information *ClientInformation, dest string) {
	file, err := json.MarshalIndent(information, "", " ")
	check(err)
	_ = ioutil.WriteFile(dest, file, 0644)
}

func ReadClientInformation(file string) (ClientInformation, error) {
	if IsFileExisting(file) {
		clientInformation := ClientInformation{}
		content, _ := ioutil.ReadFile(ClientInfoFileDestination())
		err := json.Unmarshal(content, &clientInformation)
		check(err)
		return clientInformation, nil
	}
	return ClientInformation{}, errors.New("no ClientInformation existing")
}
