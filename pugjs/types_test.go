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

func TestArray_Sort(t *testing.T) {
	input := convert([]string{"zuletzt", "test", "somewhere", "anfang", "anywhere"}).(*Array)
	expectedResult := convert([]string{"anfang", "anywhere", "somewhere", "test", "zuletzt"}).(*Array)

	input.Sort()
	assert.Equal(t, expectedResult.items, input.items)
}

func TestIndex_Of(t *testing.T) {

	tests := []struct {
		name           string
		input          *Array
		search         string
		expectedResult Number
	}{
		{
			name:           "can be found at position 2",
			input:          convert([]string{"can", "you", "find", "me"}).(*Array),
			search:         "find",
			expectedResult: Number(2),
		},
		{
			name:           "i am hidden",
			input:          convert([]string{"i", "am", "hidden"}).(*Array),
			search:         "find",
			expectedResult: Number(-1),
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expectedResult, tt.input.IndexOf(tt.search))
	}

}

func TestArray_Pop(t *testing.T) {

	tests := []struct {
		name           string
		input          *Array
		expectedResult Object
	}{
		{
			name:           "test pop with string",
			input:          convert([]string{"something", "test", "somewhere", "whatever", "last"}).(*Array),
			expectedResult: String("last"),
		},
		{
			name:           "test pop with int",
			input:          convert([]int{1, 2, 4, 3}).(*Array),
			expectedResult: Number(3),
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expectedResult, tt.input.Pop())
	}
}

func TestArray_Shift(t *testing.T) {

	tests := []struct {
		name           string
		input          *Array
		expectedArray  *Array
		expectedResult Object
	}{
		{
			name:           "test shift with string",
			input:          convert([]string{"something", "test", "somewhere", "whatever", "last"}).(*Array),
			expectedArray:  convert([]string{"test", "somewhere", "whatever", "last"}).(*Array),
			expectedResult: String("something"),
		},
		{
			name:           "test shift with int",
			input:          convert([]int{1, 2, 4, 3}).(*Array),
			expectedArray:  convert([]int{2, 4, 3}).(*Array),
			expectedResult: Number(1),
		},
		{
			name:           "test shift with int and array of length 1",
			input:          convert([]int{7}).(*Array),
			expectedArray:  convert([]int{}).(*Array),
			expectedResult: Number(7),
		},
		{
			name:           "test shift with int and empty input array",
			input:          convert([]int{}).(*Array),
			expectedArray:  convert([]int{}).(*Array),
			expectedResult: nil,
		},
	}

	for _, tt := range tests {
		array := tt.input
		result := array.Shift()
		assert.Equal(t, tt.expectedResult, result)
		assert.Equal(t, tt.expectedArray.items, array.items)
	}
}

func TestArray_Unshift(t *testing.T) {

	tests := []struct {
		name           string
		array          *Array
		addElement     Object
		expectedResult *Array
		expectedLength int
	}{
		{
			name:           "test unshift with string",
			array:          convert([]string{"something", "test"}).(*Array),
			addElement:     String("first"),
			expectedResult: convert([]string{"first", "something", "test"}).(*Array),
			expectedLength: 3,
		},
		{
			name:           "test unshift with int",
			array:          convert([]int{4, 3}).(*Array),
			addElement:     Number(1),
			expectedResult: convert([]int{1, 4, 3}).(*Array),
			expectedLength: 3,
		},
	}

	for _, tt := range tests {
		array := tt.array
		length := array.Unshift(tt.addElement)
		assert.Equal(t, tt.expectedLength, length)
		assert.Equal(t, tt.expectedResult.items, array.items)
	}
}

func TestArray_True(t *testing.T) {
	tests := []struct {
		name           string
		input          *Array
		expectedResult bool
	}{
		{
			name:           "length greater zero, should be true",
			input:          convert([]int{1, 2, 4, 3}).(*Array),
			expectedResult: true,
		},
		{
			name:           "empty, should be false",
			input:          convert([]int{}).(*Array),
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expectedResult, tt.input.True())
	}
}

func TestMap_Assign(t *testing.T) {
	tests := []struct {
		name           string
		input          *Map
		key            string
		value          Object
		expectedResult *Map
	}{
		{
			name:           "empty map",
			input:          new(Map),
			key:            "first",
			value:          String("something"),
			expectedResult: &Map{items: map[string]Object{"first": String("something")}, o: interface{}(nil), order: []string{}},
		},
		{
			name:           "map with existing key, replace value",
			input:          &Map{items: map[string]Object{"first": String("something")}, o: interface{}(nil), order: []string{"first"}},
			key:            "first",
			value:          String("different"),
			expectedResult: &Map{items: map[string]Object{"first": String("different")}, o: interface{}(nil), order: []string{"first"}},
		},
		{
			name:           "map with new key, append",
			input:          &Map{items: map[string]Object{"first": String("something")}, o: interface{}(nil), order: []string{"first"}},
			key:            "second",
			value:          String("append"),
			expectedResult: &Map{items: map[string]Object{"first": String("something"), "second": String("append")}, o: interface{}(nil), order: []string{"first", "second"}},
		},
	}

	for _, tt := range tests {
		tt.input.Assign(tt.key, tt.value)
		assert.Equal(t, tt.expectedResult, tt.input)
	}
}

func TestMap_String(t *testing.T) {
	tests := []struct {
		name           string
		input          *Map
		expectedResult string
	}{
		{
			name:           "nil map",
			input:          nil,
			expectedResult: "",
		},
		{
			name:           "empty map",
			input:          new(Map),
			expectedResult: "{}",
		},
		{
			name:           "a real map with a string",
			input:          &Map{items: map[string]Object{"first": String("something")}, o: interface{}(nil), order: []string{"first"}},
			expectedResult: "{\"first\":\"something\"}",
		},
		{
			name:           "a real map with a value that has a string method",
			input:          &Map{items: map[string]Object{"first": Bool(true)}, o: interface{}(nil), order: []string{"first"}},
			expectedResult: "{\"first\":true}",
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expectedResult, tt.input.String())
	}
}

func TestMap_HasMember(t *testing.T) {
	tests := []struct {
		name           string
		input          *Map
		member         string
		expectedResult bool
	}{
		{
			name:           "empty map",
			input:          new(Map),
			member:         "",
			expectedResult: false,
		},
		{
			name:           "map with member",
			input:          &Map{items: map[string]Object{"first": String("something"), "findMe": String("something")}, o: interface{}(nil), order: []string{"first", "findMe"}},
			member:         "findMe",
			expectedResult: true,
		},
		{
			name:           "map without member",
			input:          &Map{items: map[string]Object{"first": String("something"), "findMeNot": String("something")}, o: interface{}(nil), order: []string{"first", "findMeNot"}},
			member:         "findMe",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expectedResult, tt.input.HasMember(tt.member))
	}
}
