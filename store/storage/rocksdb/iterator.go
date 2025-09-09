package rocksdb

import (
	"bytes"

	"github.com/linxGnu/grocksdb"
)

// rocksDBIterator implements the Iterator interface for RocksDB
type rocksDBIterator struct {
	iter      *grocksdb.Iterator
	start     []byte
	end       []byte
	keyLength int // Length adjustment for versioned keys
	valid     bool
	closed    bool
}

func newRocksDBIterator(iter *grocksdb.Iterator, start, end []byte, keyLength int) *rocksDBIterator {
	iterator := &rocksDBIterator{
		iter:      iter,
		start:     start,
		end:       end,
		keyLength: keyLength,
		closed:    false,
	}

	// Seek to start position
	if start != nil {
		iter.Seek(start)
	} else {
		iter.SeekToFirst()
	}

	iterator.updateValid()
	return iterator
}

func newRocksDBReverseIterator(iter *grocksdb.Iterator, start, end []byte, keyLength int) *rocksDBIterator {
	iterator := &rocksDBIterator{
		iter:      iter,
		start:     start,
		end:       end,
		keyLength: keyLength,
		closed:    false,
	}

	// Seek to end position for reverse iteration
	if end != nil {
		iter.Seek(end)
		if iter.Valid() {
			// Move to previous since Seek is inclusive and we want exclusive end
			iter.Prev()
		} else {
			// If end doesn't exist, seek to last
			iter.SeekToLast()
		}
	} else {
		iter.SeekToLast()
	}

	iterator.updateValid()
	return iterator
}

func (it *rocksDBIterator) Domain() (start, end []byte) {
	return it.start, it.end
}

func (it *rocksDBIterator) Valid() bool {
	return it.valid && !it.closed
}

func (it *rocksDBIterator) Next() {
	if it.closed {
		return
	}

	it.iter.Next()
	it.updateValid()
}

func (it *rocksDBIterator) Key() []byte {
	if !it.Valid() {
		return nil
	}

	key := it.iter.Key()
	defer key.Free()

	keyData := key.Data()

	// Remove version suffix if keyLength is negative
	if it.keyLength < 0 {
		suffixLen := -it.keyLength
		if len(keyData) >= suffixLen {
			keyData = keyData[:len(keyData)-suffixLen]
		}
	}

	return keyData
}

func (it *rocksDBIterator) Value() []byte {
	if !it.Valid() {
		return nil
	}

	value := it.iter.Value()
	defer value.Free()

	return value.Data()
}

func (it *rocksDBIterator) Close() error {
	if it.closed {
		return nil
	}

	it.iter.Close()
	it.closed = true
	it.valid = false
	return nil
}

func (it *rocksDBIterator) updateValid() {
	if it.closed {
		it.valid = false
		return
	}

	if !it.iter.Valid() {
		it.valid = false
		return
	}

	// Check if current key is within domain
	key := it.iter.Key()
	defer key.Free()

	keyData := key.Data()

	// Check start boundary
	if it.start != nil && bytes.Compare(keyData, it.start) < 0 {
		it.valid = false
		return
	}

	// Check end boundary
	if it.end != nil && bytes.Compare(keyData, it.end) >= 0 {
		it.valid = false
		return
	}

	it.valid = true
}

// rocksDBReverseIterator is a specialized reverse iterator
type rocksDBReverseIterator struct {
	*rocksDBIterator
}

func (it *rocksDBReverseIterator) Next() {
	if it.closed {
		return
	}

	it.iter.Prev()
	it.updateValid()
}
