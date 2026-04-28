package checksum

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBytesToCheckSum(t *testing.T) {
	assert.EqualValues(t, BytesToChecksum([]byte{0x0, 0x0}), 0xffff)
	assert.EqualValues(t, BytesToChecksum([]byte{0x0, 0x1}), 0xfffe)
	assert.EqualValues(t, BytesToChecksum([]byte{0x1, 0x1}), 0xfefe)
	assert.EqualValues(t, BytesToChecksum([]byte{0x1, 0x1, 0x1, 0x1}), 0xfdfd)
}

func TestCheckSum(t *testing.T) {
	assert.True(t, CheckSum([]byte{0x0, 0x0}, 0xffff))
	assert.True(t, CheckSum([]byte{0x0, 0x1}, 0xfffe))
	assert.True(t, CheckSum([]byte{0x1, 0x1}, 0xfefe))
	assert.True(t, CheckSum([]byte{0x1, 0x1, 0x1, 0x1}, 0xfdfd))

	assert.False(t, CheckSum([]byte{0x12, 0x0}, 0xffff))
	assert.False(t, CheckSum([]byte{0xff, 0x1}, 0xfffe))
	assert.False(t, CheckSum([]byte{0x1, 0x51}, 0xfefe))
	assert.False(t, CheckSum([]byte{0x1, 0x2, 0x1, 0x1}, 0xfdfd))

	var data [1000]byte
	for range 100 {
		for i := range 1000 {
			data[i] = byte(rand.Intn(256))
		}
		checksum := BytesToChecksum(data[:])
		assert.True(t, CheckSum(data[:], checksum))
		data[rand.Intn(1000)] += byte(1 + rand.Intn(255))
		assert.False(t, CheckSum(data[:], checksum))
	}
}
