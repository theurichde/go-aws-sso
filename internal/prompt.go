package internal

import (
	"github.com/chzyer/readline"
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

type noBellStdout struct{}

func (n *noBellStdout) Write(p []byte) (int, error) {
	if len(p) == 1 && p[0] == readline.CharBell {
		return 0, nil
	}
	return readline.Stdout.Write(p)
}

func (n *noBellStdout) Close() error {
	return readline.Stdout.Close()
}

var NoBellStdout = &noBellStdout{}

func (p Prompter) Select(label string, toSelect []string, searcher func(input string, index int) bool) (int, string) {
	prompt := promptui.Select{
		Label:             label,
		Items:             toSelect,
		Size:              20,
		Searcher:          searcher,
		StartInSearchMode: true,
		Stdout:            NoBellStdout,
	}

	index, value, err := prompt.Run()
	check(err)
	return index, value
}

func (p Prompter) Prompt(label string, dfault string) string {
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
