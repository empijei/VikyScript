package vikyscript

import (
	"fmt"
	"strings"
	"testing"
)

var itemNames = []string{
	"Error",
	"EOF",
	"Space",
	"LeftList",
	"LeftParam",
	"RightList",
	"RightParam",
	"Comma",
	"Colon",
	"Optional",
	"Shuffle",
	"Ignore",
	"LeftParen",
	"RightParen",
	"Keyword",
	"CommandName",
	"Word",
	"ListName",
	"ParamName",
	"ParamType",
}

var lexTests = []struct {
	name, input string
}{
	{"simple", "command:trial"},
	{"simplespace", "command : trial"},
	{"list", "command:[foo,bar]"},
	{"namedlist", "command:[which:foo,bar]"},
	{"param", "command:{foo}"},
	{"typedparam", "command:{foo:integer}"},
	{"listspace", "command:[foo foo,bar , lol]"},
	{"namedlistspace", "command:[which one : foo,bar]"},
	{"paramspace", "command:{ foo } "},
	{"typedparamspace", "command: { foo : integer }"},
	{"unary", "command: * ?(foo) bar #(foo bar) * ?foo"},
}

func TestCorrect(t *testing.T) {
	for _, tt := range lexTests {
		fmt.Printf("\nNow parsing: %s\n\t<%s>\n", tt.name, tt.input)
		l := lex(tt.name, tt.input)
		var i item
		for i = range l.items {
			fmt.Print(itemNames[i.typ] + " ")
			fmt.Println(i)
		}
		if i.typ == itemError {
			indicator := strings.Repeat(" ", int(l.pos))
			indicator += "^"
			t.Errorf("Failure while parsing %s:\n%s\n<%s>\n%s", tt.name, i, tt.input, indicator)
		}
	}
}

var lexErrorTests = []struct {
	name, input string
}{
	{"unmatched", "command:("},
	{"undeclared", "foo bar nope"},
	{"unexpected", "command:)"},
	{"unendedlist", "command:["},
	{"unendedparam", "command:{"},
	{"unendedlist", "command:[foo,"},
	{"unendedparam", "command:{bar:"},
}

func TestError(t *testing.T) {
	for _, tt := range lexErrorTests {
		fmt.Printf("\nNow parsing: %s\n\t<%s>\n", tt.name, tt.input)
		l := lex(tt.name, tt.input)
		var i item
		for i = range l.items {
			fmt.Print(itemNames[i.typ] + " ")
			fmt.Println(i)
		}
		if i.typ != itemError {
			t.Errorf("Expected error while parsing %s:\n<%s>", tt.name, tt.input)
		}
	}
}
