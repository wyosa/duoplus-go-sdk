package team_test

import (
	"encoding/json"
	"testing"

	"github.com/wyosa/duoplus-go-sdk/team"
)

func TestResourcesBalancePreservesDecimalPrecision(t *testing.T) {
	for _, input := range []string{`9007199254740993.01`, `"9007199254740993.01"`} {
		var result team.Resources
		if err := json.Unmarshal([]byte(`{"balance":`+input+`}`), &result); err != nil {
			t.Fatal(err)
		}
		if result.Balance.String() != "9007199254740993.01" {
			t.Fatalf("balance rounded to %s", result.Balance)
		}
	}
}
