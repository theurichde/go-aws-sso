# go-aws-sso

Make working with AWS SSO on local machines an ease.

## Description

* Choose and retrieve short-living role credentials from all of your SSO available accounts and roles  
* No nasty manual copy and pasting of credentials 

## Getting Started

Compile from source or download the according binary.

* Execute `go-aws-sso` and set your SSO start url `--start-url "https://my-sso-login.awsapps.com"`
* Choose the account you want to the roles to be displayed
* Choose a role
  * in case there is only one role available this role is taken as default
* Short living credentials are written to `~/.aws/credentials`

```
./go-aws-sso --start-url "https://my-sso-login.awsapps.com"

2021/09/10 22:08:27 Please verify your client request: https://device.sso.eu-central-1.amazonaws.com/?user_code=USR-CDE
2021/09/10 22:08:27 Still waiting for authorization...
[0] AccountName: "SDLC"
[1] AccountName: "Sandbox"
[2] AccountName: "Production"
Please choose an Account: 1
Only one role available. Selected role: AWSAccess
```

## License

This project is licensed under the MIT License - see the LICENSE.md file for details
