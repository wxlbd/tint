package tint

import "sync"

type buffer []byte

// bufPool 是用于存储缓冲区的同步池，通过调用 New 方法获取新的缓冲区
var bufPool = sync.Pool{
	New: func() any {
		b := make(buffer, 0, 1024)
		return &b
	},
}

// newBuffer 从 bufPool 中获取一个缓冲区
func newBuffer() *buffer {
	return bufPool.Get().(*buffer)
}

// Free 将缓冲区放回池中
// 检查缓冲区的容量是否小于最大缓冲区大小;
// 如果是，则将缓冲区设置为空片并将其放回池中
func (b *buffer) Free() {
	const maxBufferSize = 16 << 10
	// 如果缓冲区容量小于等于最大缓冲区大小，则将缓冲区置为空片并放回池中
	if cap(*b) <= maxBufferSize {
		*b = (*b)[:0]
		bufPool.Put(b)
	}
}

// Write 将字节切片追加到缓冲区末尾，并返回追加的字节数
func (b *buffer) Write(bytes []byte) int {
	*b = append(*b, bytes...)
	return len(bytes)
}

// WriteByte 将指定字节追加到缓冲区末尾
func (b *buffer) WriteByte(char byte) {
	*b = append(*b, char)
}

// WriteString 将字符串追加到缓冲区末尾，并返回追加的字符串长度
func (b *buffer) WriteString(str string) int {
	*b = append(*b, str...)
	return len(str)
}

// WriteStringIf 判断 ok 的值，如果为真则将字符串追加到缓冲区末尾，并返回追加的字符串长度
// 如果为假，则返回 0 和 nil
func (b *buffer) WriteStringIf(ok bool, str string) int {
	if !ok {
		return 0
	}
	b.WriteString(str)
	return len(str)
}
