// Copyright 2022 The golang.design Initiative Authors.
// All rights reserved. Use of this source code is governed
// by a MIT license that can be found in the LICENSE file.
//
// Written by Changkun Ou <changkun.de>

package chann

// IsClosed checks if a channel is entirely closed or not.
// This function is only exported for testing.
func IsClosed[T any](ch *Chann[T]) bool {
	return ch.isClosed()
}
