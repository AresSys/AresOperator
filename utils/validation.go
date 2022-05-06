package utils

import (
	"fmt"
	"regexp"
)

var dns1123Pattern = regexp.MustCompile("^[a-z0-9]([-a-z0-9]*[a-z0-9])?$")

func MatchDNS1123(s string) error {
	if dns1123Pattern.MatchString(s) {
		return nil
	}
	return fmt.Errorf("dose not match '%s': %s", dns1123Pattern.String(), s)
}
