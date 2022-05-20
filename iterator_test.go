// Copyright 2022 Hayo van Loon. All rights reserved.
// Use of this source code is governed by an Apache
// license that can be found in the LICENSE file.

package listfilter

import (
	"fmt"
	"reflect"
	"testing"
)

func readAll[T any](it Iterator[T]) ([]T, error) {
	if it == nil {
		return nil, nil
	}
	var xs []T
	for {
		x, err := it.Next()
		if err != nil && err != Done {
			return xs, err
		}
		if !reflect.ValueOf(x).IsZero() {
			xs = append(xs, x)
		}
		if err == Done {
			break
		}
	}
	return xs, nil
}

func count[T comparable](xs []T) map[T]int {
	m := make(map[T]int)
	for _, v := range xs {
		m[v] += 1
	}
	return m
}

func TestForMap(t *testing.T) {
	type args struct {
		m map[string]int
	}
	tests := []struct {
		name string
		args args
		want map[int]int
	}{
		{
			"simple",
			args{map[string]int{"a": 3, "b": 1, "c": 2}},
			map[int]int{3: 1, 1: 1, 2: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it := ForMap(tt.args.m)
			xs, _ := readAll(it)
			got := count(xs)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Expected %v\n, got      %v", tt.want, got)
			}
			_, err := it.Next()
			if err != Done {
				t.Errorf("expected Done, got %v", err)
			}
		})
	}
}

func TestForFunc(t *testing.T) {
	type args[T any] struct {
		f func(chan<- T, chan<- error)
	}
	tests := []struct {
		name    string
		args    args[any]
		want    []int
		wantErr bool
	}{
		{
			"simple",
			args[any]{func(ch chan<- any, errCh chan<- error) {
				defer close(ch)
				defer close(errCh)
				ch <- 3
				ch <- 1
				ch <- 2
			}},
			[]int{3, 1, 2},
			false,
		},
		{
			"error",
			args[any]{func(ch chan<- any, errCh chan<- error) {
				defer close(ch)
				defer close(errCh)
				ch <- 3
				ch <- 1
				errCh <- fmt.Errorf("oh noes")
			}},
			[]int{3, 1},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it := ForFunc(tt.args.f)
			for i := 0; ; i += 1 {
				x, err := it.Next()
				if err != nil && err != Done {
					if got := err != nil; got != tt.wantErr {
						t.Errorf("error %v != %v", tt.wantErr, got)
					}
					return
				}
				if i == len(tt.want) {
					if x != nil {
						t.Errorf("unexpected item %v", x)
					}
					return
				}
				if x != tt.want[i] {
					t.Errorf("Expected %v @ %d\n, got      %v", tt.want[i], i, x)
				}
				if err == Done {
					break
				}
			}
			_, err := it.Next()
			if err != Done {
				t.Errorf("expected Done, got %v", err)
			}
		})
	}
}

func TestForSlice(t *testing.T) {
	type args[T any] struct {
		xs []any
	}
	type Foo struct {
		Value int
	}
	tests := []struct {
		name    string
		args    args[any]
		want    []any
		wantErr bool
	}{
		{
			"simple",
			args[any]{[]any{3, 1, 2}},
			[]any{3, 1, 2},
			false,
		},
		{
			"simple structs",
			args[any]{[]any{Foo{3}, Foo{1}, Foo{2}}},
			[]any{Foo{3}, Foo{1}, Foo{2}},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it := ForSlice(tt.args.xs)
			for i := 0; ; i += 1 {
				x, err := it.Next()
				if err != nil && err != Done {
					if got := err != nil; got != tt.wantErr {
						t.Errorf("error %v != %v", tt.wantErr, got)
					}
					return
				}
				if i == len(tt.want) {
					if x != nil {
						t.Errorf("unexpected item %v", x)
					}
					return
				}
				if x != tt.want[i] {
					t.Errorf("Expected %v @ %d\n, got      %v", tt.want[i], i, x)
				}
				if err == Done {
					break
				}
			}
			_, err := it.Next()
			if err != Done {
				t.Errorf("expected Done, got %v", err)
			}
		})
	}
}
