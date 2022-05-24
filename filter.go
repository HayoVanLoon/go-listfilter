// Copyright 2022 Hayo van Loon. All rights reserved.
// Use of this source code is governed by an Apache
// license that can be found in the LICENSE file.

/*
Package listfilter implements a query parser and the resulting filter as used in
List and Search requests.

Syntax and semantics are reverse engineered from:
https://cloud.google.com/service-infrastructure/docs/service-consumer-management/reference/rest/v1/services/search#query-parameters

Examples of filter strings:

  "foo=bar"
  "foo.bar=bla"
  "foo=bar AND bla=vla"
  "foo>bar AND foo=bar"

The filter string should adher to the following grammar:

  Filter =        <nil> | Conditions
  Conditions =    Condition { Separator Conditions }
  Separator =     Space 'AND' Space
  Condition =     FullName Operator Value
  FullName =      NameParts
  NameParts =     Name | Name NameSeparator NameParts
  NameSeparator = '.'
  Name =          regex([a-zA-Z][a-zA-Z0-9_]*)
  Operator =      regex([^a-zA-Z0-9_].*)
  Value =         NormalValue | QuotedValue
  NormalValue =   [^separator\s"] { regex([^separator\s]*) }
  QuotedValue =   '"' Escaped '"'
  Escaped =       <nil> | NormalChar Escaped | EscapedChar Escaped
  EscapedChar =   '\\' | '\"' NormalChar | <not eChar>

An empty string is considered a valid input and will result in an empty Filter.
*/
package listfilter

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// A FilterParser parses a filter string into a Filter. If parsing fails, a
// ParseError is returned and the Filter will be nil.
type FilterParser interface {
	Parse(s string) (Filter, ParseError)
}

// Condition stores a filter condition.
type Condition interface {
	Key() string
	KeyParts() []string
	Op() string
	StringValue() string
	IntValue() (int, error)
	BoolValue() (bool, error)
	FloatValue() (float64, error)
	AndOr() (Condition, Condition)
}

type condition struct {
	key         string
	keyParts    []string
	op          string
	stringValue string
	nextAnd     *condition
	nextOr      *condition
}

// NewCondition creates a new Condition from the specified parameters.
func NewCondition(key string, keyParts []string, op, stringValue string) Condition {
	return condition{key, keyParts, op, stringValue, nil, nil}
}

// Key returns the condition's key.
func (c condition) Key() string {
	return c.key
}

// KeyParts returns the condition's key part list, which has at least one item.
func (c condition) KeyParts() []string {
	return c.keyParts
}

// Op returns the condition's operator as a string.
func (c condition) Op() string {
	return c.op
}

// StringValue returns the raw string value of the condition.
func (c condition) StringValue() string {
	return c.stringValue
}

// IntValue is a convenience function for getting a filter condition value as an
// integer. If the value is not an integer, an error is returned.
func (c condition) IntValue() (int, error) {
	i, err := strconv.Atoi(c.stringValue)
	if err != nil {
		return 0, fmt.Errorf("%s is not an integer", c.stringValue)
	}
	return i, nil
}

// BoolValue is a convenience function for getting a filter condition value as
// a boolean. If the value is not a strict boolean (case-insensitive 'true' or
// 'false'), an error is returned.
func (c condition) BoolValue() (bool, error) {
	switch strings.ToLower(c.stringValue) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	return false, fmt.Errorf("%s is not a valid boolean", c.stringValue)
}

// FloatValue is a convenience function for getting a filter condition value as
// a 64-bit float. If the value is not a float, an error is returned.
func (c condition) FloatValue() (float64, error) {
	f, err := strconv.ParseFloat(c.stringValue, 64)
	if err != nil {
		return 0, fmt.Errorf("%s is not a valid float", c.stringValue)
	}
	return f, nil
}

func (c condition) and() Condition {
	if c.nextAnd == (*condition)(nil) {
		return nil
	}
	return c.nextAnd
}

