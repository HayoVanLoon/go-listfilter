// Copyright 2022 Hayo van Loon. All rights reserved.
// Use of this source code is governed by an Apache
// license that can be found in the LICENSE file.
package listfilter

import (
	"fmt"
	"reflect"
	"testing"
)

func Test_filterParser_Parse(t *testing.T) {
	type fields struct {
		ops map[string]bool
	}
	type args struct {
		s string
	}
	standardFields := fields{ops: NewParser().(*filterParser).ops}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Filter
		wantErr error
	}{
		{
			"simple single condition",
			standardFields,
			args{s: "fo_o1=bar"},
			map[string][]Condition{
				"fo_o1": {NewCondition("fo_o1", []string{"fo_o1"}, "=", "bar")},
			},
			nil,
		},
		{
			"empty filter",
			standardFields,
			args{s: ""},
			map[string][]Condition{},
			nil,
		},
		{
			"complex name",
			standardFields,
			args{s: "foo.bar.bla=vla"},
			map[string][]Condition{
				"foo.bar.bla": {NewCondition("foo.bar.bla", []string{"foo", "bar", "bla"}, "=", "vla")},
			},
			nil,
		},
		{
			"multi-character operator",
			standardFields,
			args{s: "foo!=bar"},
			map[string][]Condition{
				"foo": {NewCondition("foo", []string{"foo"}, "!=", "bar")},
			},
			nil,
		},
		{
			"operator as value",
			standardFields,
			args{s: "foo=="},
			map[string][]Condition{"foo": {NewCondition("foo", []string{"foo"}, "=", "=")}},
			nil,
		},
		{
			"! unknown operator",
			standardFields,
			args{s: "foo*bar"},
			nil,
			NewParseError("expected operator", 3, "*bar"),
		},
		{
			"multiple conditions",
			standardFields,
			args{s: "foo=bar,bla=vla,moo=boo"},
			map[string][]Condition{
				"foo": {NewCondition("foo", []string{"foo"}, "=", "bar")},
				"bla": {NewCondition("bla", []string{"bla"}, "=", "vla")},
				"moo": {NewCondition("moo", []string{"moo"}, "=", "boo")},
			},
			nil,
		},
		{
			"! empty condition",
			standardFields,
			args{s: "foo=bar,,bla=vla"},
			nil,
			NewParseError("name must start with letter", 8, ",bla=vla"),
		},
		{
			"simple single condition",
			standardFields,
			args{s: "foo=bar"},
			map[string][]Condition{
				"foo": {NewCondition("foo", []string{"foo"}, "=", "bar")},
			},
			nil,
		},
		{
			"empty value",
			standardFields,
			args{s: "foo="},
			map[string][]Condition{
				"foo": {NewCondition("foo", []string{"foo"}, "=", "")},
			},
			nil,
		},
		{
			"quoted value",
			standardFields,
			args{s: "foo=\"say \\\"bar\\\"\""},
			map[string][]Condition{
				"foo": {NewCondition("foo", []string{"foo"}, "=", "say \"bar\"")},
			},
			nil,
		},
		{
			"empty quoted value",
			standardFields,
			args{s: "foo=\"\""},
			map[string][]Condition{
				"foo": {NewCondition("foo", []string{"foo"}, "=", "")},
			},
			nil,
		},
		{
			"quoted value with escaped escape character",
			standardFields,
			args{s: "foo=\"say\\\\ \\n \\\"bar\\\"\""},
			map[string][]Condition{
				"foo": {NewCondition("foo", []string{"foo"}, "=", "say\\ \\n \"bar\"")},
			},
			nil,
		},
		{
			"! name only",
			standardFields,
			args{s: "foo"},
			nil,
			NewParseError("expected operator", 3, ""),
		},
		{
			"! name starting with non-letter",
			standardFields,
			args{s: "1foo=bar"},
			nil,
			NewParseError("name must start with letter", 0, "1foo=bar"),
		},
		{
			"! name with empty path",
			standardFields,
			args{s: "foo..bar=bla"},
			nil,
			NewParseError("name must start with letter", 4, ".bar=bla"),
		},
		{
			"! name with invalid part",
			standardFields,
			args{s: "foo.1.bar=bla"},
			nil,
			NewParseError("name must start with letter", 4, "1.bar=bla"),
		},
		{
			"! name only first (error)",
			standardFields,
			args{s: "foo,bar=bla"},
			nil,
			NewParseError("expected operator", 3, ",bar=bla"),
		},
		{
			"! name only second (error)",
			standardFields,
			args{s: "foo=bar,bla"},
			nil,
			NewParseError("expected operator", 11, ""),
		},
		{
			"empty first element",
			standardFields,
			args{s: ",foo=bar"},
			nil,
			NewParseError("name must start with letter", 0, ",foo=bar"),
		},
		{
			"empty last element",
			standardFields,
			args{s: "foo=bar,"},
			nil,
			NewParseError("unexpected end of string, expected a name", 8, ""),
		},
		{
			"empty middle element",
			standardFields,
			args{s: "foo=bar,,bla=vla"},
			nil,
			NewParseError("name must start with letter", 8, ",bla=vla"),
		},
		{
			"! unterminated quoted value",
			standardFields,
			args{s: "foo=\"bar"},
			nil,
			NewParseError("unterminated quoted value", 4, "\"bar"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &filterParser{
				ops: tt.fields.ops,
			}
			got, err := p.Parse(tt.args.s)
			if !reflect.DeepEqual(err, tt.wantErr) {
				if err == nil {
					t.Errorf("\nExpected: %v,\ngot:      %v", tt.wantErr, err)
				} else {
					t.Errorf("\nExpected: %v,\ngot:      %v", tt.wantErr, err)
				}
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkFilterParser_Parse(b *testing.B) {
	p := NewParser()
	type args struct {
		s string
	}
	cases := []struct {
		args args
	}{
		{args: args{s: ""}},
		{args: args{s: "foo=bar"}},
		{args: args{s: "foo=bar,bla=vla"}},
		{args: args{s: "foo.bar=bla"}},
		{args: args{s: "foo.bar=bla,vla=moo"}},
		{args: args{s: "foo=bar,bla=vla,moo=boo"}},
		{args: args{s: "foo=bar,bla=vla,moo=boo,,error"}},
	}

	for i := 0; i < b.N; i += 1 {
		for _, c := range cases {
			_, _ = p.Parse(c.args.s)
		}
	}
}

func TestFilter_GetFirst(t *testing.T) {
	type args struct {
		k string
	}
	tests := []struct {
		name  string
		f     Filter
		args  args
		want  Condition
		want1 bool
	}{
		{
			"simple",
			map[string][]Condition{
				"foo": {NewCondition("foo", []string{"foo"}, "=", "bar")},
			},
			args{"foo"},
			NewCondition("foo", []string{"foo"}, "=", "bar"),
			true,
		},
		{
			"multi-part name",
			map[string][]Condition{
				"foo.bar": {NewCondition("foo.bar", []string{"foo", "bar"}, "=", "bla")},
			},
			args{"foo.bar"},
			NewCondition("foo.bar", []string{"foo", "bar"}, "=", "bla"),
			true,
		},
		{
			"empty",
			nil,
			args{"foo"},
			nil,
			false,
		},
		{
			"unknown",
			map[string][]Condition{
				"foo.bar": {},
			},
			args{"bar"},
			nil,
			false,
		},
		{
			"two conditions",
			map[string][]Condition{
				"foo": {
					NewCondition("foo", []string{"foo"}, "=", "bar"),
					NewCondition("bla", []string{"bla"}, "<", "vla"),
				},
			},
			args{"foo"},
			NewCondition("foo", []string{"foo"}, "=", "bar"),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.f.GetFirst(tt.args.k)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Get() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestFilter_GetLast(t *testing.T) {
	type args struct {
		k string
	}
	tests := []struct {
		name  string
		f     Filter
		args  args
		want  Condition
		want1 bool
	}{
		{
			"simple",
			map[string][]Condition{
				"foo": {NewCondition("foo", []string{"foo"}, "=", "bar")},
			},
			args{"foo"},
			NewCondition("foo", []string{"foo"}, "=", "bar"),
			true,
		},
		{
			"multi-part name",
			map[string][]Condition{
				"foo.bar": {NewCondition("foo.bar", []string{"foo", "bar"}, "=", "bla")},
			},
			args{"foo.bar"},
			NewCondition("foo.bar", []string{"foo", "bar"}, "=", "bla"),
			true,
		},
		{
			"empty",
			nil,
			args{"foo"},
			nil,
			false,
		},
		{
			"unknown",
			map[string][]Condition{
				"foo.bar": {},
			},
			args{"bar"},
			nil,
			false,
		},
		{
			"two conditions",
			map[string][]Condition{
				"foo": {
					NewCondition("foo", []string{"foo"}, "=", "bar"),
					NewCondition("foo", []string{"foo"}, "<", "bar"),
				},
			},
			args{"foo"},
			NewCondition("foo", []string{"foo"}, "<", "bar"),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.f.GetLast(tt.args.k)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Get() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_condition_BoolValue(t *testing.T) {
	type fields struct {
		key         string
		keyParts    []string
		op          string
		stringValue string
	}
	tests := []struct {
		name    string
		fields  fields
		want    bool
		wantErr error
	}{
		{
			"simple true",
			fields{"foo", []string{"foo"}, "=", "true"},
			true,
			nil,
		},
		{
			"simple false",
			fields{"foo", []string{"foo"}, "=", "false"},
			false,
			nil,
		},
		{
			"case-insensitive true",
			fields{"foo", []string{"foo"}, "=", "tRue"},
			true,
			nil,
		},
		{
			"case-insensitive false",
			fields{"foo", []string{"foo"}, "=", "faLse"},
			false,
			nil,
		},
		{
			"invalid input",
			fields{"foo", []string{"foo"}, "=", "42"},
			false,
			fmt.Errorf("42 is not a valid boolean"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := condition{
				key:         tt.fields.key,
				keyParts:    tt.fields.keyParts,
				op:          tt.fields.op,
				stringValue: tt.fields.stringValue,
			}
			got, err := c.BoolValue()
			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("BoolValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("BoolValue() got = %v, want %v", got, tt.want)
			}
		})
	}
}
