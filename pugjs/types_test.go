package pugjs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNil(t *testing.T) {
	n := Nil{}

	assert.Equal(t, false, n.True())
	assert.Equal(t, "", n.String())
	assert.Equal(t, Nil{}, n.Member(""))
	assert.Equal(t, Nil{}, n.Member("aaa"))
	assert.Equal(t, Nil{}, n.copy())
}

func TestBool(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		b := Bool(true)
		assert.Equal(t, true, b.True())
		assert.Equal(t, "true", b.String())
		assert.Equal(t, Nil{}, b.Member(""))
		assert.Equal(t, Nil{}, b.Member("aaa"))
		assert.Equal(t, Bool(true), b.copy())
	})

	t.Run("false", func(t *testing.T) {
		b := Bool(false)
		assert.Equal(t, false, b.True())
		assert.Equal(t, "false", b.String())
		assert.Equal(t, Nil{}, b.Member(""))
		assert.Equal(t, Nil{}, b.Member("aaa"))
		assert.Equal(t, Bool(false), b.copy())
	})
}

func TestNumber(t *testing.T) {
	n := Number(1.2)

	assert.Equal(t, "1.2", n.String())
	assert.Equal(t, "1", Number(1).String())
	assert.Equal(t, "0", Number(0).String())
	assert.Equal(t, "-1", Number(-1).String())

	assert.Equal(t, Nil{}, n.Member(""))
	assert.Equal(t, Nil{}, n.Member("aaa"))

	assert.Equal(t, n, n.copy())
}

func TestArray_Splice(t *testing.T) {
	arr := convert([]int{1, 2, 3, 4, 5}).(*Array)

	assert.Len(t, arr.items, 5)
	leftover := arr.Splice(Number(2)).(*Array)
	assert.Len(t, arr.items, 2)
	assert.Len(t, leftover.items, 3)

	assert.Contains(t, arr.items, Number(1))
	assert.Contains(t, arr.items, Number(2))
	assert.Contains(t, leftover.items, Number(3))
	assert.Contains(t, leftover.items, Number(4))
	assert.Contains(t, leftover.items, Number(5))
}

func TestArray_Slice(t *testing.T) {
	arr := convert([]int{1, 2, 3, 4, 5}).(*Array)

	assert.Len(t, arr.items, 5)
	leftover := arr.Slice(Number(2)).(*Array)
	assert.Len(t, arr.items, 5)
	assert.Len(t, leftover.items, 3)

	assert.Contains(t, arr.items, Number(1))
	assert.Contains(t, arr.items, Number(2))
	assert.Contains(t, arr.items, Number(3))
	assert.Contains(t, arr.items, Number(4))
	assert.Contains(t, arr.items, Number(5))
	assert.Contains(t, leftover.items, Number(3))
	assert.Contains(t, leftover.items, Number(4))
	assert.Contains(t, leftover.items, Number(5))
}

func TestString_Slice(t *testing.T) {
	s := String("test123")

	assert.Equal(t, s.Slice(Number(1)), "est123")
	assert.Equal(t, s.Slice(Number(-1)), "3")

	assert.Equal(t, s.Slice(1, 3), "es")
	assert.Equal(t, s.Slice(1, -2), "est1")
	assert.Equal(t, s.Slice(-4, 4), "t")
	assert.Equal(t, s.Slice(-4, -2), "t1")
}

func TestString_Length(t *testing.T) {
	assert.Equal(t, String("").Length(), 0)
	assert.Equal(t, String("test123").Length(), 7)
}

func TestString_ToLowerCase(t *testing.T) {
	assert.Equal(t, "test", String("TeSt").ToLowerCase())
	assert.Equal(t, "test123", String("TeSt123").ToLowerCase())
}

func TestMap(t *testing.T) {
	m := new(Map)
	assert.False(t, m.True())
}