func (c condition) or() Condition {
	if c.nextOr == (*condition)(nil) {
		return nil
	}
	return c.nextOr
}

// AndOr returns the next condition in the filter. It returns a tuple; the
// first points to an AND condition, the second to an OR.
func (c condition) AndOr() (Condition, Condition) {
	return c.and(), c.or()
}

func (c condition) String() string {
	return fmt.Sprintf("%s%s%s (%v,%v)", strings.Join(c.keyParts, "."), c.op, c.stringValue, c.nextAnd, c.nextOr)
}

// A ParseError describes the error that occurred while parsing. In addition, it
// provides details to help pinpoint the error.
type ParseError interface {
	error
	Message() string
	Position() int
	Unparsable() string
}

type parseError struct {
	message    string
	position   int
	unparsable string
}

// NewParseError returns a new ParseError with the specified parameters.
func NewParseError(message string, position int, unparsable string) ParseError {
	return &parseError{message, position, unparsable}
}

// Message provides a user-friendly error message.
func (pe *parseError) Message() string {
	return pe.message
}

// Position returns the position in the string at which parsing failed.
func (pe *parseError) Position() int {
	return pe.position
}

// Unparsable returns the part of the string from which parsing failed.
func (pe *parseError) Unparsable() string {
	return pe.unparsable
}

func (pe *parseError) Error() string {
	return fmt.Sprintf("%s @ %d (%s)", pe.message, pe.position, pe.unparsable)
}

type Filter interface {
	Get(k string) ([]Condition, bool)
	GetFirst(k string) (Condition, bool)
	GetLast(k string) (Condition, bool)
	Len() int
	FirstCondition() Condition
	Conditions() Iterator[Condition]
}

type filter struct {
	m     map[string][]Condition
	first Condition
}

// Get retrieves the conditions for a given key.
func (f filter) Get(k string) ([]Condition, bool) {
	cs, ok := f.m[k]
	return cs, ok
}

// GetFirst retrieves the first condition for a given key.
func (f filter) GetFirst(k string) (Condition, bool) {
	if cs := f.m[k]; cs != nil {
		return cs[0], true
	}
	return nil, false
}

// GetLast retrieves the last condition for a given key.
func (f filter) GetLast(k string) (Condition, bool) {
	if cs := f.m[k]; cs != nil {
		return cs[len(cs)-1], true
	}
	return nil, false
}

// Len returns the number of keys in the filter (not the number of conditions!).
func (f filter) Len() int {
	return len(f.m)
}

// FirstCondition returns the first condition in the filter.
func (f filter) FirstCondition() Condition {
	return f.first
}

// Conditions returns an iterator that iterates over all conditions in the
// filter, starting with the first.
func (f filter) Conditions() Iterator[Condition] {
	var cs []Condition
	c := f.FirstCondition()
	if c == nil {
		return Empty[Condition]()
	}
	cs = append(cs, c)

	for and, or := c.AndOr(); and != nil || or != nil; and, or = c.AndOr() {
		if and != nil {
			c = and
		} else if or != nil {
			c = or
		}
		cs = append(cs, c)
	}

	ch := make(chan Condition, len(cs))
	for _, c := range cs {
		ch <- c
	}
	close(ch)

	return ForFunc[Condition](func() (Condition, error) {
		select {
		case v, ok := <-ch:
			if !ok {
				return nil, Done
			}
			return v, nil
		default:
			return nil, Done
		}
	})
}

type filterParser struct {
	ops       map[string]bool
	snakeCase bool
	camelCase bool
}

// NewParser creates a new FilterParser.
func NewParser(options ...Option) FilterParser {
	f := &filterParser{ops: map[string]bool{"=": true, "!=": true}}
	for _, opt := range options {
		opt.Apply(f)
	}
	if f.camelCase && f.snakeCase {
		panic("conflicting options for name casing")
	}
	return f
}

