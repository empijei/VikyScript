package main

import "regexp"

type Recognizer struct {
	source string
	re     regexp.Regexp
}

func NewRecognizer(source string) *Recognizer {
	return &Recognizer{source: source}
}
func (r *Recognizer) Compile() {}
func (r *Recognizer) Match(what string) []string {
	return []string{""}
}
