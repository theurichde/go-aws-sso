package internal

import (
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/manifoldco/promptui"
	"strings"
)

type Prompt interface {
	Select(label string, toSelect []string, searcher func(input string, index int) bool) (index int, value string)
	Prompt(label string, dfault string) string
}

type Prompter struct {
}

func (receiver Prompter) Select(label string, toSelect []string, searcher func(input string, index int) bool) (int, string) {
	prompt := promptui.Select{
		Label:             label,
		Items:             toSelect,
		Size:              20,
		Searcher:          searcher,
		StartInSearchMode: true,
	}
	index, value, err := prompt.Run()
	check(err)
	return index, value
}

func (receiver Prompter) Prompt(label string, dfault string) string {
	prompt := promptui.Prompt{
		Label:     label,
		Default:   dfault,
		AllowEdit: false,
	}
	val, err := prompt.Run()
	check(err)
	return val
}

func fuzzySearchWithPrefixAnchor(itemsToSelect []string, linePrefix string) func(input string, index int) bool {
	return func(input string, index int) bool {
		role := itemsToSelect[index]

		if strings.HasPrefix(input, linePrefix) {
			if strings.HasPrefix(role, input) {
				return true
			}
			return false
		} else {
			if fuzzy.MatchFold(input, role) {
				return true
			}
		}
		return false
	}
}