// Parse parses a filter string into a Filter.
func (p *filterParser) Parse(s string) (Filter, ParseError) {
	if len(s) == 0 {
		return filter{}, nil
	}
	filter, _, err := p.parseConditions(s, 0)
	if err != nil {
		return nil, err
	}
	return filter, nil
}

const (
	separator       = "AND"
	nameSeparator   = '.'
	escapeCharacter = '\\'
	quote           = '"'
)

func (p *filterParser) parseConditions(s string, start int) (filter, int, ParseError) {
	cond, i, err := p.parseCondition(s, start)
	if err != nil {
		return filter{}, i, err
	}
	f := filter{make(map[string][]Condition), cond}
	if i == len(s) {
		f.m[cond.key] = []Condition{cond}
		return f, i, nil
	}
	prev := cond
	for i < len(s) {
		_, i, err = parseSeparator(s, i)
		if err != nil {
			return filter{}, i, err
		}
		cond, i, err = p.parseCondition(s, i)
		if err != nil {
			return filter{}, i, err
		}
		prev.nextAnd = &cond
		f.m[prev.key] = append(f.m[prev.key], prev)
		prev = cond
	}
	f.m[prev.key] = append(f.m[prev.key], prev)
	return f, start, nil
}

func parseSeparator(s string, start int) (string, int, ParseError) {
	i := start
	for ; i < len(s) && unicode.IsSpace(rune(s[i])); i += 1 {
	}
	if i == start {
		return "", i, NewParseError("expected a whitespace", i, s[i:])
	}
	if len(s) < i+len(separator) {
		return "", i, NewParseError("expected AND", i, s[i:])
	}
	j := i + len(separator)
	sep := s[i:j]
	if sep != separator {
		return "", i, NewParseError("expected AND", i, s[i:])
	}
	for ; j < len(s) && unicode.IsSpace(rune(s[j])); j += 1 {
	}
	if j == i {
		return "", j, NewParseError("expected a whitespace", j, s[j:])
	}
	return sep, j, nil
}

func (p *filterParser) parseCondition(s string, start int) (condition, int, ParseError) {
	key, keyParts, i, err := p.parseFullName(s, start)
	if err != nil {
		return condition{}, i, err
	}
	op, i, err := p.parseOperator(s, i)
	if err != nil {
		return condition{}, i, err
	}
	value, i, err := p.parseValue(s, i)
	if err != nil {
		return condition{}, i, err
	}
	return condition{key, keyParts, op, value, nil, nil}, i, nil
}

func (p *filterParser) parseFullName(s string, start int) (string, []string, int, ParseError) {
	parts, i, err := p.parseNameParts(s, start)
	if err != nil {
		return "", nil, i, err
	}
	return strings.Join(parts, string(nameSeparator)), parts, i, nil
}

func (p *filterParser) parseNameParts(s string, start int) ([]string, int, ParseError) {
	part, i, err := p.parseName(s, start)
	if err != nil {
		return nil, i, err
	}
	parts := []string{part}
	for i < len(s) && s[i] == nameSeparator {
		i += 1
		part, i, err = p.parseName(s, i)
		if err != nil {
			return nil, i, err
		}
		parts = append(parts, part)
	}
	return parts, i, nil
}

func (p *filterParser) parseName(s string, start int) (string, int, ParseError) {
	if len(s) == start {
		return "", start, NewParseError("unexpected end of string, expected a name", start, s[start:])
	}
	if !unicode.IsLetter(rune(s[start])) {
		return "", start, NewParseError("name must start with letter", start, s[start:])
	}
	i := start + 1
	for ; i < len(s); i += 1 {
		if unicode.IsLetter(rune(s[i])) {
			continue
		}
		if unicode.IsNumber(rune(s[i])) {
			continue
		}
		if s[i] == '_' {
			continue
		}
		break
	}
	if p.snakeCase {
		return snakeCase(s[start:i]), i, nil
	}
	if p.camelCase {
		return camelCase(s[start:i]), i, nil
	}
	return s[start:i], i, nil
}

