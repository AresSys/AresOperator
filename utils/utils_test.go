package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMatchDNS1123(t *testing.T) {
	cases := []struct {
		str     string
		matches bool
	}{
		{
			str:     "PsWorker",
			matches: false,
		}, {
			str:     "psWorker",
			matches: false,
		}, {
			str:     "ps-worker",
			matches: true,
		},
	}
	for i, c := range cases {
		err := MatchDNS1123(c.str)
		t.Logf("case #%d: matches=%v, error=%v", i, c.matches, err)
		if c.matches {
			assert.Nil(t, err)
		} else {
			assert.NotNil(t, err)
		}
	}
}
