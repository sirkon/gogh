package blocks

import "bytes"

// New creates the first block of the sequence.
func New() *Manager {
	return &Manager{
		cur: &block{
			prev: nil,
			next: nil,
			data: new(bytes.Buffer),
		},
	}
}

// Manager holds a "block" node from a double linked list of them.
type Manager struct {
	cur *block
}

type block struct {
	prev *block
	next *block
	data *bytes.Buffer
}

// Insert creates and inserts a new block and moves the
// manager to it. Returns itself.
func (b *Manager) Insert() *Manager {
	cur := b.cur
	newBlock := &block{
		prev: cur,
		next: cur.next,
		data: new(bytes.Buffer),
	}
	if newBlock.next != nil {
		newBlock.next.prev = newBlock
	}
	cur.next = newBlock
	b.cur = newBlock

	return b
}

// Prev returns a manager pointing to a previous block.
func (b *Manager) Prev() *Manager {
	return &Manager{
		cur: b.cur.prev,
	}
}

// Data returns current buffer
func (b *Manager) Data() *bytes.Buffer {
	return b.cur.data
}

// Collect returns an ordered sequence of block
func (b *Manager) Collect() []*bytes.Buffer {
	var res []*bytes.Buffer

	// Go to the first block
	cur := b.cur
	for cur.prev != nil {
		cur = cur.prev
	}

	// And collect all text buffers going through the next field
	for cur != nil {
		res = append(res, cur.data)
		cur = cur.next
	}

	return res
}
