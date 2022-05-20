// Copyright 2022 Hayo van Loon. All rights reserved.
// Use of this source code is governed by an Apache
// license that can be found in the LICENSE file.

package listfilter

import "errors"

// TODO(hvl): move to separate package

type Iterator[T any] interface {
	Next() (T, error)
}

var Done = errors.New("done")

type iterator[T any] struct {
	next chan T
	errs chan error
	err  error
}

func Empty[T any]() Iterator[T] {
	return &iterator[T]{}
}

func ForMap[K comparable, V any](m map[K]V) Iterator[V] {
	if len(m) == 0 {
		return Empty[V]()
	}
	ch := make(chan V, 1)
	errCh := make(chan error)
	go func() {
		defer close(ch)
		defer close(errCh)
		for _, v := range m {
			ch <- v
		}
	}()
	return &iterator[V]{next: ch, errs: errCh}
}

func ForSlice[T any](xs []T) Iterator[T] {
	if len(xs) == 0 {
		return Empty[T]()
	}
	ch := make(chan T, 1)
	errCh := make(chan error)
	go func() {
		defer close(ch)
		defer close(errCh)
		for _, v := range xs {
			ch <- v
		}
	}()
	return &iterator[T]{next: ch, errs: errCh}
}

// ForFunc creates an iterator from a function that pushes to a channel (and/or
// an error channel). The function should close both channels before returning.
func ForFunc[T any](f func(chan<- T, chan<- error)) Iterator[T] {
	ch := make(chan T, 1)
	errCh := make(chan error, 1)
	go f(ch, errCh)
	return &iterator[T]{
		next: ch,
		errs: errCh,
	}
}

// Flatten takes an iterator that produces slices and returns an iterator that
// returns the slice' elements.
func Flatten[T []U, U any](it Iterator[T]) Iterator[U] {
	ch := make(chan U, 1)
	errCh := make(chan error, 1)
	go func() {
		defer close(ch)
		defer close(errCh)
		for {
			us, err := it.Next()
			if err != nil && err != Done {
				errCh <- err
				break
			}
			for _, u := range us {
				ch <- u
			}
			if err == Done {
				break
			}
		}
	}()
	return &iterator[U]{next: ch, errs: errCh}
}

// Next returns the next value in the iterator. From the moment the last value
// is returned and onward, the error will always be Done. If an error has been
// returned, all subsequent calls to Next will (only) return this error.
//
// The following is a typical pattern for consuming the Iterator.
//
//   for {
//       x, err := it.Next()
//       if err != nil && err != Done {
//           // handle error
//       }
//
//       // do something with x
//
//       if err == Done {
//           break
//       }
//   }
func (it *iterator[T]) Next() (T, error) {
	if it.err != nil {
		return *new(T), it.err
	}
	n, ok := <-it.next
	if !ok {
		if err, ok := <-it.errs; ok {
			it.err = err
		} else {
			it.err = Done
		}
	}
	return n, it.err
}
