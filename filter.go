// Copyright 2022 Hayo van Loon. All rights reserved.
// Use of this source code is governed by an Apache
// license that can be found in the LICENSE file.

/*
Package listfilter implements a query parser and the resulting filter as used in
List and Search requests.

Syntax and semantics are reverse engineered (and expanded upon) from:
https://cloud.google.com/service-infrastructure/docs/service-consumer-management/reference/rest/v1/services/search#query-parameters

Examples of filter strings:

  "foo=bar"
  "foo.bar=bla"
  "foo=bar AND bla=vla"
  "foo>bar AND foo=bar"
  "foo>bar AND foo=bar OR moo=boo"

The filter string should adher to the following grammar:

  Filter =        <nil> | Conditions
  Conditions =    Condition { Separator Conditions }
  Separator =     Space SeparatorToken Space
  SeparatorToken  'AND' | 'OR'
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

// A Parser parses a filter string into a Filter. If parsing fails, an error is
// returned and the Filter will be nil. The error will be a ParseError, which
// has methods for diagnosing the parsing failure.
type Parser interface {
	// Parse parses a filter string into a Filter.
	Parse(s string) (Filter, error)
}

// Condition stores a filter condition.
type Condition interface {
	// Key returns the condition's key.
	Key() string
	// KeyParts returns the condition's key part list, which has at least one item.
	KeyParts() []string
	// Op returns the condition's operator as a string.
	Op() string
	// StringValue returns the raw string value of the condition.
	StringValue() string
	// IntValue is a convenience function for getting a filter condition value as an
	// integer. If the value is not an integer, an error is returned.
	IntValue() (int, error)
	// BoolValue is a convenience function for getting a filter condition value as
	// a boolean. If the value is not a strict boolean (case-insensitive 'true' or
	// 'false'), an error is returned.
	BoolValue() (bool, error)
	// FloatValue is a convenience function for getting a filter condition value as
	// a 64-bit float. If the value is not a float, an error is returned.
	FloatValue() (float64, error)
	// And returns the next AND Condition, if there is one, nil otherwise.
	And() Condition
	// Or returns the next OR Condition, if there is one, nil otherwise.
	Or() Condition
	// AndOr returns the next condition in the filter. It returns a tuple; the
	// first points to an AND condition, the second to an OR.
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

func (c condition) Key() string {
	return c.key
}

func (c condition) KeyParts() []string {
	return c.keyParts
}

func (c condition) Op() string {
	return c.op
}

func (c condition) StringValue() string {
	return c.stringValue
}

func (c condition) IntValue() (int, error) {
	i, err := strconv.Atoi(c.stringValue)
	if err != nil {
		return 0, fmt.Errorf("%s is not an integer", c.stringValue)
	}
	return i, nil
}

func (c condition) BoolValue() (bool, error) {
	switch strings.ToLower(c.stringValue) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	return false, fmt.Errorf("%s is not a valid boolean", c.stringValue)
}

func (c condition) FloatValue() (float64, error) {
	f, err := strconv.ParseFloat(c.stringValue, 64)
	if err != nil {
		return 0, fmt.Errorf("%s is not a valid float", c.stringValue)
	}
	return f, nil
}

func (c condition) And() Condition {
	if c.nextAnd == (*condition)(nil) {
		return nil
	}
	return c.nextAnd
}

func (c condition) Or() Condition {
	if c.nextOr == (*condition)(nil) {
		return nil
	}
	return c.nextOr
}

func (c condition) AndOr() (Condition, Condition) {
	return c.And(), c.Or()
}

func (c condition) String() string {
	return fmt.Sprintf("%s%s%s", c.key, c.op, c.stringValue)
}

// A ParseError describes the error that occurred while parsing. In addition, it
// provides details to help pinpoint the error.
type ParseError interface {
	error
	// Message provides a user-friendly error message.
	Message() string
	// Position returns the position in the string at which parsing failed.
	Position() int
	// Unparsable returns the part of the string from which parsing failed.
	Unparsable() string
}

type parseError struct {
	message    string
	position   int
	unparsable string
}

// newParseError returns a new ParseError with the specified parameters.
func newParseError(message string, position int, unparsable string) error {
	return &parseError{message, position, unparsable}
}

func (pe *parseError) Message() string {
	return pe.message
}

func (pe *parseError) Position() int {
	return pe.position
}

func (pe *parseError) Unparsable() string {
	return pe.unparsable
}

func (pe *parseError) Error() string {
	return fmt.Sprintf("%s @ %d (%s)", pe.message, pe.position, pe.unparsable)
}

// A Filter is a container for filter conditions as parsed by the Parser.
type Filter interface {
	// Get retrieves the conditions for a given key.
	Get(k string) ([]Condition, bool)
	// GetFirst retrieves the first condition for a given key.
	GetFirst(k string) (Condition, bool)
	// GetLast retrieves the last condition for a given key.
	GetLast(k string) (Condition, bool)
	// Keys returns all Condition keys found in the filter.
	Keys() []string
	// Values returns every Condition found in the filter. Other than that
	// conditions are grouped in blocks with the same key, there are no guarantees
	// on ordering. If for instance insertion order is required, use Conditions.
	Values() []Condition
	// Len returns the number of keys in the filter. This is may be less than
	// the total number of conditions.
	Len() int
	// First returns the first condition (as encountered in the original string).
	// Starting from this Condition and moving through its Condition.AndOr method
	// will allow reconstruction of the original filter string.
	First() Condition
	// Conditions returns all conditions by order of appearance in the original
	// filter string.
	Conditions() []Condition

	fmt.Stringer
}

type filter struct {
	m     map[string][]Condition
	first *condition
}

func (f filter) Keys() []string {
	var ks []string
	for k := range f.m {
		ks = append(ks, k)
	}
	return ks
}

func (f filter) Values() []Condition {
	var ys []Condition
	for _, xs := range f.m {
		for _, x := range xs {
			ys = append(ys, x)
		}
	}
	return ys
}

func (f filter) Get(k string) ([]Condition, bool) {
	cs, ok := f.m[k]
	return cs, ok
}

func (f filter) GetFirst(k string) (Condition, bool) {
	if cs := f.m[k]; cs != nil {
		return cs[0], true
	}
	return nil, false
}

func (f filter) GetLast(k string) (Condition, bool) {
	if cs := f.m[k]; cs != nil {
		return cs[len(cs)-1], true
	}
	return nil, false
}

func (f filter) Len() int {
	return len(f.m)
}

func (f filter) First() Condition {
	return f.first
}

func (f filter) Conditions() []Condition {
	c := f.First()
	if c == (*condition)(nil) {
		return nil
	}
	var cs []Condition
	for {
		cs = append(cs, c)
		and, or := c.AndOr()
		if and != nil {
			c = and
		} else if or != nil {
			c = or
		} else {
			break
		}
	}
	return cs
}

func (f filter) String() string {
	b := strings.Builder{}
	c := f.First()
	if c == (*condition)(nil) {
		return b.String()
	}
	for {
		b.WriteString(c.(*condition).String())
		and, or := c.AndOr()
		if and != nil {
			b.WriteString(" " + separatorAnd + " ")
			c = and
		} else if or != nil {
			b.WriteString(" " + separatorOr + " ")
			c = or
		} else {
			break
		}
	}
	return b.String()
}

type parser struct {
	ops       map[string]bool
	snakeCase bool
	camelCase bool
}

// NewParser creates a new Parser.
func NewParser(options ...Option) Parser {
	f := &parser{ops: map[string]bool{"=": true, "!=": true}}
	for _, opt := range options {
		opt.Apply(f)
	}
	if f.camelCase && f.snakeCase {
		panic("conflicting options for name casing")
	}
	return f
}

var emptyFilter = filter{m: make(map[string][]Condition)}

func (p *parser) Parse(s string) (Filter, error) {
	if len(s) == 0 {
		return emptyFilter, nil
	}
	f, _, err := p.parseConditions(s, 0)
	if err != nil {
		return nil, err
	}
	return f, nil
}

const (
	nameSeparator   = '.'
	escapeCharacter = '\\'
	quote           = '"'
)

const (
	separatorAnd = "AND"
	separatorOr  = "OR"
)

func (p *parser) parseConditions(s string, start int) (filter, int, error) {
	f := filter{m: make(map[string][]Condition)}
	first, i, err := p.parseCondition(s, start)
	if err != nil {
		return emptyFilter, i, err
	}
	f.first = &first
	prev := f.first
	for i < len(s) {
		var sep string
		sep, i, err = parseSeparator(s, i)
		if err != nil {
			return emptyFilter, i, err
		}
		var cond condition
		cond, i, err = p.parseCondition(s, i)
		if err != nil {
			return emptyFilter, i, err
		}
		if sep == separatorAnd {
			prev.nextAnd = &cond
		} else {
			prev.nextOr = &cond
		}
		f.m[prev.key] = append(f.m[prev.key], *prev)
		prev = &cond
	}
	f.m[prev.key] = append(f.m[prev.key], *prev)
	return f, start, nil
}

func spaceOrNonSpace(s string, start int, space bool) int {
	i := start
	for i < len(s) {
		r, width := utf8.DecodeRuneInString(s[i:])
		if unicode.IsSpace(r) != space {
			return i
		}
		i += width
	}
	return i
}

func parseSeparator(s string, start int) (string, int, error) {
	i := spaceOrNonSpace(s, start, true)
	if i == start {
		return "", i, newParseError("expected a whitespace", i, s[i:])
	}
	j := spaceOrNonSpace(s, i, false)
	sep := s[i:j]
	if !(sep == separatorAnd || sep == separatorOr) {
		return "", i, newParseError("expected a condition separator (AND, OR)", i, s[i:])
	}
	k := spaceOrNonSpace(s, j, true)
	if k == j {
		return "", k, newParseError("expected a whitespace", k, s[k:])
	}
	return sep, k, nil
}

func (p *parser) parseCondition(s string, start int) (condition, int, error) {
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

func (p *parser) parseFullName(s string, start int) (string, []string, int, error) {
	parts, i, err := p.parseNameParts(s, start)
	if err != nil {
		return "", nil, i, err
	}
	return strings.Join(parts, string(nameSeparator)), parts, i, nil
}

func (p *parser) parseNameParts(s string, start int) ([]string, int, error) {
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

func (p *parser) parseName(s string, start int) (string, int, error) {
	if len(s) == start {
		return "", start, newParseError("unexpected end of string, expected a name", start, s[start:])
	}
	if !unicode.IsLetter(rune(s[start])) {
		return "", start, newParseError("name must start with letter", start, s[start:])
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

func (p *parser) parseOperator(s string, start int) (string, int, error) {
	i := start
	for i < len(s) {
		i += 1
		if v := s[start:i]; p.ops[v] {
			return v, i, nil
		}
	}
	return "", i, newParseError("expected operator", start, s[start:])
}

func (p *parser) parseValue(s string, start int) (string, int, error) {
	if start == len(s) {
		return "", start, nil
	}
	if s[start] == quote {
		return p.parseQuotedValue(s, start)
	}
	return p.parseNormalValue(s, start)
}

func (p *parser) parseNormalValue(s string, start int) (string, int, error) {
	i := spaceOrNonSpace(s, start, false)
	return s[start:i], i, nil
}

func (p *parser) parseQuotedValue(s string, start int) (string, int, error) {
	i := start + 1
	v, i, err := p.parseQuotesEscaped(s, i)
	if err != nil {
		return v, i, err
	}
	if len(s) == i || s[i] != quote {
		return "", start, newParseError("unterminated quoted value", start, s[start:])
	}
	return v, i + 1, nil
}

func (p *parser) parseQuotesEscaped(s string, start int) (string, int, ParseError) {
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

// An Option that can be passed to the NewParser factory method.
type Option interface {
	Apply(parser *parser)
}

type optionSnakeCase struct{}

func (o optionSnakeCase) Apply(parser *parser) {
	parser.snakeCase = true
}

// OptionSnakeCase will instruct the parser to make a best-effort attempt at
// converting field names to snake_case. Cannot be used along with
// OptionCamelCase.
// When an uppercase character is encountered, it will be lower-cased. It will
// be prefixed with an underscore, unless it is the starting character, preceded
// by another uppercase character, or preceded by an underscore.
func OptionSnakeCase() Option {
	return &optionSnakeCase{}
}

type optionCamelCase struct{}

func (o optionCamelCase) Apply(parser *parser) {
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
