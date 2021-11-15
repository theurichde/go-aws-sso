# go-aws-sso

Make working with AWS SSO on local machines an ease.

## Description

* Choose and retrieve short-living role credentials from all of your SSO available accounts and roles
* No nasty manual copy and pasting of credentials

## Getting Started

Compile from source or download the according binary.

* A) Execute `go-aws-sso` and set your SSO start url `--start-url "https://my-sso-login.awsapps.com"`
* B) Create a .yaml file, put your start url in there and refer this file via `go-aws-sso -c my-config-file.yaml`
  * ```start-url: https://my-sso-login.awsapps.com``` 
* C) Create a file `~/.aws/go-aws-sso-config.yaml` and put the start-url in there
* Choose the account you want the roles to be displayed
* Choose a role
    * in case there is only one role available this role is taken as default
* Short living credentials are written to `~/.aws/credentials`

```
./go-aws-sso --start-url "https://my-sso-login.awsapps.com"

2021/11/08 19:34:40 Please verify your client request: https://device.sso.eu-central-1.amazonaws.com/?user_code=USR-CDE
2021/11/08 19:34:40 Still waiting for authorization...
Search: 
? Select your account - Hint: fuzzy search supported. To choose one account directly just enter #{Int}: 
  ▸ #0 Awesome API - SDLC YYYYYXXXXXXX
    #1 Team Sandbox XXXXXXXXXXXX
    #2 Awesome API - Production YYYYYYYYYYYY

2021/11/08 19:34:43 Selected account: Team Sandbox - 217238673132

2021/11/08 19:34:43 Only one role available. Selected role: AWSAdministratorAccess
2021/11/08 19:34:43 Credentials expire at: 2021-11-08 20:34:43 +0100 CET
```

## License

This project is licensed under the MIT License - see the LICENSE.md file for details
