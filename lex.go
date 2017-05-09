package vikyscript

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Pos int

// item represents a token or text string returned from the scanner.
type item struct {
	typ  itemType // The type of this item.
	pos  Pos      // The starting position, in bytes, of this item in the input string.
	val  string   // The value of this item.
	line int      // The line number at the start of this item.
}

func (i item) String() string {
	switch {
	case i.typ == itemEOF:
		return "EOF"
	case i.typ == itemError:
		return i.val
	case i.typ > itemKeyword:
		return fmt.Sprintf("<%s>", i.val)
	case len(i.val) > 10:
		return fmt.Sprintf("%.10q...", i.val)
	}
	return fmt.Sprintf("%q", i.val)
}

// itemType identifies the type of lex items.
type itemType int

const (
	itemError itemType = iota // error occurred; value is text of error
	itemEOF
	itemSpace // run of spaces separating arguments
	itemLeftList
	itemLeftParam
	itemRightList
	itemRightParam
	itemComma
	itemColon
	itemOptional
	itemShuffle
	itemIgnore
	itemLeftParen
	itemRightParen
	// Keywords appear after all the rest.
	itemKeyword     // used only to delimit the keywords
	itemCommandName // name of the command
	itemWord        // words
	itemListName
	itemParamName
	itemParamType
)

const eof = -1

// Trimming spaces.
// If the action begins "{{- " rather than "{{", then all space/tab/newlines
// preceding the action are trimmed; conversely if it ends " -}}" the
// leading spaces are trimmed. This is done entirely in the lexer; the
// parser never sees it happen. We require an ASCII space to be
// present to avoid ambiguity with things like "{{-3}}". It reads
// better with the space present anyway. For simplicity, only ASCII
// space does the job.
const (
	spaceChars      = " \t\r\n" // These are the space characters defined by Go itself.
	leftTrimMarker  = "- "      // Attached to left delimiter, trims trailing spaces from preceding text.
	rightTrimMarker = " -"      // Attached to right delimiter, trims leading spaces from following text.
	trimMarkerLen   = Pos(len(leftTrimMarker))
)

// stateFn represents the state of the scanner as a function that returns the next state.
type stateFn func(*lexer) stateFn

// lexer holds the state of the scanner.
type lexer struct {
	name       string    // the name of the input; used only for error reports
	input      string    // the string being scanned
	state      stateFn   // the next lexing function to enter
	pos        Pos       // current position in the input
	start      Pos       // start position of this item
	width      Pos       // width of last rune read from input
	lastPos    Pos       // position of most recent item returned by nextItem
	items      chan item // channel of scanned items
	parenDepth int       // nesting depth of ( ) exprs
	line       int       // 1+number of newlines seen
}

// next returns the next rune in the input.
func (l *lexer) next() rune {
	if int(l.pos) >= len(l.input) {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = Pos(w)
	l.pos += l.width
	if r == '\n' {
		l.line++
	}
	return r
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	w := l.width
	r := l.next()
	l.backup()
	l.width = w
	return r
}

// backup steps back one rune. Can only be called once per call of next.
func (l *lexer) backup() {
	l.pos -= l.width
	// Correct newline count.
	if l.width == 1 && l.input[l.pos] == '\n' {
		l.line--
	}
}

// emit passes an item back to the client.
func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.start, l.input[l.start:l.pos], l.line}
	// Some items contain text internally. If so, count their newlines.
	l.line += strings.Count(l.input[l.start:l.pos], "\n")
	l.start = l.pos
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
}

// accept consumes the next rune if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.ContainsRune(valid, l.next()) {
		return true
	}
	l.backup()
	return false
}

// acceptRun consumes a run of runes from the valid set.
func (l *lexer) acceptRun(valid string) {
	for strings.ContainsRune(valid, l.next()) {
	}
	l.backup()
}

// errorf returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating l.nextItem.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{itemError, l.start, fmt.Sprintf(format, args...), l.line}
	return nil
}

// nextItem returns the next item from the input.
// Called by the parser, not in the lexing goroutine.
func (l *lexer) nextItem() item {
	item := <-l.items
	l.lastPos = item.pos
	return item
}

// drain drains the output so the lexing goroutine will exit.
// Called by the parser, not in the lexing goroutine.
func (l *lexer) drain() {
	for range l.items {
	}
}

// lex creates a new scanner for the input string.
func lex(name, input string) *lexer {
	l := &lexer{
		name:  name,
		input: input,
		items: make(chan item),
		line:  1,
	}
	go l.run()
	return l
}

// run runs the state machine for the lexer.
func (l *lexer) run() {
	for l.state = lexCommandName; l.state != nil; {
		l.state = l.state(l)
	}
	close(l.items)
}

