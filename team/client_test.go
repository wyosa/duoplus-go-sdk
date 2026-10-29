package team_test

import (
	"encoding/json"
	"testing"

	"github.com/wyosa/duoplus-go-sdk/team"
)

func TestResourcesBalancePreservesDecimalPrecision(t *testing.T) {
	for input, want := range map[string]string{
		`9007199254740993.01`:   "9007199254740993.01",
		`"9007199254740993.01"`: "9007199254740993.01",
		`""`:                    "",
		`null`:                  "",
	} {
		var result team.Resources
		if err := json.Unmarshal([]byte(`{"balance":`+input+`}`), &result); err != nil {
			t.Fatalf("%s: %v", input, err)
		}
		if string(result.Balance) != want {
			t.Fatalf("%s: balance = %q, want %q", input, result.Balance, want)
		}
	}
}
