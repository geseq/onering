//+build histogram

package onering

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

// courtesy of Kelly Sommers aka @kellabyte

const (
	sampleE     = 12
	sampleTimes = 1 << sampleE
	sampleMask  = sampleTimes - 1
)

func BenchmarkResponseTimesRing(b *testing.B) {
	var ring = New{Size: 8192, BatchSize: 127}.SPSC()
	var wg sync.WaitGroup
	wg.Add(2)
	var diffs = make([]int64, (b.N/sampleTimes)+1)
	var times = make([]int64, (b.N/sampleTimes)+1)
	var zero int64 = 0
	b.ResetTimer()
	go func(n int) {
		runtime.LockOSThread()
		var t1 = time.Now().UnixNano()
		var j = 0
		for i := 1; i < n; i++ {
			var v = &zero
			if i&sampleMask == 0 {
				times[j] = t1
				v = &times[j]
				t1 = time.Now().UnixNano()
				j++
			} else {
				v = &zero
			}
			ring.Put(v)
		}
		wg.Done()
	}(b.N + 1)
	go func(n int) {
		runtime.LockOSThread()
		var i int = 0
		ring.Consume(func(it Iter, v *int64) {
			if *v != 0 {
				diffs[i] = (time.Now().UnixNano() - *v) / sampleTimes
				i++
			}
			n--
			if n <= 0 {
				ring.Close()
			}
		})
		wg.Done()
	}(b.N)

	wg.Wait()
}

func BenchmarkResponseTimesChannel(b *testing.B) {
	var ch = make(chan int64, 8192)
	var wg sync.WaitGroup
	wg.Add(2)
	var diffs = make([]int64, (b.N/sampleTimes)+1)
	b.ResetTimer()
	go func(n int) {
		runtime.LockOSThread()
		var t1 = time.Now().UnixNano()
		for i := 1; i < n; i++ {
			var v int64 = 0
			if i&sampleMask == 0 {
				v = t1
				t1 = time.Now().UnixNano()
			}
			ch <- v
		}
		close(ch)
		wg.Done()
	}(b.N + 1)
	go func(n int) {
		runtime.LockOSThread()
		var i = 0
		for v := range ch {
			if v != 0 {
				diffs[i] = (time.Now().UnixNano() - v) / sampleTimes
				i++
			}
		}
		wg.Done()
	}(b.N)

	wg.Wait()
}
