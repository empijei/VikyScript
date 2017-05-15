package vikyscript

import (
	"fmt"
	"runtime"
	"strings"
)

// Tree is the representation of a single parsed template.
type Tree struct {
	Name      string    // name of the template represented by the tree.
	ParseName string    // name of the top-level template during parsing, for error messages.
	Root      *ListNode // top-level root of the tree.
	text      string    // text parsed to create the template (or its parent)
	// Parsing only; cleared after parse.
	lex       *lexer
	token     [3]item // three-token lookahead for parser.
	peekCount int
	treeSet   map[string]*Tree
}

// Copy returns a copy of the Tree. Any parsing state is discarded.
func (t *Tree) Copy() *Tree {
	if t == nil {
		return nil
	}
	return &Tree{
		Name:      t.Name,
		ParseName: t.ParseName,
		Root:      t.Root.CopyList(),
		text:      t.text,
	}
}

func Parse(name, text string) (map[string]*Tree, error) {
	treeSet := make(map[string]*Tree)
	t := New(name)
	t.text = text
	_, err := t.Parse(text, treeSet)
	return treeSet, err
}

// next returns the next token.
func (t *Tree) next() item {
	if t.peekCount > 0 {
		t.peekCount--
	} else {
		t.token[0] = t.lex.nextItem()
	}
	return t.token[t.peekCount]
}

// backup backs the input stream up one token.
func (t *Tree) backup() {
	t.peekCount++
}

// backup2 backs the input stream up two tokens.
// The zeroth token is already there.
func (t *Tree) backup2(t1 item) {
	t.token[1] = t1
	t.peekCount = 2
}

// backup3 backs the input stream up three tokens
// The zeroth token is already there.
func (t *Tree) backup3(t2, t1 item) { // Reverse order: we're pushing back.
	t.token[1] = t1
	t.token[2] = t2
	t.peekCount = 3
}

// peek returns but does not consume the next token.
func (t *Tree) peek() item {
	if t.peekCount > 0 {
		return t.token[t.peekCount-1]
	}
	t.peekCount = 1
	t.token[0] = t.lex.nextItem()
	return t.token[0]
}

// nextNonSpace returns the next non-space token.
func (t *Tree) nextNonSpace() (token item) {
	for {
		token = t.next()
		if token.typ != itemSpace {
			break
		}
	}
	return token
}

// peekNonSpace returns but does not consume the next non-space token.
func (t *Tree) peekNonSpace() (token item) {
	for {
		token = t.next()
		if token.typ != itemSpace {
			break
		}
	}
	t.backup()
	return token
}

// Parsing.

// New allocates a new parse tree with the given name.
func New(name string, funcs ...map[string]interface{}) *Tree {
	return &Tree{
		Name: name,
	}
}

// ErrorContext returns a textual representation of the location of the node in the input text.
// The receiver is only used when the node does not have a pointer to the tree inside,
// which can occur in old code.
func (t *Tree) ErrorContext(n Node) (location, context string) {
	pos := int(n.Position())
	tree := n.tree()
	if tree == nil {
		tree = t
	}
	text := tree.text[:pos]
	byteNum := strings.LastIndex(text, "\n")
	if byteNum == -1 {
		byteNum = pos // On first line.
	} else {
		byteNum++ // After the newline.
		byteNum = pos - byteNum
	}
	lineNum := 1 + strings.Count(text, "\n")
	context = n.String()
	if len(context) > 20 {
		context = fmt.Sprintf("%.20s...", context)
	}
	return fmt.Sprintf("%s:%d:%d", tree.ParseName, lineNum, byteNum), context
}

// errorf formats the error and terminates processing.
func (t *Tree) errorf(format string, args ...interface{}) {
	t.Root = nil
	format = fmt.Sprintf("template: %s:%d: %s", t.ParseName, t.token[0].line, format)
	panic(fmt.Errorf(format, args...))
}

// error terminates processing.
func (t *Tree) error(err error) {
	t.errorf("%s", err)
}

// expect consumes the next token and guarantees it has the required type.
func (t *Tree) expect(expected itemType, context string) item {
	token := t.nextNonSpace()
	if token.typ != expected {
		t.unexpected(token, context)
	}
	return token
}

// expectOneOf consumes the next token and guarantees it has one of the required types.
func (t *Tree) expectOneOf(expected1, expected2 itemType, context string) item {
	token := t.nextNonSpace()
	if token.typ != expected1 && token.typ != expected2 {
		t.unexpected(token, context)
	}
	return token
}

// unexpected complains about the token and terminates processing.
func (t *Tree) unexpected(token item, context string) {
	t.errorf("unexpected %s in %s", token, context)
}

// recover is the handler that turns panics into returns from the top level of Parse.
func (t *Tree) recover(errp *error) {
	e := recover()
	if e != nil {
		if _, ok := e.(runtime.Error); ok {
			panic(e)
		}
		if t != nil {
			t.lex.drain()
			t.stopParse()
		}
		*errp = e.(error)
	}
	return
}

// startParse initializes the parser, using the lexer.
func (t *Tree) startParse(lex *lexer, treeSet map[string]*Tree) {
	t.Root = nil
	t.lex = lex
	t.treeSet = treeSet
}

// stopParse terminates parsing.
func (t *Tree) stopParse() {
	t.lex = nil
	t.treeSet = nil
}

// Parse parses the template definition string to construct a representation of
// the template for execution. If either action delimiter string is empty, the
// default ("{{" or "}}") is used. Embedded template definitions are added to
// the treeSet map.
func (t *Tree) Parse(text string, treeSet map[string]*Tree) (tree *Tree, err error) {
	defer t.recover(&err)
	t.ParseName = t.Name
	t.startParse(lex(t.Name, text), treeSet)
	t.text = text
	t.parse()
	t.add()
	t.stopParse()
	return t, nil
}

// add adds tree to t.treeSet.
func (t *Tree) add() {
	tree := t.treeSet[t.Name]
	if tree == nil || IsEmptyTree(tree.Root) {
		t.treeSet[t.Name] = t
		return
	}
	if !IsEmptyTree(t.Root) {
		t.errorf("template: multiple definition of template %q", t.Name)
	}
}

// IsEmptyTree reports whether this tree (node) is empty of everything but space.
func IsEmptyTree(n Node) bool {
	panic("not implemented")
}

// parse is the top-level parser for a template, essentially the same
// as itemList except it also parses {{define}} actions.
// It runs to EOF.
func (t *Tree) parse() {
	t.Root = t.newList(t.peek().pos)
	for t.peek().typ != itemEOF {
		/*
			if t.peek().typ == itemLeftDelim {
				delim := t.next()
				if t.nextNonSpace().typ == itemDefine {
					newT := New("definition") // name will be updated once we know it.
					newT.text = t.text
					newT.ParseName = t.ParseName
					newT.startParse(t.funcs, t.lex, t.treeSet)
					newT.parseDefinition()
					continue
				}
				t.backup2(delim)
			}
			switch n := t.textOrAction(); n.Type() {
			case nodeEnd, nodeElse:
				t.errorf("unexpected %s", n)
			default:
				t.Root.append(n)
			}
		*/
	}

}
