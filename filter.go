// Copyright 2022 Hayo van Loon. All rights reserved.
// Use of this source code is governed by an Apache
// license that can be found in the LICENSE file.

// Package listfilter Lorem ipsum dolor sic amet.
package listfilter

import (
	"fmt"
	"strings"
	"unicode"
)

type FilterParser interface {
	// Parse a string into a Filter or return a ParseError.
	//
	// The string should adher to the following grammar:
	//
	// Filter -> [Conditions]
	// Conditions -> Condition | Condition separator Conditions
	// separator -> ","
	// Condition -> fullName operator value
	// fullName -> nameParts
	// nameParts -> name | name nameSeparator nameParts
	// nameSeparator -> "."
	// name -> [a-zA-Z][a-zA-Z0-9_]*
	// operator -> [^a-zA-Z0-9_]+
	// value -> "([^"]|(\\"))*" | ([^"=][^=]*)?
	Parse(s string) (Filter, error)
}

type Filter interface {
	// Get conditions by key.
	Get(k string) ([]Condition, bool)
	// Size returns the number of conditions in the filter.
	Size() int
}

type Condition struct {
	Key         string
	KeyParts    []string
	Op          string
	StringValue string
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
	separator     = ','
	nameSeparator = '.'
)

func (p *filterParser) parseConditions(s string, start int) (map[string][]Condition, int, error) {
	cond, i, err := p.parseCondition(s, start)
	if err != nil {
		return nil, i, err
	}
	m := map[string][]Condition{cond.Key: {cond}}
	for i < len(s) && s[i] == separator {
		i += 1
		cond, i, err = p.parseCondition(s, i)
		if err != nil {
			return nil, i, err
		}
		xs := m[cond.Key]
		if xs == nil {
			m[cond.Key] = []Condition{cond}
		} else {
			m[cond.Key] = append(xs, cond)
		}
	}
	return m, start, nil
}

func (p *filterParser) parseCondition(s string, start int) (Condition, int, error) {
	keyParts, i, err := p.parseNameParts(s, start)
	if err != nil {
		return Condition{}, i, err
	}
	op, i, err := p.parseOperator(s, i)
	if err != nil {
		return Condition{}, i, err
	}
	value, i, err := p.parseValue(s, i)
	if err != nil {
		return Condition{}, i, err
	}
	return Condition{strings.Join(keyParts, string(nameSeparator)), keyParts, op, value}, i, nil
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
	i := start
	// TODO(hvl): optimise
	for ; i < len(s) && s[i] != separator; i += 1 {
	}
	v := s[start:i]
	for k := range p.ops {
		if j := strings.Index(v, k); j >= 0 {
			return "", j, &ParseError{"operator found in value", start + j + len(k), v[:j+len(k)]}
		}
	}
	return v, i, nil
}
