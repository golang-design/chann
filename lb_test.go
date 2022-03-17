// Copyright 2022 The golang.design Initiative Authors.
// All rights reserved. Use of this source code is governed
// by a MIT license that can be found in the LICENSE file.
//
// Written by Changkun Ou <changkun.de>

package chann_test

import (
	"math/rand"
	"testing"

	"golang.design/x/chann"
)

func getInputChan() *chann.Chann[int] {
	input := chann.New[int](chann.Cap(20))
	numbers := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	go func() {
		for num := range numbers {
			input.In() <- num
		}
		input.Close()
	}()
	return input
}

func TestFanin(t *testing.T) {
	chs := make([]*chann.Chann[int], 10)
	for i := 0; i < 10; i++ {
		chs[i] = getInputChan()
	}

	out := chann.Fanin(chs...)
	count := 0
	for range out.Out() {
		count++
	}
	if count != 100 {
		t.Fatalf("Fanin failed")
	}
}

func TestLB(t *testing.T) {
	ins := make([]*chann.Chann[int], 10)
	for i := 0; i < 10; i++ {
		ins[i] = getInputChan()
	}
	outs := make([]*chann.Chann[int], 10)
	for i := 0; i < 10; i++ {
		outs[i] = chann.New[int](chann.Cap(10))
	}
	chann.LB(func(m int) int { return rand.Intn(m) }, ins, outs)
}
