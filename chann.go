// Copyright 2021 The golang.design Initiative Authors.
// All rights reserved. Use of this source code is governed
// by a MIT license that can be found in the LICENSE file.
//
// Written by Changkun Ou <changkun.de>

// Package chann provides a unified channel package.
//
// The package is compatible with existing buffered and unbuffered
// channels. For example, in Go, to create a buffered or unbuffered
// channel, one uses built-in function `make` to create a channel:
//
//	ch := make(chan int)     // unbuffered channel
//	ch := make(chan int, 42) // or buffered channel
//
// However, all these channels have a finite capacity for caching, and
// it is impossible to create a channel with unlimited capacity, namely,
// an unbounded channel.
//
// This package provides the ability to create all possible types of
// channels. To create an unbuffered or a buffered channel:
//
//	ch := chann.New[int](chann.Cap(0))  // unbuffered channel
//	ch := chann.New[int](chann.Cap(42)) // or buffered channel
//
// More importantly, when the capacity of the channel is unspecified,
// or provided as negative values, the created channel is an unbounded
// channel:
//
//	ch := chann.New[int]()               // unbounded channel
//	ch := chann.New[int](chann.Cap(-42)) // or unbounded channel
//
// Furthermore, all channels provides methods to send (In()),
// receive (Out()), and close (Close()).
//
// Note that to close a channel, must use Close() method instead of the
// language built-in method.
// Two additional methods: Len and Cap returns the current status of the
// channel: an approximation of the current length of the channel, as
// well as the current capacity of the channel.
//
// See https://golang.design/research/ultimate-channel to understand
// the motivation of providing this package and the possible use cases
// with this package.
package chann // import "golang.design/x/chann"

import (
	"runtime"
	"sync/atomic"
)

// Opt represents an option to configure the created channel. The current possible
// option is Cap.
type Opt func(*config)

// Cap is the option to configure the capacity of a creating buffer.
// if the provided number is 0, Cap configures the creating buffer to a
// unbuffered channel; if the provided number is a positive integer, then
// Cap configures the creating buffer to a buffered channel with the given
// number of capacity  for caching. If n is a negative integer, then it
// configures the creating channel to become an unbounded channel.
func Cap(n int) Opt {
	return func(s *config) {
		switch {
		case n == 0:
			s.cap = int64(0)
			s.typ = unbuffered
		case n > 0:
			s.cap = int64(n)
			s.typ = buffered
		default:
			s.cap = int64(-1)
			s.typ = unbounded
		}
	}
}

// Chann is a generic channel abstraction that can be either buffered,
// unbuffered, or unbounded. To create a new channel, use New to allocate
// one, and use Cap to configure the capacity of the channel.
type Chann[T any] struct {
	in, out chan T
	close   chan struct{}
	cfg     *config
}

// New returns a Chann that may be a buffered, an unbuffered or an
// unbounded channel. To configure the type of the channel, use Cap.
//
// By default, or without specification, the function returns an unbounded
// channel with unlimited capacity.
//
//	ch := chann.New[float64]()
//	// or
//	ch := chann.New[float64](chann.Cap(-1))
//
// If the chann.Cap specified a non-negative integer, the returned channel
// is either unbuffered (0) or buffered (positive).
//
// An unbounded channel is not a buffered channel with infinite capacity,
// and they have different memory model semantics in terms of receiving
// a value: The recipient of a buffered channel is immediately available
// after a send is complete. However, the recipient of an unbounded channel
// may be available within a bounded time frame after a send is complete.
//
// Note that although the input arguments are specified as variadic parameter
// list, however, the function panics if there is more than one option is
// provided.
func New[T any](opts ...Opt) *Chann[T] {
	cfg := &config{
		cap: -1, len: 0,
		typ: unbounded,
	}

	if len(opts) > 1 {
		panic("chann: too many arguments")
	}
	for _, o := range opts {
		o(cfg)
	}
	ch := &Chann[T]{cfg: cfg, close: make(chan struct{})}
	switch ch.cfg.typ {
	case unbuffered:
		ch.in = make(chan T)
		ch.out = ch.in
	case buffered:
		ch.in = make(chan T, ch.cfg.cap)
		ch.out = ch.in
	case unbounded:
		ch.in = make(chan T, 16)
		ch.out = make(chan T, 16)

		// released is closed once ch becomes unreachable so that the
		// processing goroutine can terminate even if there is no
		// receiver to drain the backlog after Close. The processing
		// goroutine must NOT close over ch (doing so would keep ch
		// reachable forever and the cleanup would never run); it only
		// captures the extracted channels and config below.
		released := make(chan struct{})
		runtime.AddCleanup(ch, func(r chan struct{}) { close(r) }, released)

		go unboundedProcessing(ch.in, ch.out, ch.close, released, ch.cfg)
	}
	return ch
}

