// Copyright 2021 The golang.design Initiative Authors.
// All rights reserved. Use of this source code is governed
// by a MIT license that can be found in the LICENSE file.
//
// Written by Changkun Ou <changkun.de>

package chann

import "sync/atomic"

type Opt func(*config)

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

type Chann[T any] struct {
	in, out chan T
	cfg     *config
}

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
	ch := &Chann[T]{cfg: cfg}
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
		ready := make(chan struct{})
		go func() {
			q := make([]T, 0, 1<<10)
			ready <- struct{}{}
			for {
				e, ok := <-ch.in
				if !ok {
					close(ch.out)
					return
				}
				atomic.AddInt64(&ch.cfg.len, 1)
				q = append(q, e)

				for len(q) > 0 {
					select {
					case ch.out <- q[0]:
						atomic.AddInt64(&ch.cfg.len, -1)
						q = q[1:]
					case e, ok := <-ch.in:
						if ok {
							atomic.AddInt64(&ch.cfg.len, 1)
							q = append(q, e)
							break
						}
						for _, e := range q {
							atomic.AddInt64(&ch.cfg.len, -1)
							ch.out <- e
						}
						close(ch.out)
						return
					}
				}
				if cap(q) < 1<<5 {
					q = make([]T, 0, 1<<10)
				}
			}
		}()
		<-ready
	}
	return ch
}
func (ch *Chann[T]) In() chan<- T  { return ch.in }
func (ch *Chann[T]) Out() <-chan T { return ch.out }
func (ch *Chann[T]) Close()        { close(ch.in) }

func (ch *Chann[T]) ApproxLen() int {
	switch ch.cfg.typ {
	case buffered, unbuffered:
		return len(ch.in)
	default:
		return int(atomic.LoadInt64(&ch.cfg.len)) + len(ch.in) + len(ch.out)
	}
}

func (ch *Chann[T]) Cap() int {
	switch ch.cfg.typ {
	case buffered, unbuffered:
		return cap(ch.in)
	default:
		return int(atomic.LoadInt64(&ch.cfg.cap)) + cap(ch.in) + cap(ch.out)
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
