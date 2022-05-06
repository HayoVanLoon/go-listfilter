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

type FilterParser interface {
	// Parse a string into a Filter or return a ParseError.
	//
	// The string should adher to the following grammar:
	//
	// Filter -> 		<nil> | Conditions
	// conditions ->	condition | Condition separator Conditions
	// separator ->  	,
	// Condition ->  	fullName operator value
	// fullName ->		nameParts
	// nameParts ->  	name | name nameSeparator nameParts
	// nameSeparator -> .
	// name -> 			regex([a-zA-Z][a-zA-Z0-9_]*)
	// operator -> 		regex([^a-zA-Z0-9_].*)
	// value -> 		normalValue | quotedValue
	// normalValue ->	regex([^separator]*)
	// quotedValue ->	" escaped "
	// escaped ->		<nil> | nChar escaped | eChar escaped
	// eChar ->			\\ | \"
	// nChar ->			<not eChar>
	Parse(s string) (Filter, error)
}

type Filter interface {
	// Get conditions by key.
	Get(k string) ([]Condition, bool)
	// Size returns the number of conditions in the filter.
	Size() int
}

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

func NewCondition(key string, keyParts []string, op, stringValue string) Condition {
	return condition{key, keyParts, op, stringValue}
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
	return strconv.Atoi(c.stringValue)
}

func (c condition) BoolValue() (bool, error) {
	switch strings.ToLower(c.stringValue) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	return false, fmt.Errorf("not a valid boolean: %s", c.stringValue)
}

func (c condition) FloatValue() (float64, error) {
	return strconv.ParseFloat(c.stringValue, 64)
}

type ParseError struct {
	Message  string
	Position int
	Text     string
}

func (pe *ParseError) Error() string {
	return fmt.Sprintf("%s @ %d (%s)", pe.Message, pe.Position, pe.Text)
}

type filter struct {
	conds map[string][]Condition
}

func (f *filter) Get(k string) ([]Condition, bool) {
	cs, ok := f.conds[k]
	return cs, ok
}

func (f *filter) Size() int {
	return len(f.conds)
}

type filterParser struct {
	ops map[string]bool
}

// NewParser creates a new parser.
func NewParser() FilterParser {
	return &filterParser{ops: map[string]bool{"=": true, "!=": true}}
}

func (p *filterParser) Parse(s string) (Filter, error) {
	if len(s) == 0 {
		return &filter{conds: make(map[string][]Condition, 0)}, nil
	}
	conds, _, err := p.parseConditions(s, 0)
	if err != nil {
		return nil, err
	}
	return &filter{conds: conds}, nil
}

const (
	separator       = ','
	nameSeparator   = '.'
	escapeCharacter = '\\'
	quote           = '"'
)

func (p *filterParser) parseConditions(s string, start int) (map[string][]Condition, int, error) {
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
		if xs == nil {
			m[cond.key] = []Condition{cond}
		} else {
			m[cond.key] = append(xs, cond)
		}
	}
	return m, start, nil
}

func (p *filterParser) parseCondition(s string, start int) (condition, int, error) {
	keyParts, i, err := p.parseNameParts(s, start)
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
	return condition{strings.Join(keyParts, string(nameSeparator)), keyParts, op, value}, i, nil
}

func (p *filterParser) parseFullName(s string, start int) (string, int, error) {
	parts, i, err := p.parseNameParts(s, start)
	if err != nil {
		return "", i, err
	}
	return strings.Join(parts, "."), i, nil
}

func (p *filterParser) parseNameParts(s string, start int) ([]string, int, error) {
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

func (p *filterParser) parseName(s string, start int) (string, int, error) {
	if len(s) == start {
		return "", start, &ParseError{"expected a name", start, s[start:start]}
	}
	if !unicode.IsLetter(rune(s[start])) {
		return "", start, &ParseError{"expected a letter", start, s[start:start]}
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

func (p *filterParser) parseOperator(s string, start int) (string, int, error) {
	i := start
	for i < len(s) {
		i += 1
		if v := s[start:i]; p.ops[v] {
			return v, i, nil
		}
	}
	return "", i, &ParseError{"expected operator", start, s[start:i]}
}

func (p *filterParser) parseValue(s string, start int) (string, int, error) {
	if start == len(s) {
		return "", start, nil
	}
	if s[start] == quote {
		return parseQuotedValue(s, start)
	}
	return parseNormalValue(s, start)
}

func parseNormalValue(s string, start int) (string, int, error) {
	// normalValue ->	regex([^separator]*)
	i := strings.IndexByte(s[start:], separator)
	if i == -1 {
		return s[start:], len(s), nil
	}
	return s[start : start+i], start + i, nil
}

func parseQuotedValue(s string, start int) (string, int, error) {
	i := start + 1
	v, i, err := parseQuotesEscaped(s, i)
	if err != nil {
		return v, i, err
	}
	if len(s) <= i || s[i] != quote {
		return "", start, &ParseError{"unterminated quoted value", start, s[start:i]}
	}
	return v, i, err
}

func parseQuotesEscaped(s string, start int) (string, int, error) {
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
