// Copyright 2022 Hayo van Loon. All rights reserved.
// Use of this source code is governed by an Apache
// license that can be found in the LICENSE file.

// Package listfilter Lorem ipsum dolor sic amet.
package listfilter

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
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
}

type condition struct {
	key         string
	keyParts    []string
	op          string
	stringValue string
}

// NewCondition creates a new Condition from the specified parameters.
func NewCondition(key string, keyParts []string, op, stringValue string) Condition {
	return condition{key, keyParts, op, stringValue}
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
func NewParseError(message string, position int, text string) ParseError {
	return &parseError{message, position, text}
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

type Filter map[string][]Condition

// GetFirst retrieves the first condition for a given key.
func (f Filter) GetFirst(k string) (Condition, bool) {
	cs := f[k]
	// empty lists can exist, go beyond nil check
	if len(cs) > 0 {
		return cs[0], true
	}
	return nil, false
}

// GetLast retrieves the last condition for a given key.
func (f Filter) GetLast(k string) (Condition, bool) {
	cs := f[k]
	// empty lists can exist, go beyond nil check
	if l := len(cs); l > 0 {
		return cs[l-1], true
	}
	return nil, false
}

type filterParser struct {
	ops map[string]bool
}

// NewParser creates a new FilterParser.
func NewParser() FilterParser {
	return &filterParser{ops: map[string]bool{"=": true, "!=": true}}
}

// Parse parses a filter string into a Filter.
//
// Examples of filter strings:
//
//   "foo=bar"
//   "foo.bar=bla"
//   "foo=bar,bla=vla"
//
// The filter string should adher to the following grammar:
//
//   Filter ->        <nil> | Conditions
//   Conditions ->    Condition | Condition Separator Conditions
//   Separator ->     ,
//   Condition ->     FullName Operator Value
//   FullName ->      NameParts
//   NameParts ->     Name | Name NameSeparator NameParts
//   NameSeparator -> .
//   Name ->          regex([a-zA-Z][a-zA-Z0-9_]*)
//   Operator ->      regex([^a-zA-Z0-9_].*)
//   Value ->         NormalValue | QuotedValue
//   NormalValue ->   regex([^separator]*)
//   QuotedValue ->   " Escaped "
//   Escaped ->       <nil> | NormalChar Escaped | EscapedChar Escaped
//   EscapedChar ->   \\ | \"
//   NormalChar ->    <not eChar>
//
// An empty string is considered a valid input and will result in an empty
// Filter.
func (p *filterParser) Parse(s string) (Filter, ParseError) {
	if len(s) == 0 {
		return make(map[string][]Condition, 0), nil
	}
	filter, _, err := p.parseConditions(s, 0)
	if err != nil {
		return nil, err
	}
	return filter, nil
}

const (
	separator       = ','
	nameSeparator   = '.'
	escapeCharacter = '\\'
	quote           = '"'
)

func (p *filterParser) parseConditions(s string, start int) (map[string][]Condition, int, ParseError) {
	cond, i, err := p.parseCondition(s, start)
	if err != nil {
		return nil, i, err
	}
	m := map[string][]Condition{cond.key: {cond}}
	for i < len(s) && s[i] == separator {
		i += 1
		cond, i, err = p.parseCondition(s, i)
		if err != nil {
			return nil, i, err
		}
		xs := m[cond.key]
		m[cond.key] = append(xs, cond)
	}
	return m, start, nil
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
	return condition{key, keyParts, op, value}, i, nil
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
		return parseQuotedValue(s, start)
	}
	return parseNormalValue(s, start)
}

func parseNormalValue(s string, start int) (string, int, ParseError) {
	i := strings.IndexByte(s[start:], separator)
	if i == -1 {
		return s[start:], len(s), nil
	}
	return s[start : start+i], start + i, nil
}

func parseQuotedValue(s string, start int) (string, int, ParseError) {
	i := start + 1
	v, i, err := parseQuotesEscaped(s, i)
	if err != nil {
		return v, i, err
	}
	if len(s) <= i || s[i] != quote {
		return "", start, NewParseError("unterminated quoted value", start, s[start:])
	}
	return v, i, err
}

func parseQuotesEscaped(s string, start int) (string, int, ParseError) {
	sb := strings.Builder{}
	i := start
	escape := false
	for ; i < len(s); i += 1 {
		if escape {
			switch s[i] {
			case quote, escapeCharacter:
			default:
				// no special meaning, add escape character retroactively
				sb.WriteRune(escapeCharacter)
			}
			escape = false
		} else if s[i] == quote {
			break
		} else if s[i] == escapeCharacter {
			escape = true
			continue
		}
		sb.WriteByte(s[i])
	}
	return sb.String(), i, nil
}
