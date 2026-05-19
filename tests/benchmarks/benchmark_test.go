package benchmarks_test

import (
	"testing"

	"github.com/codingminions/blockdev"
)

// Helpers — duplicated rather than imported because tests/e2e and
// tests/benchmarks are separate test packages.

func newDeviceWithBase(blocks int) *blockdev.BlockDevice {
	base := make([]byte, blocks*blockdev.BlockSize)
	bd, _ := blockdev.New(base)
	return bd
}

func newDeviceWithOverlay(deviceBlocks, populatedBlocks int) *blockdev.BlockDevice {
	bd := newDeviceWithBase(deviceBlocks)
	data := make([]byte, blockdev.BlockSize)
	for i := 0; i < populatedBlocks; i++ {
		bd.WriteAt(data, int64(i)*blockdev.BlockSize)
	}
	return bd
}

// ============ ReadAt ============

func BenchmarkReadAt_FromBase_4KB(b *testing.B) {
	bd := newDeviceWithBase(256)
	p := make([]byte, blockdev.BlockSize)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bd.ReadAt(p, 0)
	}
}

func BenchmarkReadAt_FromOverlay_4KB(b *testing.B) {
	bd := newDeviceWithOverlay(256, 256)
	p := make([]byte, blockdev.BlockSize)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bd.ReadAt(p, 0)
	}
}

func BenchmarkReadAt_Multi_64KB(b *testing.B) {
	bd := newDeviceWithBase(256)
	p := make([]byte, 16*blockdev.BlockSize)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bd.ReadAt(p, 0)
	}
}

// ============ WriteAt ============

func BenchmarkWriteAt_OverwriteSameBlock(b *testing.B) {
	bd := newDeviceWithBase(256)
	data := make([]byte, blockdev.BlockSize)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bd.WriteAt(data, 0)
	}
}

func BenchmarkWriteAt_DifferentBlocks(b *testing.B) {
	const devBlocks = 1024
	bd := newDeviceWithBase(devBlocks)
	data := make([]byte, blockdev.BlockSize)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bd.WriteAt(data, int64(i%devBlocks)*blockdev.BlockSize)
	}
}

func BenchmarkWriteAt_Multi_64KB(b *testing.B) {
	bd := newDeviceWithBase(256)
	data := make([]byte, 16*blockdev.BlockSize)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bd.WriteAt(data, 0)
	}
}

// ============ Serialize ============

func BenchmarkSerialize_Empty(b *testing.B) {
	bd := newDeviceWithBase(256)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = bd.Serialize()
	}
}

func BenchmarkSerialize_1Block(b *testing.B) {
	bd := newDeviceWithOverlay(256, 1)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = bd.Serialize()
	}
}

func BenchmarkSerialize_100Blocks(b *testing.B) {
	bd := newDeviceWithOverlay(256, 100)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = bd.Serialize()
	}
}

func BenchmarkSerialize_1000Blocks(b *testing.B) {
	bd := newDeviceWithOverlay(1024, 1000)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = bd.Serialize()
	}
}

// ============ Deserialize ============

func BenchmarkDeserialize_100Blocks(b *testing.B) {
	bd := newDeviceWithOverlay(256, 100)
	base := make([]byte, 256*blockdev.BlockSize)
	blob := bd.Serialize()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = blockdev.Deserialize(blob, base)
	}
}

func BenchmarkDeserialize_1000Blocks(b *testing.B) {
	bd := newDeviceWithOverlay(1024, 1000)
	base := make([]byte, 1024*blockdev.BlockSize)
	blob := bd.Serialize()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = blockdev.Deserialize(blob, base)
	}
}

// ============ Concurrent ============

func BenchmarkConcurrent_ReadOverlay(b *testing.B) {
	bd := newDeviceWithOverlay(256, 256)
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		p := make([]byte, blockdev.BlockSize)
		for pb.Next() {
			bd.ReadAt(p, 0)
		}
	})
}

func BenchmarkConcurrent_MixedReadWrite(b *testing.B) {
	bd := newDeviceWithOverlay(256, 256)
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		read := make([]byte, blockdev.BlockSize)
		write := make([]byte, blockdev.BlockSize)
		i := 0
		for pb.Next() {
			if i&1 == 0 {
				bd.ReadAt(read, 0)
			} else {
				bd.WriteAt(write, 0)
			}
			i++
		}
	})
}