func (l *lexer) unexpectedChar(r rune) stateFn {
	if r == eof {
		return l.errorf("Unexpected EOF at character %d, line %d", l.pos, l.line)
	}
	return l.errorf("Unexpected %#U at character %d, line %d", r, l.pos, l.line)
}

// state functions

const (
	openParen   = '('
	closedParen = ')'
	openList    = '['
	closedList  = ']'
	openParam   = '{'
	closedParam = '}'
	nameDelim   = ':'
	listDelim   = ','
	shuffle     = '#'
	optional    = '?'
	ignore      = '*'
)

// lexCommandName scans until an opening action delimiter, "{{".
func lexCommandName(l *lexer) stateFn {
	l.width = 0
	for {
		switch r := l.next(); {
		case r == nameDelim:
			l.backup()
			l.emit(itemCommandName)
			l.next()
			l.emit(itemColon)
			return lexCommand
		case isAlphaNumeric(r) || isSpace(r):
			//keep going, the parser will handle whitespace
		default:
			return l.unexpectedChar(r)
		}
	}
}

func lexCommand(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case isSpace(r):
			return lexSpace
		case isAlphaNumeric(r):
			return lexWord
		case r == openList:
			l.emit(itemLeftList)
			return lexList
		case r == openParam:
			l.emit(itemLeftParam)
			return lexParam
		case r == eof:
			l.emit(itemEOF)
			if l.parenDepth != 0 {
				return l.errorf("Unexpected EOF: unmatched left paren")
			}
			return nil
		case r == optional:
			l.emit(itemOptional)
		case r == shuffle:
			l.emit(itemShuffle)
		case r == ignore:
			l.emit(itemIgnore)
		case r == openParen:
			l.emit(itemLeftParen)
			l.parenDepth++
		case r == closedParen:
			l.emit(itemRightParen)
			l.parenDepth--
			if l.parenDepth < 0 {
				return l.unexpectedChar(r)
			}
		default:
			return l.unexpectedChar(r)
		}
	}
}

func lexWord(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case unicode.IsLetter(r):
		default:
			l.backup()
			l.emit(itemWord)
			return lexCommand
		}
	}
}

func lexList(l *lexer) stateFn {
	for isSpace(l.peek()) {
		l.next()
		l.emit(itemSpace)
	}
	for {
		switch r := l.peek(); {
		case isAlphaNumeric(r) || isSpace(r):
			//either parsing the list name or the first element of the list
			//parser will be responsible of trimming/replacing whitespace
			l.next()
		case r == listDelim:
			//this is an unnamed list, let's emit the word and keep lexing
			l.emit(itemWord)
			l.next()
			l.emit(itemComma)
			return lexUnnamedList
		case r == nameDelim:
			l.emit(itemListName)
			l.next()
			l.emit(itemColon)
			return lexUnnamedList
		case r == closedList:
			l.emit(itemWord)
			l.next()
			l.emit(itemRightList)
			return lexCommand
		default:
			return l.unexpectedChar(r)
		}
	}
}

func lexUnnamedList(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case isAlphaNumeric(r) || isSpace(r):
			//keep going
		case r == listDelim:
			l.backup()
			l.emit(itemWord)
			l.next()
			l.ignore()
		case r == closedList:
			l.backup()
			l.emit(itemWord)
			l.next()
			l.emit(itemRightList)
			return lexCommand
		default:
			return l.unexpectedChar(r)
		}
	}
}

func lexParam(l *lexer) stateFn {
	for isSpace(l.peek()) {
		l.next()
		l.ignore()
	}
	for {
		switch r := l.peek(); {
		case isAlphaNumeric(r) || isSpace(r):
			//parsing the name of the command, parser will be responsible of
			//handling whitespace
			l.next()
		case r == nameDelim:
			l.emit(itemParamName)
			l.next()
			l.emit(itemColon)
			return lexTypedParam
		case r == closedParam:
			l.emit(itemParamName)
			l.next()
			l.emit(itemRightParam)
			return lexCommand
		default:
			return l.unexpectedChar(r)
		}
	}
}

func lexTypedParam(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case isAlphaNumeric(r) || isSpace(r):
			//parser will handle whitespace
			//this could be removed...
		case r == closedParam:
			l.backup()
			l.emit(itemParamType)
			l.next()
			l.emit(itemRightParam)
			return lexCommand
		default:
			return l.unexpectedChar(r)
		}
	}
}

// lexSpace scans a run of space characters.
// One space has already been seen.
func lexSpace(l *lexer) stateFn {
	for isSpace(l.peek()) {
		l.next()
	}
	l.emit(itemSpace)
	return lexCommand
}

// isSpace reports whether r is a space character.
func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

// isEndOfLine reports whether r is an end-of-line character.
func isEndOfLine(r rune) bool {
	return r == '\r' || r == '\n'
}

// isAlphaNumeric reports whether r is an alphabetic, digit, or underscore.
func isAlphaNumeric(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}
