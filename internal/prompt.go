package internal

import (
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/manifoldco/promptui"
	"strings"
)

type Prompt interface {
	Select(label string, toSelect []string, searcher func(input string, index int) bool) promptui.Select
	Prompt(label string, def string) string
}

type Prompter struct {
}

func (receiver Prompter) Select(label string, toSelect []string, searcher func(input string, index int) bool) promptui.Select {
	return promptui.Select{
		Label:             label,
		Items:             toSelect,
		Size:              20,
		Searcher:          searcher,
		StartInSearchMode: true,
	}
}

func (receiver Prompter) Prompt(label string, def string) string {
	prompt := promptui.Prompt{
		Label:     label,
		Default:   def,
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
			} else {
				return false
			}
		} else {
			if fuzzy.MatchFold(input, role) {
				return true
			}
		}
		return false
	}
}
