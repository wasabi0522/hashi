package resource

import (
	"fmt"
	"strings"
)

type branchRule struct {
	check   func(string) bool
	message string
}

var branchRules = []branchRule{
	{func(n string) bool { return n == "" }, "branch name must not be empty"},
	{func(n string) bool { return strings.ContainsAny(n, " \t") }, "branch name contains whitespace"},
	{func(n string) bool {
		return strings.ContainsFunc(n, func(r rune) bool { return r < 0x20 || r == 0x7f })
	}, "branch name contains control character"},
	{func(n string) bool { return strings.ContainsAny(n, "~^*?[\\") }, "branch name contains invalid character"},
	{func(n string) bool { return strings.Contains(n, ":") }, "branch name contains ':'"},
	{func(n string) bool { return strings.Contains(n, "..") }, "branch name contains '..'"},
	{func(n string) bool { return strings.Contains(n, "@{") }, "branch name contains '@{'"},
	{func(n string) bool { return strings.HasPrefix(n, "-") }, "branch name must not start with '-'"},
	{func(n string) bool { return strings.HasPrefix(n, ".") }, "branch name must not start with '.'"},
	{func(n string) bool { return strings.HasSuffix(n, ".") }, "branch name must not end with '.'"},
	{func(n string) bool { return strings.HasSuffix(n, "/") }, "branch name must not end with '/'"},
	{func(n string) bool { return strings.Contains(n, "//") }, "branch name contains '//'"},
	{func(n string) bool { return strings.HasSuffix(n, ".lock") }, "branch name must not end with '.lock'"},
}

// ValidateBranchName checks that a branch name is safe for use with git and tmux.
func ValidateBranchName(name string) error {
	for _, r := range branchRules {
		if r.check(name) {
			return fmt.Errorf("%s", r.message)
		}
	}
	return nil
}
