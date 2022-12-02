package internal

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/sso"
	"github.com/aws/aws-sdk-go/service/sso/ssoiface"
	. "github.com/theurichde/go-aws-sso/pkg/sso"
	"go.uber.org/zap"
	"sort"
	"strconv"
)

func RetrieveRoleInfo(accountInfo *sso.AccountInfo, clientInformation ClientInformation, ssoClient ssoiface.SSOAPI, selector Prompt) *sso.RoleInfo {
	lari := &sso.ListAccountRolesInput{AccountId: accountInfo.AccountId, AccessToken: &clientInformation.AccessToken}
	roles, _ := ssoClient.ListAccountRoles(lari)

	if len(roles.RoleList) == 1 {
		zap.S().Infof("Only one role available. Selected role: %s\n", *roles.RoleList[0].RoleName)
		return roles.RoleList[0]
	}

	var rolesToSelect []string
	linePrefix := "#"

	for i, info := range roles.RoleList {
		rolesToSelect = append(rolesToSelect, linePrefix+strconv.Itoa(i)+" "+*info.RoleName)
	}

	label := "Select your role - Hint: fuzzy search supported. To choose one role directly just enter #{Int}"
	indexChoice, _ := selector.Select(label, rolesToSelect, fuzzySearchWithPrefixAnchor(rolesToSelect, linePrefix))
	roleInfo := roles.RoleList[indexChoice]
	return roleInfo
}

func RetrieveAccountInfo(clientInformation ClientInformation, ssoClient ssoiface.SSOAPI, selector Prompt) *sso.AccountInfo {
	var maxSize int64 = 1000 // default is 20, but sometimes you have more accounts available ;-)
	lai := sso.ListAccountsInput{AccessToken: &clientInformation.AccessToken, MaxResults: &maxSize}
	accounts, _ := ssoClient.ListAccounts(&lai)

	sortedAccounts := sortAccounts(accounts.AccountList)

	var accountsToSelect []string
	linePrefix := "#"

	for i, info := range sortedAccounts {
		accountsToSelect = append(accountsToSelect, linePrefix+strconv.Itoa(i)+" "+*info.AccountName+" "+*info.AccountId)
	}

	label := "Select your account - Hint: fuzzy search supported. To choose one account directly just enter #{Int}"
	indexChoice, _ := selector.Select(label, accountsToSelect, fuzzySearchWithPrefixAnchor(accountsToSelect, linePrefix))

	fmt.Println()

	accountInfo := sortedAccounts[indexChoice]

	zap.S().Infof("Selected account: %s - %s", *accountInfo.AccountName, *accountInfo.AccountId)
	fmt.Println()
	return &accountInfo
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
