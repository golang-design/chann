// Copyright 2022 The golang.design Initiative Authors.
// All rights reserved. Use of this source code is governed
// by a MIT license that can be found in the LICENSE file.
//
// Written by Changkun Ou <changkun.de>

package chann

import (
	"math/rand"
	"sync"
)

// Fanin provides a generic fan-in functionality for variadic channels.
func Fanin[T any](chans ...*Chann[T]) *Chann[T] {
	buf := 0
	for _, ch := range chans {
		if ch.Len() > buf {
			buf = ch.Len()
		}
	}
	out := New[T](Cap(buf))
	wg := sync.WaitGroup{}
	wg.Add(len(chans))
	for _, ch := range chans {
		go func(ch *Chann[T]) {
			for v := range ch.Out() {
				out.In() <- v
			}
			wg.Done()
		}(ch)
	}
	go func() {
		wg.Wait()
		out.Close()
	}()
	return out
}

// Fanout provides a generic fan-out functionality for variadic channels.
func Fanout[T any](randomizer func(max int) int, in *Chann[T], outs ...*Chann[T]) {
	l := len(outs)
	for v := range in.Out() {
		i := randomizer(l)
		if i < 0 || i > l {
			i = rand.Intn(l)
		}
		go func(v T) { outs[i].In() <- v }(v)
	}
}

// LB load balances the given input channels to the output channels.
func LB[T any](randomizer func(max int) int, ins []*Chann[T], outs []*Chann[T]) {
	Fanout(randomizer, Fanin(ins...), outs...)
}
