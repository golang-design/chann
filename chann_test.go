// Copyright 2021 Changkun Ou <changkun.de>. All rights reserved.
// Use of this source code is governed by a MIT license that
// can be found in the LICENSE file.

package chann_test

import (
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"golang.design/x/chann"
)

func TestChan(t *testing.T) {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(4))
	N := 200
	if testing.Short() {
		N = 20
	}
	for chanCap := 0; chanCap < N; chanCap++ {
		{
			// Ensure that receive from empty chan blocks.
			c := chann.New[int](chann.Cap(chanCap))
			recv1 := false
			go func() {
				_ = <-c.Out()
				recv1 = true
			}()
			recv2 := false
			go func() {
				_, _ = <-c.Out()
				recv2 = true
			}()
			time.Sleep(time.Millisecond)
			if recv1 || recv2 {
				t.Fatalf("chan[%d]: receive from empty chan", chanCap)
			}
			// Ensure that non-blocking receive does not block.
			select {
			case _ = <-c.Out():
				t.Fatalf("chan[%d]: receive from empty chan", chanCap)
			default:
			}
			select {
			case _, _ = <-c.Out():
				t.Fatalf("chan[%d]: receive from empty chan", chanCap)
			default:
			}
			c.In() <- 0
			c.In() <- 0
		}

		{
			// Ensure that send to full chan blocks.
			c := chann.New[int](chann.Cap(chanCap))
			for i := 0; i < chanCap; i++ {
				c.In() <- i
			}
			sent := uint32(0)
			go func() {
				c.In() <- 0
				atomic.StoreUint32(&sent, 1)
			}()
			time.Sleep(time.Millisecond)
			if atomic.LoadUint32(&sent) != 0 {
				t.Fatalf("chan[%d]: send to full chan", chanCap)
			}
			// Ensure that non-blocking send does not block.
			select {
			case c.In() <- 0:
				t.Fatalf("chan[%d]: send to full chan", chanCap)
			default:
			}
			<-c.Out()
		}

		{
			// Ensure that we receive 0 from closed chan.
			c := chann.New[int](chann.Cap(chanCap))
			for i := 0; i < chanCap; i++ {
				c.In() <- i
			}
			c.Close()
			for i := 0; i < chanCap; i++ {
				v := <-c.Out()
				if v != i {
					t.Fatalf("chan[%d]: received %v, expected %v", chanCap, v, i)
				}
			}
			if v := <-c.Out(); v != 0 {
				t.Fatalf("chan[%d]: received %v, expected %v", chanCap, v, 0)
			}
			if v, ok := <-c.Out(); v != 0 || ok {
				t.Fatalf("chan[%d]: received %v/%v, expected %v/%v", chanCap, v, ok, 0, false)
			}
		}

		{
			// Ensure that close unblocks receive.
			c := chann.New[int](chann.Cap(chanCap))
			done := make(chan bool)
			go func() {
				v, ok := <-c.Out()
				done <- v == 0 && ok == false
			}()
			time.Sleep(time.Millisecond)
			c.Close()
			if !<-done {
				t.Fatalf("chan[%d]: received non zero from closed chan", chanCap)
			}
		}

		{
			// Send 100 integers,
			// ensure that we receive them non-corrupted in FIFO order.
			c := chann.New[int](chann.Cap(chanCap))
			go func() {
				for i := 0; i < 100; i++ {
					c.In() <- i
				}
			}()
			for i := 0; i < 100; i++ {
				v := <-c.Out()
				if v != i {
					t.Fatalf("chan[%d]: received %v, expected %v", chanCap, v, i)
				}
			}

			// Same, but using recv2.
			go func() {
				for i := 0; i < 100; i++ {
					c.In() <- i
				}
			}()
			for i := 0; i < 100; i++ {
				v, ok := <-c.Out()
				if !ok {
					t.Fatalf("chan[%d]: receive failed, expected %v", chanCap, i)
				}
				if v != i {
					t.Fatalf("chan[%d]: received %v, expected %v", chanCap, v, i)
				}
			}

			// Send 1000 integers in 4 goroutines,
			// ensure that we receive what we send.
			const P = 4
			const L = 1000
			for p := 0; p < P; p++ {
				go func() {
					for i := 0; i < L; i++ {
						c.In() <- i
					}
				}()
			}
			done := chann.New[map[int]int](chann.Cap(0))
			for p := 0; p < P; p++ {
				go func() {
					recv := make(map[int]int)
					for i := 0; i < L; i++ {
						v := <-c.Out()
						recv[v] = recv[v] + 1
					}
					done.In() <- recv
				}()
			}
			recv := make(map[int]int)
			for p := 0; p < P; p++ {
				for k, v := range <-done.Out() {
					recv[k] = recv[k] + v
				}
			}
			if len(recv) != L {
				t.Fatalf("chan[%d]: received %v values, expected %v", chanCap, len(recv), L)
			}
			for _, v := range recv {
				if v != P {
					t.Fatalf("chan[%d]: received %v values, expected %v", chanCap, v, P)
				}
			}
		}

		{
			// Test len/cap.
			c := chann.New[int](chann.Cap(chanCap))
			if c.ApproxLen() != 0 || c.Cap() != chanCap {
				t.Fatalf("chan[%d]: bad len/cap, expect %v/%v, got %v/%v", chanCap, 0, chanCap, c.ApproxLen(), c.Cap())
			}
			for i := 0; i < chanCap; i++ {
				c.In() <- i
			}
			if c.ApproxLen() != chanCap || c.Cap() != chanCap {
				t.Fatalf("chan[%d]: bad len/cap, expect %v/%v, got %v/%v", chanCap, chanCap, chanCap, c.ApproxLen(), c.Cap())
			}
		}
	}
}