func (p *filterParser) parseOperator(s string, start int) (string, int, ParseError) {
	i := start
	for i < len(s) {
		i += 1
		if v := s[start:i]; p.ops[v] {
			return v, i, nil
		}
	}
	return "", i, NewParseError("expected operator", start, s[start:])
}

func (p *filterParser) parseValue(s string, start int) (string, int, ParseError) {
	if start == len(s) {
		return "", start, nil
	}
	if s[start] == quote {
		return p.parseQuotedValue(s, start)
	}
	return p.parseNormalValue(s, start)
}

func (p *filterParser) parseNormalValue(s string, start int) (string, int, ParseError) {
	i := start
	w := 0
	for ; i < len(s); i += w {
		r, width := utf8.DecodeRuneInString(s[i:])
		if unicode.IsSpace(r) {
			break
		}
		w = width
	}
	return s[start:i], i, nil
}

func (p *filterParser) parseQuotedValue(s string, start int) (string, int, ParseError) {
	i := start + 1
	v, i, err := p.parseQuotesEscaped(s, i)
	if err != nil {
		return v, i, err
	}
	if len(s) == i || s[i] != quote {
		return "", start, NewParseError("unterminated quoted value", start, s[start:])
	}
	return v, i + 1, nil
}

func (p *filterParser) parseQuotesEscaped(s string, start int) (string, int, ParseError) {
	sb := strings.Builder{}
	i := start
	escape := false
	w := 0
	for ; i < len(s); i += w {
		r, width := utf8.DecodeRuneInString(s[i:])
		if escape {
			switch r {
			case quote, escapeCharacter:
			default:
				// no special meaning, add escape character retroactively
				sb.WriteRune(escapeCharacter)
			}
			escape = false
		} else if r == quote {
			break
		} else if r == escapeCharacter {
			escape = true
			w = width
			continue
		}
		sb.WriteRune(r)
		w = width
	}
	return sb.String(), i, nil
}

// An Option that can be passed to the FilterParser factory method.
type Option interface {
	Apply(parser *filterParser)
}

type optionSnakeCase struct{}

func (o optionSnakeCase) Apply(parser *filterParser) {
	parser.snakeCase = true
}

// OptionSnakeCase will instruct the parser to make a best-effort attempt at
// converting field names to snake_case. Cannot be used along with
//OptionCamelCase.
// When an uppercase character is encountered, it will be lower-cased. It will
// be prefixed with an underscore, unless it is the starting character, preceded
// by another uppercase character, or preceded by an underscore.
func OptionSnakeCase() Option {
	return &optionSnakeCase{}
}

type optionCamelCase struct{}

func (o optionCamelCase) Apply(parser *filterParser) {
	parser.camelCase = true
}

// OptionCamelCase will instruct the parser to make a best-effort attempt at
// converting field names to camelCase. Cannot be used along with
// OptionSnakeCase.
func OptionCamelCase() Option {
	return &optionCamelCase{}
}

func snakeCase(s string) string {
	sb := strings.Builder{}
	underscore := true
	for _, c := range s {
		if unicode.IsUpper(c) {
			if !underscore {
				sb.WriteRune('_')
			}
			sb.WriteRune(unicode.ToLower(c))
			underscore = true
		} else {
			sb.WriteRune(c)
			underscore = c == '_'
		}
	}
	return sb.String()
}

func camelCase(s string) string {
	sb := strings.Builder{}
	underscore := false
	upper := true
	for _, c := range s {
		if c == '_' {
			underscore, upper = true, false
			continue
		}

		if underscore {
			sb.WriteRune(unicode.ToUpper(c))
			underscore, upper = false, true
			continue
		}

		if upper {
			sb.WriteRune(unicode.ToLower(c))
		} else {
			sb.WriteRune(c)
		}

		upper = unicode.IsUpper(c)
	}
	return sb.String()
}
