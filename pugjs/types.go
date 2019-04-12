package pugjs

import (
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"sort"
	"strings"
)

type (
	// Object describes a pugjs JavaScript object
	Object interface {
		Member(name string) Object
		String() string
		iface() interface{}
		copy() Object
	}

	truer interface {
		True() bool
	}

	sortable interface {
		Order() []string
	}
)

// Convert an object
func Convert(in interface{}) Object {
	return convert(in)
}

func convert(in interface{}) Object {
	if in == nil {
		return Nil{}
	}

	if in, ok := in.(Object); ok {
		return in
	}

	val, ok := in.(reflect.Value)
	if !ok {
		val = reflect.ValueOf(in)
	}

	if !val.IsValid() {
		return Nil{}
	}

	if !val.CanInterface() {
		return Nil{}
	}

	if in, ok := val.Interface().(Object); ok {
		return in
	}

	if err, ok := in.(error); ok && err != nil {
		return String(fmt.Sprintf("Error: %+v", err))
	}

	switch val.Kind() {
	case reflect.Slice:
		array := &Array{
			items: make([]Object, val.Len()),
			o:     val.Interface(),
		}
		for i := 0; i < val.Len(); i++ {
			array.items[i] = convert(val.Index(i))
		}
		return array

	case reflect.Map:
		newMap := &Map{
			items: make(map[string]Object, val.Len()),
			o:     val.Interface(),
		}
		for _, k := range val.MapKeys() {
			// dereference interfaces
			if k.Kind() == reflect.Interface {
				k = k.Elem()
			}
			newMap.items[k.String()] = convert(val.MapIndex(k))
		}

		if sortable, ok := val.Interface().(sortable); ok {
			order := sortable.Order()
			newMap.order = make([]string, len(order))
			for i, o := range order {
				newMap.order[i] = o
			}
		}

		return newMap

	case reflect.Struct:
		newMap := &Map{
			o: val.Interface(),
		}
		// no item conversion here. It will be done on the fly on first member access

		return newMap

	case reflect.String:
		return String(val.String())

	case reflect.Interface:

		if val.Type().NumMethod() == 0 {
			return convert(val.Interface())
		}

		newMap := &Map{
			items: make(map[string]Object, val.Type().NumMethod()),
			o:     val.Interface(),
		}
		if !val.IsNil() {
			for i := 0; i < val.NumMethod(); i++ {
				newMap.items[lowerFirst(val.Type().Method(i).Name)] = convert(val.Method(i))
			}

			if m, ok := convert(val.Interface()).(*Map); ok {
				m.convert()
				for k, v := range m.items {
					newMap.items[k] = v
				}
			}
		}

		if sortable, ok := val.Interface().(sortable); ok {
			order := sortable.Order()
			newMap.order = make([]string, len(order))
			copy(newMap.order, order)
		}

		return newMap

	case reflect.Float32, reflect.Float64:
		return Number(val.Float())

	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		return Number(float64(val.Int()))

	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		return Number(float64(val.Uint()))

	case reflect.Complex128:
		return Nil{}

	case reflect.Func:
		return &Func{fnc: val}

	case reflect.Ptr:
		if val.IsValid() && val.Elem().IsValid() {
			newVal := convert(val.Elem())
			if m, ok := newVal.(*Map); ok {
				for i := 0; i < val.NumMethod(); i++ {
					m.Assign(lowerFirst(val.Type().Method(i).Name), convert(val.Method(i)))
				}
			}
			return newVal
		}
		return Nil{}

	case reflect.Uintptr:
		return Nil{}

	case reflect.Bool:
		return Bool(val.Bool())

	case reflect.Chan:
		// TODO iterable
		return Nil{}
	}

	panic(fmt.Sprintf("Cannot convert %#v %T %s %s", val, val, val.Type(), val.Kind()))
}

// Func type
type Func struct {
	fnc reflect.Value
}

// Member getter
func (f *Func) Member(name string) Object { return Nil{} }

// String formatter
func (f *Func) String() string { return f.fnc.String() }

// True getter
func (f *Func) True() bool { return true }

func (f *Func) copy() Object       { return &(*f) }
func (f *Func) iface() interface{} { return f.fnc.Interface() }

var AllowDeep = true

// MarshalJSON implementation
func (f *Func) MarshalJSON() ([]byte, error) {
	if f.fnc.Type().NumIn() == 0 && f.fnc.Type().NumOut() == 1 {
		if AllowDeep {
			return json.Marshal(convert(f.fnc.Call(nil)[0]))
		}
		// return function name as string, to avoid circular calls
		return json.Marshal(f.fnc.String())
	}
	return []byte(`"` + f.String() + `"`), nil
}

// Array type
type Array struct {
	items []Object
	o     interface{}
}

func (a *Array) Items() []Object {
	return a.items
}

func (a *Array) iface() interface{} { return a.o }

// String formatter
func (a *Array) String() string {
	tmp := make([]string, len(a.items))
	for i, v := range a.items {
		tmp[i] = v.String()
	}
	return strings.Join(tmp, " ")
}

