module github.com/theurichde/go-aws-sso

require (
	github.com/aws/aws-sdk-go v1.42.9
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/theurichde/go-aws-sso/internal v1.0.0
	github.com/urfave/cli/v2 v2.3.0
)

replace github.com/theurichde/go-aws-sso/internal v1.0.0 => ./internal

go 1.16
