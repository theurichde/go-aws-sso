# go-aws-sso

Make working with AWS SSO on local machines an ease.

## What is it about?

* Choose and retrieve short-living role credentials from all of your SSO available accounts and roles
* No nasty manual copy and pasting of credentials

### But... why? 🤔
* You have one go-to binary
* No external dependencies (e.g. a python runtime)
* Forget about dealing with different profiles and role names, just choose them directly!

## Getting Started

### Installation
* a) Download your according target platform binary from the [releases page](https://github.com/theurichde/go-aws-sso/releases)
* b) Compile from source with `go build -v ./cmd/go-aws-sso`
* c) use `go install github.com/theurichde/go-aws-sso/cmd/go-aws-sso@main`
  * Maybe you want to make sure your GOBIN is in your PATH 😉

### Usage
* Just execute `go-aws-sso`
  * When you run `go-aws-sso` the first time, you will be prompted for your SSO Start URL and your region
  * A config file (located at  `$HOME/.aws/go-aws-sso-config.yaml`) will be written with your values
* ❔ Verify your client request if necessary 
* ✅ Choose the account you want the roles to be displayed
* ✅ Choose a role
    * in case there is only one role available this role is taken as default
* 🥳 Tadaa 🥳 short living credentials are written to `~/.aws/credentials`

### Configuration
```
$ go-aws-sso config                                 
NAME:
   go-aws-sso config - Handles configuration. Note: Config location defaults to ${HOME}/.aws/go-aws-sso-config.yaml

USAGE:
   go-aws-sso config command [command options] [arguments...]

COMMANDS:
   generate  Generate a config file
   edit      Edit the config file
   help, h   Shows a list of commands or help for one command

OPTIONS:
   --help, -h  show help (default: false)
```

#### Config Generation
  ```
  $ go-aws-sso config generate --help
  NAME:
     go-aws-sso config generate - Generate a config file
  
  USAGE:
     go-aws-sso config generate [command options] [arguments...]
  
  DESCRIPTION:
     Generates a config file. All available properties are interactively prompted.
     Overrides the existing config file!
  
  OPTIONS:
     --help, -h  show help (default: false)
  ```

#### Config Editing
  ```
  $ go-aws-sso config edit --help    
  NAME:
     go-aws-sso config edit - Edit the config file
  
  USAGE:
     go-aws-sso config edit [command options] [arguments...]
  
  DESCRIPTION:
     Edit the config file. All available properties are interactively prompted.
     Overrides the existing config file!
  
  OPTIONS:
     --help, -h  show help (default: false)
  ```

### Example Usage
```
$ go-aws-sso help  
NAME:
   go-aws-sso - Retrieve short-living credentials via AWS SSO & SSOOIDC

USAGE:
   go-aws-sso [global options] command [command options] [arguments...]

COMMANDS:
   config   Handles configuration
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --start-url value, -u value  Set / override the SSO login start-url. (Example: https://my-login.awsapps.com/start#/)
   --region value, -r value     Set / override the AWS region (default: "eu-central-1")
   --help, -h                   show help (default: false)
```

---
```
./go-aws-sso

2021/11/08 19:34:40 No Start URL given. Please set it now.
✔ SSO Start URL: https://my-sso-login.awsapps.com
Search: █
? Select your AWS Region. Hint: FuzzySearch supported: 
  ▸ us-east-2
    us-east-1
    us-west-1
    us-west-2
    af-south-1
    ap-east-1
    ap-south-1
    ap-northeast-3
    ap-northeast-2
    [...]
2021/11/08 19:34:40 Config file generated: /home/theurichde/.aws/go-aws-sso-config.yaml
2021/11/08 19:34:40 Please verify your client request: https://device.sso.eu-central-1.amazonaws.com/?user_code=USR-CDE
2021/11/08 19:34:40 Still waiting for authorization...
Search: 
? Select your account - Hint: fuzzy search supported. To choose one account directly just enter #{Int}: 
  ▸ #0 Awesome API - SDLC YYYYYXXXXXXX
    #1 Team Sandbox XXXXXXXXXXXX
    #2 Awesome API - Production YYYYYYYYYYYY

2021/11/08 19:34:43 Selected account: Team Sandbox - XXXXXXXXXXXX

2021/11/08 19:34:43 Only one role available. Selected role: AWSAdministratorAccess
2021/11/08 19:34:43 Credentials expire at: 2021-11-08 20:34:43 +0100 CET
```
---

## Contributions

*Contributions are highly welcome!*

* Feel free to contribute enhancements or bug fixes. 
  * Fork this repo, apply your changes and create a PR pointing to this repo and the develop branch
* If you have any ideas or suggestions please open an issue and describe your idea or feature request

## License

This project is licensed under the MIT License - see the LICENSE.md file for details
