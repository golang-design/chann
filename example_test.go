// Copyright 2021 The golang.design Initiative Authors.
// All rights reserved. Use of this source code is governed
// by a MIT license that can be found in the LICENSE file.
//
// Written by Changkun Ou <changkun.de>

package chann_test

import (
	"fmt"

	"golang.design/x/chann"
)

func ExampleNew() {
	ch := chann.New[int]()

	go func() {
		for i := 0; i < 10; i++ {
			ch.In() <- i // never block
		}
		ch.Close()
	}()

	sum := 0
	for i := range ch.Out() {
		sum += i
	}
	fmt.Println(sum)
	// Output:
	// 45
}
