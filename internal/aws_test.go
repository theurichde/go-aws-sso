package internal

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/sso"
	"github.com/aws/aws-sdk-go/service/sso/ssoiface"
	. "github.com/theurichde/go-aws-sso/pkg/sso"
)

type accountListingSSOClient struct {
	ssoiface.SSOAPI
	t        *testing.T
	requests []sso.ListAccountsInput
}

func (m *accountListingSSOClient) ListAccounts(input *sso.ListAccountsInput) (*sso.ListAccountsOutput, error) {
	m.t.Helper()
	m.requests = append(m.requests, *input)
	if input.MaxResults == nil {
		m.t.Fatal("ListAccounts MaxResults must be set")
	}
	if *input.MaxResults > 100 {
		m.t.Fatalf("ListAccounts MaxResults = %d, want no more than 100", *input.MaxResults)
	}

	accountID := "123456789012"
	accountName := "Production"
	return &sso.ListAccountsOutput{
		AccountList: []*sso.AccountInfo{
			{
				AccountId:   &accountID,
				AccountName: &accountName,
			},
		},
	}, nil
}

type accountSelectionPrompt struct{}

func (p accountSelectionPrompt) Select(_ string, toSelect []string, _ func(input string, index int) bool) (int, string) {
	return 0, toSelect[0]
}

func (p accountSelectionPrompt) Prompt(_ string, _ string) string {
	return ""
}

func TestRetrieveAccountInfoUsesAWSAllowedMaxResults(t *testing.T) {
	ssoClient := &accountListingSSOClient{t: t}

	accountInfo, awsErr := RetrieveAccountInfo(
		ClientInformation{AccessToken: "access-token"},
		ssoClient,
		accountSelectionPrompt{},
	)

	if awsErr != nil {
		t.Fatalf("RetrieveAccountInfo returned AWS error: %v", awsErr)
	}
	if accountInfo == nil {
		t.Fatal("RetrieveAccountInfo returned nil account info")
	}
	if got := *accountInfo.AccountId; got != "123456789012" {
		t.Fatalf("account ID = %s, want 123456789012", got)
	}
	if got := *ssoClient.requests[0].MaxResults; got != 100 {
		t.Fatalf("ListAccounts MaxResults = %d, want 100", got)
	}
}
