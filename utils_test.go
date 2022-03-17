// Copyright 2021 The golang.design Initiative Authors.
// All rights reserved. Use of this source code is governed
// by a MIT license that can be found in the LICENSE file.
//
// Written by Changkun Ou <changkun.de>

package chann_test

import (
	"testing"

	"golang.design/x/chann"
)

func TestRanger(t *testing.T) {
	s, r := chann.Ranger[int]()

	go func() {
		s.Send(42)
	}()

	n, ok := r.Next()
	if !ok {
		t.Fatalf("cannot receive from senter")
	}
	t.Log(n)
}
