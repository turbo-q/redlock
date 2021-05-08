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
			var rl *redlock.RedLock = &redlock.RedLock{
				K: k,
				V: helper.UUID(),
				Cfg: redlock.Config{
					Timeout: 30000,
				},
			}
			defer wg.Done()
			// 没获得锁则轮询拿锁
			for !ok {
				fmt.Printf("goroutine %d获取锁失败\n", i)
				ok = rl.Lock()
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
