// Copyright 2022 Hayo van Loon. All rights reserved.
// Use of this source code is governed by an Apache
// license that can be found in the LICENSE file.
package listfilter

import (
	"reflect"
	"testing"
)

func Test_filter_Size(t *testing.T) {
	type fields struct {
		conds map[string][]Condition
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &filter{
				conds: tt.fields.conds,
			}
			if got := f.Size(); got != tt.want {
				t.Errorf("Size() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_filter_Get(t *testing.T) {
	type fields struct {
		conds map[string][]Condition
	}
	type args struct {
		k string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   Condition
		want1  bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &filter{
				conds: tt.fields.conds,
			}
			got, got1 := f.Get(tt.args.k)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Get() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

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
			args{s: "foo=bar"},
			&filter{conds: map[string][]Condition{
				"foo": {NewCondition("foo", []string{"foo"}, "=", "bar")}},
			},
			nil,
		},
		{
			"empty filter",
			standardFields,
			args{s: ""},
			&filter{conds: map[string][]Condition{}},
			nil,
		},
		{
			"complex name",
			standardFields,
			args{s: "foo.bar.bla=vla"},
			&filter{conds: map[string][]Condition{
				"foo.bar.bla": {NewCondition("foo.bar.bla", []string{"foo", "bar", "bla"}, "=", "vla")}},
			},
			nil,
		},
		{
			"multi-character operator",
			standardFields,
			args{s: "foo!=bar"},
			&filter{conds: map[string][]Condition{
				"foo": {NewCondition("foo", []string{"foo"}, "!=", "bar")}},
			},
			nil,
		},
		{
			"operator as value",
			standardFields,
			args{s: "foo=="},
			&filter{conds: map[string][]Condition{"foo": {NewCondition("foo", []string{"foo"}, "=", "=")}}},
			nil,
		},
		{
			"! unknown operator",
			standardFields,
			args{s: "foo*bar"},
			nil,
			&ParseError{"expected operator", 3, "*bar"},
		},
		{
			"multiple conditions",
			standardFields,
			args{s: "foo=bar,bla=vla,moo=boo"},
			&filter{conds: map[string][]Condition{
				"foo": {NewCondition("foo", []string{"foo"}, "=", "bar")},
				"bla": {NewCondition("bla", []string{"bla"}, "=", "vla")},
				"moo": {NewCondition("moo", []string{"moo"}, "=", "boo")},
			}},
			nil,
		},
		{
			"! empty condition",
			standardFields,
			args{s: "foo=bar,,bla=vla"},
			nil,
			&ParseError{"expected a letter", 8, ""},
		},
		{
			"simple single condition",
			standardFields,
			args{s: "foo=bar"},
			&filter{conds: map[string][]Condition{
				"foo": {NewCondition("foo", []string{"foo"}, "=", "bar")}},
			},
			nil,
		},
		{
			"empty value",
			standardFields,
			args{s: "foo="},
			&filter{conds: map[string][]Condition{
				"foo": {NewCondition("foo", []string{"foo"}, "=", "")}},
			},
			nil,
		},
		{
			"quoted value",
			standardFields,
			args{s: "foo=\"say \\\"bar\\\"\""},
			&filter{conds: map[string][]Condition{
				"foo": {NewCondition("foo", []string{"foo"}, "=", "say \"bar\"")}},
			},
			nil,
		},
		{
			"quoted value with escaped escape character",
			standardFields,
			args{s: "foo=\"say\\\\ \\n \\\"bar\\\"\""},
			&filter{conds: map[string][]Condition{
				"foo": {NewCondition("foo", []string{"foo"}, "=", "say\\ \\n \"bar\"")}},
			},
			nil,
		},
		{
			"! name only",
			standardFields,
			args{s: "foo"},
			nil,
			&ParseError{"expected operator", 3, ""},
		},
		{
			"! name only first (error)",
			standardFields,
			args{s: "foo,bar=bla"},
			nil,
			&ParseError{"expected operator", 3, ",bar=bla"},
		},
		{
			"! name only second (error)",
			standardFields,
			args{s: "foo=bar,bla"},
			nil,
			&ParseError{"expected operator", 11, ""},
		},
		{
			"! unterminated quoted value",
			standardFields,
			args{s: "foo=\"bar"},
			nil,
			&ParseError{"unterminated quoted value", 4, "\"bar"},
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
					t.Errorf("Expected %v, got %v", tt.wantErr, err)
				} else {
					t.Errorf("Expected = %v, wantErr %v", err, tt.wantErr)
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
	}{}

	for i := 0; i < b.N; i += 1 {
		for _, c := range cases {
			_, _ = p.Parse(c.args.s)
		}
	}
}
