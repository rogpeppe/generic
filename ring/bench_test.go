package ring

import "testing"

func BenchmarkQueueOneItem(b *testing.B) {
	var buf Buffer[int]
	for range b.N {
		buf.PushStart(2)
		buf.PopEnd()
	}
}

func BenchmarkSliceQueueOneItem(b *testing.B) {
	var buf []int
	for range b.N {
		buf = append(buf, 2)
		buf = buf[1:]
	}
}
