[![Go Report Card](https://goreportcard.com/badge/github.com/theurichde/go-aws-sso)](https://goreportcard.com/report/github.com/theurichde/go-aws-sso)

# go-aws-sso

Make working with AWS SSO on local machines an ease.

## What is it about?

* Choose and retrieve short-living role credentials from all of your SSO available accounts and roles
* No nasty manual copy and pasting of credentials

### But... why? 🤔

* You have one go-to binary
* No external dependencies (e.g. a python runtime)
* Forget about dealing with different profiles and role names, just choose them directly!

## Features

* Choose your desired account and role interactively
* Choose your account and role via flags from command line
* Utilize AWSs `credential_process` to avoid storing credentials locally
  * Locks concurrent calls to `credential_process` to no DDoS your Browser (this behaviour occurs from time to time when using e.g. the k8s plugin for IntelliJ)   
* Refresh credentials based on your previously chosen account and role (if you've chosen to persist your credentials)
* Store your Start-URL and region
* Set different profiles for your different accounts

## Getting Started

### Installation

- via homebrew
    - `brew tap theurichde/go-aws-sso && brew install go-aws-sso`
- Download your according target platform binary from
  the [releases page](https://github.com/theurichde/go-aws-sso/releases)
- Compile from source with `go build -v ./cmd/go-aws-sso`
- use `go install github.com/theurichde/go-aws-sso/cmd/go-aws-sso@main`
    * Maybe you want to make sure your GOBIN is in your PATH 😉

### Usage

#### Interactively Assume a Role

* Just execute `go-aws-sso`
    * When you run `go-aws-sso` the first time, you will be prompted for your SSO Start URL and your region
    * A config file (located at  `$HOME/$CONFIG_DIR/go-aws-sso/config.yml`) will be written with your values
* ❔ Verify your client request if necessary
* ✅ Choose the account you want the roles to be displayed
* ✅ Choose a role
    * in case there is only one role available this role is taken as default
* 🥳 Tadaa 🥳
    * `credentials_process` is written to `~/.aws/credentials` and fetches fresh credentials every time something calls
      AWS (implementing a proper credentials caching mechanism is up to the calling program)
    * if you've added the flag `--persist`: short living credentials are written to `~/.aws/credentials`

#### Directly Assume a Role From Command Line

```
$ ./go-aws-sso help assume
NAME:
   go-aws-sso assume - Assume directly into an account and SSO role

USAGE:
   go-aws-sso assume [command options] [arguments...]

DESCRIPTION:
   Assume directly into an account and SSO role

OPTIONS:
   --start-url value, -u value     set / override the SSO login start-url. (Example: https://my-login.awsapps.com/start#/)
   --region value, -r value        set / override the AWS region
   --profile value, -p value       the profile name you want to set in your ~/.aws/credentials file (default: "default")
   --persist                       whether or not you want to write your short-living credentials to ~/.aws/credentials (default: false)
   --force                         removes the temporary access token and forces the retrieval of a new token (default: false)
   --debug                         enables debug logging (default: false)
   --role-name value, -n value     The role name you want to assume
   --account-id value, -a value    The account id where your role lives in
   --quiet, -q, --non-interactive  disables logger output (default: false)
   --help, -h                      show help
```

* Execute `go-aws-sso assume --account-id YOUR_ID --role-name YOUR_ROLE_NAME`
* Optionally: Set / override your start url and region via flag


### Refresh Credentials

Refreshing credentials is only useful, if you persist your credentials (SecretAccessKey etc.) in your `~/.aws/credentials` file

```
$ go-aws-sso help refresh
NAME:
   go-aws-sso refresh - Refresh your previously used credentials.

USAGE:
   go-aws-sso refresh [command options] [arguments...]

DESCRIPTION:
   Refreshes the short living credentials based on your last account and role.

OPTIONS:
   --start-url value, -u value  set / override the SSO login start-url. (Example: https://my-login.awsapps.com/start#/)
   --region value, -r value     set / override the AWS region
   --profile value, -p value    the profile name you want to set in your ~/.aws/credentials file (default: "default")
   --persist                    whether or not you want to write your short-living credentials to ~/.aws/credentials (default: false)
   --force                      removes the temporary access token and forces the retrieval of a new token (default: false)
   --debug                      enables debug logging (default: false)
   --help, -h                   show help
```

### Configuration

* If you want to point to a specific non-default Browser, do so via the `BROWSER` environment variable

<details><summary>Basics</summary>

```
$ go-aws-sso config                                 
NAME:
   go-aws-sso config - Handles configuration. Note: Config location defaults to $HOME/$CONFIG_DIR/go-aws-sso/config.yml

USAGE:
   go-aws-sso config command [command options] [arguments...]

COMMANDS:
   generate  Generate a config file
   edit      Edit the config file
   help, h   Shows a list of commands or help for one command

OPTIONS:
   --help, -h  show help (default: false)
```

</details>

* <details><summary>Config Generation</summary>

  ```
  $ go-aws-sso config generate --help
  NAME:
     go-aws-sso config generate - Generate a config file
  
  USAGE:
     go-aws-sso config generate [command options] [arguments...]
  
  DESCRIPTION:
     Generates a config file. All available properties are interactively prompted if not set with command options.
     Overrides the existing config file!
  
  OPTIONS:
     --start-url value, -u value   set / override the SSO login start-url. (Example: https://my-login.awsapps.com/start#/)
     --region value, -r value      set / override the AWS region
     --help, -h                    show help
  ```

</details>

* <details><summary>Config Editing</summary>

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

</details>

### Example Usage

```
$ go-aws-sso help  
NAME:
   go-aws-sso - Retrieve short-living credentials via AWS SSO & SSOOIDC

USAGE:
   go-aws-sso [global options] command [command options] [arguments...]

VERSION:
   v1.2.0

COMMANDS:
   config   Handles configuration. Note: Config location defaults to $HOME/$CONFIG_DIR/go-aws-sso/config.yml
   refresh  Refresh your previously used credentials.
   assume   Assume directly into an account and SSO role
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --start-url value, -u value  set / override the SSO login start-url. (Example: https://my-login.awsapps.com/start#/)
   --region value, -r value     set / override the AWS region
   --profile value, -p value    the profile name you want to set in your ~/.aws/credentials file (default: "default")
   --persist                    whether or not you want to write your short-living credentials to ~/.aws/credentials (default: false)
   --force                      removes the temporary access token and forces the retrieval of a new token (default: false)
   --debug                      enables debug logging (default: false)
   --help, -h                   show help
   --version, -v                print the version
```

---

```
./go-aws-sso

2021/11/08 19:34:40 WARN No Start URL given. Please set it now.
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
2021/11/08 19:34:40 INFO Config file generated: /home/theurichde/.config/go-aws-sso/config.yml
2021/11/08 19:34:40 WARN Please verify your client request: https://device.sso.eu-central-1.amazonaws.com/?user_code=USR-CDE
2021/11/08 19:34:40 INFO Still waiting for authorization...
Search: 
? Select your account - Hint: fuzzy search supported. To choose one account directly just enter #{Int}: 
  ▸ #0 Awesome API - SDLC YYYYYXXXXXXX
    #1 Team Sandbox XXXXXXXXXXXX
    #2 Awesome API - Production YYYYYYYYYYYY

2021/11/08 19:34:43 INFO Selected account: Team Sandbox - XXXXXXXXXXXX

2021/11/08 19:34:43 INFO Only one role available. Selected role: AWSAdministratorAccess
2021/11/08 19:34:43 INFO Credentials expire at: 2021-11-08 20:34:43 +0100 CET
```

---

## Contributions

*Contributions are highly welcome!*

* Feel free to contribute enhancements or bug fixes.
    * Fork this repo, apply your changes and create a PR pointing to this repo and the main branch
* If you have any ideas or suggestions please open an issue and describe your idea or feature request

## License

This project is licensed under the MIT License - see the LICENSE.md file for details
