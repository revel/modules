package app

// Validate the ace file

// External ace has no accessable validation methods, so this is just a copy and paste of a couple of reads that
// occur within ace to

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/yosssi/ace"
	"strings"
)

const unicodeSpace = 32

const indentTop = 0

// Special characters
const (
	cr   = "\r"
	lf   = "\n"
	crlf = "\r\n"

	space        = " "
	equal        = "="
	pipe         = "|"
	doublePipe   = pipe + pipe
	slash        = "/"
	sharp        = "#"
	dot          = "."
	doubleDot    = dot + dot
	colon        = ":"
	doubleColon  = colon + colon
	doubleQuote  = `"`
	lt           = "<"
	gt           = ">"
	exclamation  = "!"
	hyphen       = "-"
	bracketOpen  = "["
	bracketClose = "]"
)

func AceValidate(fileData []byte) (line int, err error) {
	bytes.NewBuffer(fileData)
	reader := bytes.NewReader(fileData)
	scanner := bufio.NewReader(reader)

	for {
		var inline []byte
		inline, _, err = scanner.ReadLine()
		if err != nil {
			return
		}
		o := &ace.Options{}
		ace.InitializeOptions(o)
		parsedLine := newLine(line, string(inline), o, nil)
		if !parsedLine.isEmpty() &&
			!parsedLine.isHelperMethod() &&
			!parsedLine.isPlainText() &&
			!parsedLine.isComment() &&
			!parsedLine.isHTMLComment() &&
			!parsedLine.isAction() {
			err = fmt.Errorf("Unknow line entry '%s' at %d", string(inline), line)
			return
		}
		line++

	}

}

// line represents a line of codes.
type line struct {
	no     int
	str    string
	indent int
	tokens []string
	opts   *ace.Options
	file   *ace.File
}

// isEmpty returns true if the line is empty.
func (l *line) isEmpty() bool {
	return strings.TrimSpace(l.str) == ""
}

// isTopIndent returns true if the line's indent is the top level.
func (l *line) isTopIndent() bool {
	return l.indent == indentTop
}

// isHelperMethod returns true if the line is a helper method.
func (l *line) isHelperMethod() bool {
	return len(l.tokens) > 1 && l.tokens[0] == equal
}

// isHelperMethodOf returns true if the line is a specified helper method.
func (l *line) isHelperMethodOf(name string) bool {
	return l.isHelperMethod() && l.tokens[1] == name
}

// isPlainText returns true if the line is a plain text.
func (l *line) isPlainText() bool {
	return len(l.tokens) > 0 && (l.tokens[0] == pipe || l.tokens[0] == doublePipe)
}

// isComment returns true if the line is a comment.
func (l *line) isComment() bool {
	return len(l.tokens) > 0 && l.tokens[0] == slash
}

// isHTMLComment returns true if the line is an HTML comment.
func (l *line) isHTMLComment() bool {
	return len(l.tokens) > 0 && l.tokens[0] == slash+slash
}

// isAction returns true if the line is an action.
func (l *line) isAction() bool {
	str := strings.TrimSpace(l.str)
	return strings.HasPrefix(str, l.opts.DelimLeft) && strings.HasSuffix(str, l.opts.DelimRight)
}

// newLine creates and returns a line.
func newLine(no int, str string, opts *ace.Options, f *ace.File) *line {
	return &line{
		no:     no,
		str:    str,
		indent: indent(str),
		tokens: strings.Split(strings.TrimLeft(str, space), space),
		opts:   opts,
		file:   f,
	}
}

// indent returns the line's indent.
func indent(str string) int {
	var i int

	for _, b := range str {
		if b != unicodeSpace {
			break
		}
		i++
	}

	return i / 2
}
