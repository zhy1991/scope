package report

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"

	"github.com/ugorji/go/codec"
)

type stringSetEntry struct {
	key   string
	Value StringSet `json:"value"`
}

// Sets is a string->set-of-strings map.
type Sets []stringSetEntry

// MakeSets returns EmptySets
func MakeSets() Sets {
	return Sets{}
}

// Keys returns the keys for this set
func (s Sets) Keys() []string {
	keys := make([]string, s.Size())
	for i, v := range s {
		keys[i] = v.key
	}
	return keys
}

// locate the position where key should go, either at the end or in the middle
func (s Sets) locate(key string) int {
	return sort.Search(len(s), func(i int) bool {
		return s[i].key >= key
	})
}

// Add the given value to the Sets, merging the contents if the key is already there.
func (s Sets) Add(key string, value StringSet) Sets {
	i := s.locate(key)
	oldEntries := s
	if i == len(s) {
		s = make([]stringSetEntry, len(oldEntries)+1)
		copy(s, oldEntries)
	} else if s[i].key == key {
		value = value.Merge(s[i].Value)
		s = make([]stringSetEntry, len(oldEntries))
		copy(s, oldEntries)
	} else {
		s = make([]stringSetEntry, len(oldEntries)+1)
		copy(s, oldEntries[:i])
		copy(s[i+1:], oldEntries[i:])
	}
	s[i] = stringSetEntry{key: key, Value: value}
	return s
}

// Delete the given set from the Sets.
func (s Sets) Delete(key string) Sets {
	i := s.locate(key)
	if i == len(s) || s[i].key != key {
		return s // not found
	}
	result := make([]stringSetEntry, len(s)-1)
	if i > 0 {
		copy(result, s[:i-1])
	}
	copy(result[i:], s[i+1:])
	return result
}

// Lookup returns the sets stored under key.
func (s Sets) Lookup(key string) (StringSet, bool) {
	i := s.locate(key)
	if i < len(s) && s[i].key == key {
		return s[i].Value, true
	}
	return MakeStringSet(), false
}

// Size returns the number of elements
func (s Sets) Size() int {
	return len(s)
}

// Merge merges two sets maps into a fresh set, performing set-union merges as
// appropriate.
func (m Sets) Merge(n Sets) Sets {
	switch {
	case m == nil:
		return n
	case n == nil:
		return m
	}
	l := len(m)
	if len(n) > l {
		l = len(n)
	}
	out := make([]stringSetEntry, 0, l+1)

	i, j := 0, 0
	for i < len(m) {
		switch {
		case j >= len(n) || m[i].key < n[j].key:
			out = append(out, m[i])
			i++
		case m[i].key == n[j].key:
			newValue := m[i].Value.Merge(n[j].Value)
			out = append(out, stringSetEntry{key: m[i].key, Value: newValue})
			i++
			j++
		default:
			out = append(out, n[j])
			j++
		}
	}
	for ; j < len(n); j++ {
		out = append(out, n[j])
	}
	return out
}

func (s Sets) String() string {
	buf := bytes.NewBufferString("{")
	for _, val := range s {
		fmt.Fprintf(buf, "%s: %s,\n", val.key, val.Value)
	}
	fmt.Fprintf(buf, "}")
	return buf.String()
}

// DeepEqual tests equality with other Sets
func (s Sets) DeepEqual(t Sets) bool {
	if s.Size() != t.Size() {
		return false
	}
	for i := range s {
		if !reflect.DeepEqual(s[i], t[i]) {
			return false
		}
	}
	return true
}

// CodecEncodeSelf implements codec.Selfer
// Note this uses undocumented, internal APIs, which could break
// in the future.  See https://github.com/weaveworks/scope/pull/1709
// for more information.
func (m Sets) CodecEncodeSelf(encoder *codec.Encoder) {
	z, r := codec.GenHelperEncoder(encoder)
	if m == nil {
		r.EncodeNil()
		return
	}
	r.EncodeMapStart(m.Size())
	for _, val := range m {
		z.EncSendContainerState(containerMapKey)
		r.EncodeString(cUTF8, val.key)
		z.EncSendContainerState(containerMapValue)
		val.Value.CodecEncodeSelf(encoder)
	}
	z.EncSendContainerState(containerMapEnd)
}

// CodecDecodeSelf implements codec.Selfer
// Uses undocumented, internal APIs as for CodecEncodeSelf.
func (m *Sets) CodecDecodeSelf(decoder *codec.Decoder) {
	*m = nil
	z, r := codec.GenHelperDecoder(decoder)
	if r.TryDecodeAsNil() {
		return
	}

	length := r.ReadMapStart()
	if length > 0 {
		*m = make([]stringSetEntry, 0, length)
	}
	for i := 0; length < 0 || i < length; i++ {
		if length < 0 && r.CheckBreak() {
			break
		}
		z.DecSendContainerState(containerMapKey)
		var key string
		if !r.TryDecodeAsNil() {
			key = r.DecodeString()
		}
		j := m.locate(key)
		if j == len(*m) || (*m)[j].key != key {
			*m = append(*m, stringSetEntry{})
			copy((*m)[j+1:], (*m)[j:])
			(*m)[j] = stringSetEntry{}
		}
		(*m)[j].key = key
		z.DecSendContainerState(containerMapValue)
		if !r.TryDecodeAsNil() {
			(*m)[j].Value.CodecDecodeSelf(decoder)
		}
	}
	z.DecSendContainerState(containerMapEnd)
}

// MarshalJSON shouldn't be used, use CodecEncodeSelf instead
func (Sets) MarshalJSON() ([]byte, error) {
	panic("MarshalJSON shouldn't be used, use CodecEncodeSelf instead")
}

// UnmarshalJSON shouldn't be used, use CodecDecodeSelf instead
func (*Sets) UnmarshalJSON(b []byte) error {
	panic("UnmarshalJSON shouldn't be used, use CodecDecodeSelf instead")
}
