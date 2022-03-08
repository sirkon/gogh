package blocks

import (
	"bytes"
	"sort"
	"strconv"
	"strings"
)

// New creates new block
func New() *Blocks {
	buf := &bytes.Buffer{}
	return &Blocks{
		clock: []int{0},
		data: map[string]*bytes.Buffer{
			"0": buf,
		},
		cur: buf,
	}
}

// Blocks an object to deal with blocks
type Blocks struct {
	clock []int
	data  map[string]*bytes.Buffer
	cur   *bytes.Buffer
}

// Next extends blocks with a new one
func (b *Blocks) Next() *Blocks {
	newClock := make([]int, len(b.clock))
	copy(newClock, b.clock)
	newClock[len(newClock)-1]++
	b.clock = append(b.clock, 0)
	res := &Blocks{
		clock: newClock,
		data:  b.data,
	}
	var buf bytes.Buffer
	res.cur = &buf
	b.data[res.clockValue()] = res.cur

	return res
}

// Data returns current buffer
func (b *Blocks) Data() *bytes.Buffer {
	return b.cur
}

// Collect returns an ordered sequence of blocks
func (b *Blocks) Collect() []*bytes.Buffer {
	var keys []string
	for k := range b.data {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		a := strings.Split(keys[i], ".")
		b := strings.Split(keys[j], ".")

		for i := 0; i < len(a) && i < len(b); i++ {
			va, _ := strconv.Atoi(a[i])
			vb, _ := strconv.Atoi(b[i])
			switch {
			case va < vb:
				return true
			case va > vb:
				return false
			}
		}

		switch {
		case len(a) < len(b):
			return true
		default:
			return false
		}
	})

	var res []*bytes.Buffer
	for _, key := range keys {
		res = append(res, b.data[key])
	}

	return res
}

func (b *Blocks) clockValue() string {
	var parts []string
	for _, v := range b.clock {
		parts = append(parts, strconv.Itoa(v))
	}
	return strings.Join(parts, ".")
}
