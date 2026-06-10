package internal

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sso"
	"github.com/aws/aws-sdk-go/service/sso/ssoiface"
	. "github.com/theurichde/go-aws-sso/pkg/sso"
	"github.com/theurichde/go-aws-sso/pkg/logger"
)

func RetrieveRoleInfo(accountInfo *sso.AccountInfo, clientInformation ClientInformation, ssoClient ssoiface.SSOAPI, selector Prompt) (*sso.RoleInfo, awserr.RequestFailure) {
	var maxSize int64 = 100 // AWS SSO API limit is 100
	lari := &sso.ListAccountRolesInput{AccountId: accountInfo.AccountId, AccessToken: &clientInformation.AccessToken, MaxResults: &maxSize}

	var allRoles []*sso.RoleInfo
	for {
		roles, err := ssoClient.ListAccountRoles(lari)
		if err != nil {
			if awsError, ok := err.(awserr.RequestFailure); ok {
				return nil, awsError
			}
			logger.CheckFatal(err)
		}

		allRoles = append(allRoles, roles.RoleList...)

		if roles.NextToken == nil {
			break
		}
		lari.NextToken = roles.NextToken
	}

	if len(allRoles) == 1 {
		logger.L.Infof("Only one role available. Selected role: %s\n", *allRoles[0].RoleName)
		return allRoles[0], nil
	}

	var rolesToSelect []string
	linePrefix := "#"

	for i, info := range allRoles {
		rolesToSelect = append(rolesToSelect, linePrefix+strconv.Itoa(i)+" "+*info.RoleName)
	}

	label := "Select your role - Hint: fuzzy search supported. To choose one role directly just enter #{Int}"
	indexChoice, _ := selector.Select(label, rolesToSelect, fuzzySearchWithPrefixAnchor(rolesToSelect, linePrefix))
	roleInfo := allRoles[indexChoice]
	return roleInfo, nil
}

func RetrieveAccountInfo(clientInformation ClientInformation, ssoClient ssoiface.SSOAPI, selector Prompt) (*sso.AccountInfo, awserr.RequestFailure) {
	var maxSize int64 = 100 // default is 20, but sometimes you have more accounts available ;-)
	lai := sso.ListAccountsInput{AccessToken: &clientInformation.AccessToken, MaxResults: &maxSize}

	var allAccounts []*sso.AccountInfo
	for {
		accounts, err := ssoClient.ListAccounts(&lai)
		if err != nil {
			if awsError, ok := err.(awserr.RequestFailure); ok {
				return nil, awsError
			}
			logger.CheckFatal(err)
		}

		allAccounts = append(allAccounts, accounts.AccountList...)

		if accounts.NextToken == nil {
			break
		}
		lai.NextToken = accounts.NextToken
	}

	sortedAccounts := sortAccounts(allAccounts)

	var accountsToSelect []string
	linePrefix := "#"

	for i, info := range sortedAccounts {
		accountsToSelect = append(accountsToSelect, linePrefix+strconv.Itoa(i)+" "+*info.AccountName+" "+*info.AccountId)
	}

	fmt.Println("Hint: fuzzy search supported. To choose one account directly just enter #{Int}.")

	indexChoice, _ := selector.Select("Select your account", accountsToSelect, fuzzySearchWithPrefixAnchor(accountsToSelect, linePrefix))

	fmt.Println()

	accountInfo := sortedAccounts[indexChoice]

	logger.L.Infof("Selected account: %s - %s", *accountInfo.AccountName, *accountInfo.AccountId)
	fmt.Println()
	return &accountInfo, nil
}

func sortAccounts(accountList []*sso.AccountInfo) []sso.AccountInfo {
	var sortedAccounts []sso.AccountInfo
	for _, info := range accountList {
		sortedAccounts = append(sortedAccounts, *info)
	}
	sort.Slice(sortedAccounts, func(i, j int) bool {
		return *sortedAccounts[i].AccountName < *sortedAccounts[j].AccountName
	})
	return sortedAccounts
}
