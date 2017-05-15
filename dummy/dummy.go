package dummy

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

var state []Matcher
var state_m sync.RWMutex

type Command struct {
	Name   string
	Params []string
	raw    string
}

type Matcher struct {
	*regexp.Regexp
	Command
}

//func (m *Matcher) Match(text string) (commandName string, paramValues []string [>Maybe use dictionary here?<], err error) {
//}

func Parse(code string) (commandName string, paramNames []string, err error) {
	var m Matcher
	m.raw = code
	var retext string
	if index := strings.Index(code, ":"); index > 0 {
		commandName = code[:index]
		retext = code[index+1:]
	} else {
		err = fmt.Errorf("Invalid command, no name specified")
		return
	}
	m.Name = commandName
	m.Regexp, err = regexp.Compile(retext)
	if err != nil {
		return
	}
	for _, p := range m.SubexpNames() {
		if p != "" {
			m.Params = append(m.Params, p)
		}
	}
	paramNames = m.Params
	state_m.Lock()
	defer state_m.Unlock()
	state = append(state, m)
	return
}

type ClashError struct {
	matches []string
}

func (ce *ClashError) Error() string {
	return fmt.Sprintf("Clash of commands: %s", strings.Join(ce.matches, ", "))
}

type MatchResult struct {
	m       Matcher
	results map[string]string
}

func Match(text string) (commandName string, paramValues map[string]string, err error) {
	pool := runtime.NumCPU()
	tomatch := make(chan Matcher)
	matchedChan := make(chan MatchResult)
	var wg sync.WaitGroup
	wg.Add(pool)
	state_m.RLock()
	defer state_m.RUnlock()
	//spawn workers
	for i := 0; i < pool; i++ {
		go func() {
			//Signal we finished processing data
			defer wg.Done()
			for tm := range tomatch {
				result := tm.FindStringSubmatch(text)
				if result == nil {
					continue
				}
				paramsMap := make(map[string]string)
				for i, name := range tm.SubexpNames() {
					if i > 0 && i <= len(result) {
						paramsMap[name] = result[i]
					}
				}
				matchedChan <- MatchResult{m: tm, results: paramsMap}
			}
		}()
	}
	//Send the matchers to the workers
	go func() {
		for _, m := range state {
			tomatch <- m
		}
		close(tomatch)
		//Let the matchers match
		wg.Wait()
		//Signal we are done
		close(matchedChan)
	}()
	var matchedList []MatchResult
	for m := range matchedChan {
		matchedList = append(matchedList, m)
	}
	switch l := len(matchedList); {
	case l > 1:
		var matches []string
		for _, m := range matchedList {
			matches = append(matches, m.m.Name)
		}
		err = &ClashError{
			matches: matches,
		}
		//Maybe fallthrough here? We might want to return the value of one of the
		//matches.
	case l == 1:
		commandName = matchedList[0].m.Name
		//paramValues = matched[0].
		paramValues = matchedList[0].results
	default:
		err = fmt.Errorf("No match")
	}
	return
}
