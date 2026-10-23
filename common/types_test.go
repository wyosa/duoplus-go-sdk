package common

import (
	"encoding/json"
	"testing"
)

func TestFailReasonsAcceptsEmptyArraysWithWhitespace(t *testing.T) {
	for _, input := range []string{`[]`, `[ ]`, "[\n\t]", `null`, `{}`} {
		t.Run(input, func(t *testing.T) {
			out := OperationResult{FailReason: FailReasons{"old": "value"}}
			if err := json.Unmarshal([]byte(`{"success":["p1"],"fail_reason":`+input+`}`), &out); err != nil {
				t.Fatal(err)
			}
			if len(out.FailReason) != 0 || len(out.Success) != 1 {
				t.Fatalf("unexpected result: %+v", out)
			}
		})
	}
}

func TestFailReasonsRejectsInvalidShapes(t *testing.T) {
	for _, input := range []string{`["failure"]`, `[{}]`, `{"p1":42}`, `true`, `[`} {
		var out FailReasons
		if err := json.Unmarshal([]byte(input), &out); err == nil {
			t.Errorf("accepted invalid fail reasons %s", input)
		}
	}
	var out FailReasons
	if err := json.Unmarshal([]byte(`{"p1":"offline"}`), &out); err != nil || out["p1"] != "offline" {
		t.Fatalf("object decoding failed: %v, %v", out, err)
	}
}

func TestTextPreservesExactNumbers(t *testing.T) {
	for _, input := range []string{
		"9007199254740993", "-9007199254740993", "18446744073709551615",
		"0.1234567890123456789", "1e+30", "1e400", "-0",
	} {
		var out Text
		if err := json.Unmarshal([]byte(input), &out); err != nil {
			t.Fatal(err)
		}
		if string(out) != input {
			t.Errorf("%s became %s", input, out)
		}
	}
}

func TestTextRejectsInvalidValuesWithoutChangingReceiver(t *testing.T) {
	for _, input := range []string{`[]`, `{}`, `123 456`, `"x" garbage`, `01`, ``} {
		out := Text("unchanged")
		if err := out.UnmarshalJSON([]byte(input)); err == nil {
			t.Errorf("accepted %q", input)
		}
		if out != "unchanged" {
			t.Errorf("failed decode changed the receiver for %q", input)
		}
	}
	var out Text = "old"
	if err := out.UnmarshalJSON([]byte(" \n null \t")); err != nil || out != "" {
		t.Fatalf("null decoding failed: %q, %v", out, err)
	}
}