// Member getter
func (a *Array) Member(name string) Object {
	switch name {
	case "length":
		return &Func{fnc: reflect.ValueOf(a.Length)}

	case "indexOf":
		return &Func{fnc: reflect.ValueOf(a.IndexOf)}

	case "join":
		return &Func{fnc: reflect.ValueOf(a.Join)}

	case "push":
		return &Func{fnc: reflect.ValueOf(a.Push)}

	case "pop":
		return &Func{fnc: reflect.ValueOf(a.Pop)}

	case "splice":
		return &Func{fnc: reflect.ValueOf(a.Splice)}

	case "slice":
		return &Func{fnc: reflect.ValueOf(a.Slice)}
	
	case "sort":
		return &Func{fnc: reflect.ValueOf(a.Sort)}
	}


	panic("field " + name + " not found")
}

// Splice an array
func (a *Array) Splice(n Number) Object {
	right := &Array{
		items: a.items[int(n):],
	}
	a.items = a.items[:int(n)]
	return right
}

// Slice an array
func (a *Array) Slice(n Number) Object {
	return &Array{
		items: a.items[int(n):],
	}
}

// Sort array
func (a *Array) Sort() Object {
	sort.Slice(a.items, func(i, j int) bool {
		return a.items[i].String() < a.items[j].String()
	})
	return Nil{}
}


// Length getter
func (a *Array) Length() Object {
	return Number(len(a.items))
}

// IndexOf array element
func (a *Array) IndexOf(what interface{}) Object {
	what = convert(what)
	for i, w := range a.items {
		if reflect.DeepEqual(w, what) {
			return Number(i)
		}
	}
	return Number(-1)
}

// Join array
func (a *Array) Join(sep string) Object {
	var aa []string

	for _, v := range a.items {
		aa = append(aa, v.String())
	}

	return String(strings.Join(aa, sep))
}

// Push into array
func (a *Array) Push(what Object) Object {
	a.items = append(a.items, what)
	return Nil{}
}

// Pop from array
func (a *Array) Pop() Object {
	last := a.items[len(a.items)-1]
	a.items = a.items[:len(a.items)-1]
	return last
}

func (a *Array) True() bool                   { return len(a.items) > 0 }      // True getter
func (a *Array) MarshalJSON() ([]byte, error) { return json.Marshal(a.items) } // MarshalJSON implementation

func (a *Array) copy() Object {
	c := &Array{
		items: make([]Object, len(a.items)),
	}

	for i, o := range a.items {
		c.items[i] = o.copy()
	}

	return c
}

// Map type
type Map struct {
	items map[string]Object
	o     interface{}
	order []string
}

func (m *Map) convert() {
	if m.items != nil {
		return
	}

	val, ok := m.o.(reflect.Value)
	if !ok {
		val = reflect.ValueOf(m.o)
	}

	m.items = make(map[string]Object, val.Type().NumField()+val.Type().NumMethod())

	for i := 0; i < val.NumField(); i++ {
		if val.Field(i).CanInterface() {
			m.items[lowerFirst(val.Type().Field(i).Name)] = convert(val.Field(i))
		}
	}

	for i := 0; i < val.NumMethod(); i++ {
		m.items[lowerFirst(val.Type().Method(i).Name)] = convert(val.Method(i))
	}

	if sortable, ok := val.Interface().(sortable); ok {
		order := sortable.Order()
		m.order = make([]string, len(order))
		for i, o := range order {
			m.order[i] = o
		}
	}
}

// ValueOf returns a new Value initialized to the concrete value
// stored in the m.items
func (m *Map) ValueOf() reflect.Value {
	m.convert()

	return reflect.ValueOf(m.items)
}

// Keys returns all map keys
func (m *Map) Keys() []string {
	m.convert()
	if len(m.order) > 0 {
		return m.order
	}

	result := make([]string, len(m.items))
	i := 0
	for key := range m.items {
		result[i] = key
		i = i + 1
	}

	m.order = result

	return result
}

func (m *Map) iface() interface{} { return m.o }

// AsStringMap helper
func (m *Map) AsStringMap() map[string]string {
	m.convert()
	stringMap := make(map[string]string)
	for key, value := range m.items {
		stringMap[key] = value.String()
	}

	return stringMap
}

// AsStringIfaceMap helper
func (m *Map) AsStringIfaceMap() map[string]interface{} {
	m.convert()
	iMap := make(map[string]interface{})
	for key, value := range m.items {
		iMap[key] = value.iface()
	}
	return iMap
}

// String formatter
func (m *Map) String() string {
	if m == nil {
		return ""
	}
	if s, ok := m.o.(fmt.Stringer); ok {
		return s.String()
	}
	b, err := m.MarshalJSON()
	if err != nil {
		panic(err)
	}
	return string(b)
}

// Assign a new item to the key
func (m *Map) Assign(key string, field Object) {
	m.convert()
	m.items[key] = field
	if len(m.order) > 0 {
		found := false
		for _, k := range m.order {
			if k == key {
				found = true
				break
			}
		}
		if !found {
			m.order = append(m.order, key)
		}
	}
}

// HasMember checks if a member exists
func (m *Map) HasMember(field string) bool {
	m.convert()
	_, hasMember := m.items[field]

	return hasMember
}

