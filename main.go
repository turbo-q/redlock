package main

import (
	"fmt"
	"redlock/helper"
	"redlock/redlock"
	"sync"
)

var share = 1

func main() {

	const k = "testkey"
	const gonum = 10

	var wg sync.WaitGroup
	wg.Add(gonum)
	for i := 0; i < gonum; i++ {
		go func(i int) {
			var ok bool
			var rl *redlock.RedLock
			defer wg.Done()
			// 没获得锁则轮询拿锁
			for !ok {
				fmt.Printf("goroutine %d获取锁失败\n", i)
				rl, ok = setnx(k, 30000)
			}
			defer rl.Release()

			// 获得锁操作共享变量
			for j := 0; j < 1000; j++ {
				share++
			}
		}(i)
	}

	wg.Wait()

	fmt.Printf("final share value %d", share)
}

func setnx(k string, timeout int) (*redlock.RedLock, bool) {
	var rl *redlock.RedLock = new(redlock.RedLock)
	rl.K = k
	rl.V = helper.UUID()

	return rl, rl.SetNX(timeout)
}