// In returns the send channel of the given Chann, which can be used to
// send values to the channel. If one closes the channel using close(),
// it will result in a runtime panic. Instead, use Close() method.
func (ch *Chann[T]) In() chan<- T { return ch.in }

// Out returns the receive channel of the given Chann, which can be used
// to receive values from the channel.
func (ch *Chann[T]) Out() <-chan T { return ch.out }

// Close closes the channel gracefully.
func (ch *Chann[T]) Close() {
	switch ch.cfg.typ {
	case buffered, unbuffered:
		close(ch.in)
		close(ch.close)
	default:
		ch.close <- struct{}{}
	}
}

// unboundedProcessing is a processing loop that implements unbounded
// channel semantics.
//
// It is a free function rather than a method on purpose: it must not
// capture the owning *Chann, otherwise the running goroutine would keep
// the Chann reachable forever and the cleanup registered in New would
// never fire. It only operates on the extracted channels and config.
func unboundedProcessing[T any](in, out chan T, closed, released chan struct{}, cfg *config) {
	var nilT T

	q := make([]T, 0, 1<<10)
	for {
		select {
		case e, ok := <-in:
			if !ok {
				panic("chann: send-only channel ch.In() closed unexpectedly")
			}
			atomic.AddInt64(&cfg.len, 1)
			q = append(q, e)
		case <-closed:
			unboundedTerminate(in, out, closed, released, q)
			return
		}

		for len(q) > 0 {
			select {
			case out <- q[0]:
				atomic.AddInt64(&cfg.len, -1)
				q[0] = nilT
				q = q[1:]
			case e, ok := <-in:
				if !ok {
					panic("chann: send-only channel ch.In() closed unexpectedly")
				}
				atomic.AddInt64(&cfg.len, 1)
				q = append(q, e)
			case <-closed:
				unboundedTerminate(in, out, closed, released, q)
				return
			}
		}
		if cap(q) < 1<<5 {
			q = make([]T, 0, 1<<10)
		}
	}
}

// unboundedTerminate terminates the unbounded channel's processing loop
// and makes sure all unprocessed elements are consumed if there is a
// pending receiver.
//
// After Close, the backlog is delivered to out with a blocking send so
// that no element is dropped while a receiver is draining. If there is
// no receiver, the send would block forever; the released signal (closed
// by the cleanup once the owning Chann is garbage collected) lets the
// goroutine terminate instead of leaking.
func unboundedTerminate[T any](in, out chan T, closed, released chan struct{}, q []T) {
	var nilT T

	close(in)
	for e := range in {
		q = append(q, e)
	}
	for len(q) > 0 {
		select {
		case out <- q[0]:
			q[0] = nilT // de-reference earlier to help GC
			q = q[1:]
		case <-released:
			goto final
		}
	}

final:
	close(out)
	close(closed)
}

// isClose reports the close status of a channel.
func (ch *Chann[T]) isClosed() bool {
	select {
	case <-ch.close:
		return true
	default:
		return false
	}
}

// Len returns an approximation of the length of the channel.
//
// Note that in a concurrent scenario, the returned length of a channel
// may never be accurate. Hence the result should only be treated as an
// approximation.
func (ch *Chann[T]) Len() int {
	switch ch.cfg.typ {
	case buffered, unbuffered:
		return len(ch.in)
	default:
		return int(atomic.LoadInt64(&ch.cfg.len)) + len(ch.in) + len(ch.out)
	}
}

// Cap returns the capacity of the channel. For an unbounded channel it
// returns -1, which is consistent with how a negative Cap option creates
// an unbounded channel.
func (ch *Chann[T]) Cap() int {
	switch ch.cfg.typ {
	case buffered, unbuffered:
		return cap(ch.in)
	default:
		return -1
	}
}

type chanType int

const (
	unbuffered chanType = iota
	buffered
	unbounded
)

type config struct {
	typ      chanType
	len, cap int64
}