// Member getter
func (m *Map) Member(field string) Object {
	m.convert()
	if field == "__assign" {
		return &Func{fnc: reflect.ValueOf(func(k, v interface{}) Object {
			// if we have a ordered map we need to append to not lose it
			// this is only allowed to happen if we have an ordered list, otherwise we would
			// bring partial order into an unordered list.
			key := convert(k)
			if _, ok := m.items[key.String()]; len(m.order) > 0 && !ok {
				m.order = append(m.order, key.String())
			}
			m.items[key.String()] = convert(v)
			return Nil{}
		})}
	}

	if i, ok := m.items[field]; ok {
		return i
	}
	if i, ok := m.items[upperFirst(field)]; ok {
		return i
	}
	if i, ok := m.items[strings.Title(field)]; ok {
		return i
	}

	field = strings.NewReplacer("id", "ID", "url", "URL", "api", "API").Replace(field)

	if i, ok := m.items[field]; ok {
		return i
	}
	if i, ok := m.items[upperFirst(field)]; ok {
		return i
	}
	if i, ok := m.items[strings.Title(field)]; ok {
		return i
	}

	return Nil{}
}

// MarshalJSON implementation
func (m *Map) MarshalJSON() ([]byte, error) {
	if s, ok := m.o.(json.Marshaler); ok {
		return s.MarshalJSON()
	}
	m.convert()
	tmp := make(map[string]interface{}, len(m.items))
	for k, v := range m.items {
		tmp[lowerFirst(k)] = v
	}
	return json.Marshal(tmp)
}

// True getter
func (m *Map) True() bool {
	if m.o != nil && reflect.DeepEqual(reflect.Zero(reflect.TypeOf(m.o)).Interface(), m.o) {
		return false
	}
	m.convert()
	return len(m.items) > 0
}

func (m *Map) copy() Object {
	c := &Map{
		items: make(map[string]Object, len(m.items)),
		o:     m.o,
	}

	for k, v := range m.items {
		c.items[k] = v.copy()
	}

	return c
}

// String type
type String string

// String formatter
func (s String) String() string { return string(s) }

func (s String) iface() interface{} { return s }

// Member getter
func (s String) Member(field string) Object {
	switch field {
	case "charAt":
		return &Func{fnc: reflect.ValueOf(s.CharAt)}
	case "toUpperCase":
		return &Func{fnc: reflect.ValueOf(s.ToUpperCase)}
	case "split":
		return &Func{fnc: reflect.ValueOf(s.Split)}
	case "slice":
		return &Func{fnc: reflect.ValueOf(s.Slice)}
	case "replace":
		return &Func{fnc: reflect.ValueOf(s.Replace)}
	case "length":
		return &Func{fnc: reflect.ValueOf(s.Length)}
	case "indexOf":
		return &Func{fnc: reflect.ValueOf(s.IndexOf)}
	}
	return Nil{}
}

// CharAt function
func (s String) CharAt(nPos Number) string {
	pos := int(nPos)
	if pos >= len(s) {
		return ""
	}
	return string(s[pos])
}

// IndexOf Js func
func (s String) IndexOf(delim string) int { return strings.Index(string(s), delim) }

// ToUpperCase converter
func (s String) ToUpperCase() string { return strings.ToUpper(string(s)) }

// Split splitter
func (s String) Split(delim string) []string { return strings.Split(string(s), delim) }

// Slice a string
func (s String) Slice(nfrom Number, toList ...Number) string {
	strLength := len(s)
	from := int(nfrom)

	if from > strLength {
		return ""
	}

	if from < 0 {
		from = strLength + from
	}

	to := len(s)
	if len(toList) > 0 {
		to = int(toList[0])
	}

	if to < 0 {
		to = strLength + to
	}

	return string(s[from:to])
}

// Replace string values
func (s String) Replace(what, with String) String {
	return String(strings.Replace(string(s), string(what), string(with), -1))
}

// Return string length
func (s String) Length() int { return len(s) }

func (s String) copy() Object { return s }

// Number type
type Number float64

func (n Number) Member(string) Object { return Nil{} }                             // Member getter
func (n Number) String() string       { return big.NewFloat(float64(n)).String() } // String formatter
func (n Number) copy() Object         { return n }
func (n Number) iface() interface{}   { return n }

// Bool type
type Bool bool

func (b Bool) Member(string) Object { return Nil{} }                      // Member getter
func (b Bool) String() string       { return fmt.Sprintf("%v", bool(b)) } // String formatter
func (b Bool) True() bool           { return bool(b) }                    // True getter
func (b Bool) copy() Object         { return b }
func (b Bool) iface() interface{}   { return b }

// Nil type
type Nil struct{}

func (n Nil) Member(string) Object         { return Nil{} }               // Member is always nil
func (n Nil) String() string               { return "" }                  // String is always empty
func (n Nil) MarshalJSON() ([]byte, error) { return []byte("null"), nil } // MarshalJSON
func (n Nil) True() bool                   { return false }               // True is always false
func (n Nil) copy() Object                 { return Nil{} }
func (n Nil) iface() interface{}           { return nil }