func TestNonblockRecvRace(t *testing.T) {
	n := 10000
	if testing.Short() {
		n = 100
	}
	for i := 0; i < n; i++ {
		c := chann.New[int](chann.Cap(1))
		c.In() <- 1
		go func() {
			select {
			case <-c.Out():
			default:
				t.Error("chan is not ready")
			}
		}()
		c.Close()
		<-c.Out()
		if t.Failed() {
			return
		}
	}
}

// This test checks that select acts on the state of the channels at one
// moment in the execution, not over a smeared time window.
// In the test, one goroutine does:
//	create c1, c2
//	make c1 ready for receiving
//	create second goroutine
//	make c2 ready for receiving
//	make c1 no longer ready for receiving (if possible)
// The second goroutine does a non-blocking select receiving from c1 and c2.
// From the time the second goroutine is created, at least one of c1 and c2
// is always ready for receiving, so the select in the second goroutine must
// always receive from one or the other. It must never execute the default case.
func TestNonblockSelectRace(t *testing.T) {
	n := 1000
	if testing.Short() {
		n = 1000
	}
	done := chann.New[bool](chann.Cap(1))
	for i := 0; i < n; i++ {
		c1 := chann.New[int]()
		c2 := chann.New[int]()
		c1.In() <- 1
		go func() {
			runtime.Gosched()
			select {
			case <-c1.Out():
			case <-c2.Out():
			default:
				done.In() <- false
				return
			}
			done.In() <- true
		}()
		c2.In() <- 1
		select {
		case <-c1.Out():
		default:
		}
		if !<-done.Out() {
			t.Fatal("no chan is ready")
		}
	}
}

// Same as TestNonblockSelectRace, but close(c2) replaces c2 <- 1.
func TestNonblockSelectRace2(t *testing.T) {
	n := 100000
	if testing.Short() {
		n = 1000
	}
	done := make(chan bool, 1)
	for i := 0; i < n; i++ {
		c1 := make(chan int, 1)
		c2 := make(chan int)
		c1 <- 1
		go func() {
			select {
			case <-c1:
			case <-c2:
			default:
				done <- false
				return
			}
			done <- true
		}()
		close(c2)
		select {
		case <-c1:
		default:
		}
		if !<-done {
			t.Fatal("no chan is ready")
		}
	}
}
