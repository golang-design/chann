// Copyright 2022 The golang.design Initiative Authors.
// All rights reserved. Use of this source code is governed
// by a MIT license that can be found in the LICENSE file.
//
// Written by Changkun Ou <changkun.de>

package chann

import "runtime"

// Ranger returns a Sender and a Receiver. The Receiver provides a
// Next method to retrieve values. The Sender provides a Send method
// to send values and a Close method to stop sending values. The Next
// method indicates when the Sender has been closed, and the Send
// method indicates when the Receiver has been freed.
//
// This is a convenient way to exit a goroutine sending values when
// the receiver stops reading them.
func Ranger[T any]() (*Sender[T], *Receiver[T]) {
	c := New[T]()
	d := New[bool]()
	s := &Sender[T]{values: c, done: d}
	r := &Receiver[T]{values: c, done: d}
	runtime.SetFinalizer(r, func(r *Receiver[T]) { r.finalize() })
	return s, r
}

// A sender is used to send values to a Receiver.
type Sender[T any] struct {
	values *Chann[T]
	done   *Chann[bool]
}

// Send sends a value to the receiver. It returns whether any more
// values may be sent; if it returns false the value was not sent.
func (s *Sender[T]) Send(v T) bool {
	select {
	case s.values.In() <- v:
		return true
	case <-s.done.Out():
		return false
	}
}

// Close tells the receiver that no more values will arrive.
// After Close is called, the Sender may no longer be used.
func (s *Sender[T]) Close() { s.values.Close() }

// A Receiver receives values from a Sender.
type Receiver[T any] struct {
	values *Chann[T]
	done   *Chann[bool]
}

// Next returns the next value from the channel. The bool result
// indicates whether the value is valid, or whether the Sender has
// been closed and no more values will be received.
func (r *Receiver[T]) Next() (T, bool) {
	v, ok := <-r.values.Out()
	return v, ok
}

// finalize is a finalizer for the receiver.
func (r *Receiver[T]) finalize() { r.done.Close() }
