package dummy

import (
	"strings"
	"testing"
)

var ParseTests = []struct {
	in        string
	outname   string
	outparams []string
}{
	{
		"command:(?P<first>)(foo)(?P<second>)",
		"command",
		[]string{"first", "second"},
	},
}

func TestParse(t *testing.T) {
	for _, tt := range ParseTests {
		name, params, err := Parse(tt.in)
		if err != nil ||
			name != tt.outname ||
			strings.Join(params, "|") != strings.Join(tt.outparams, "|") {
			t.Errorf("Error in parse")
		}
		//Cleanup
		state = make([]Matcher, 0)
	}
}

var MatchTestsSuccess = []struct {
	source     string
	tomatch    string
	expCommand string
	expParams  map[string]string
	expError   error
}{
	{
		"command:(?P<first>[a-z]*) (?P<second>[a-z]*)",
		"prova uno",
		"command",
		map[string]string{"first": "prova", "second": "uno"},
		nil,
	},
}

//TODO test with many commands
func TestMatchSuccess(t *testing.T) {
	for _, tt := range MatchTestsSuccess {
		_, _, err := Parse(tt.source)
		if err != nil {
			t.Errorf("Error in parse")
		}
		name, values, err := Match(tt.tomatch)
		if err != tt.expError {
			t.Errorf("Unexpected error: " + err.Error())
		}
		if tt.expCommand != name {
			t.Errorf("Unexpected name: " + name)
		}
		if !areMapsEqual(tt.expParams, values) {
			t.Errorf("Unexpected params: %#v", values)
		}
		state = make([]Matcher, 0)
	}
}

func areMapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if val, ok := b[key]; !ok || val != value {
			return false
		}
	}
	return true
}
