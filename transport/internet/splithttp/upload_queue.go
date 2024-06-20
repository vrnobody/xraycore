package splithttp

// upload_queue is a specialized priorityqueue + channel to reorder generic
// packets by a sequence number

import (
	"io"
	"sync"
	"time"
)

type UploadQueue struct {
	readSignalSize int
	readSignal     chan struct{}

	writeMu      *sync.Mutex
	writeSignal  *CondWithTimeout
	writeTimeout time.Duration

	buffers    [][]byte
	bufferSize uint64
	cache      []byte

	seq    uint64
	closed bool
}

func NewUploadQueue(size int) *UploadQueue {
	writeMutex := sync.Mutex{}
	bufferSize := uint64(2 * size)

	return &UploadQueue{
		readSignalSize: 3,
		readSignal:     make(chan struct{}, 2*3),

		writeMu:      &writeMutex,
		writeSignal:  NewCondWithTimeout(&writeMutex),
		writeTimeout: 10 * time.Second,

		bufferSize: bufferSize,
		buffers:    make([][]byte, bufferSize),
		cache:      nil,

		seq:    0,
		closed: false,
	}
}

func (h *UploadQueue) Push(seq uint64, payload []byte) error {
	// notify reader
	defer h.sendReadSignal()

	// save packet to buffer
	idx := seq % h.bufferSize
	h.buffers[idx] = payload
	return nil
}

// Wait until buffer is available
func (h *UploadQueue) Wait(seq uint64) error {
	// fast path
	if seq < h.seq+h.bufferSize {
		if h.closed {
			return newError("splithttp packet queue closed")
		}
		return nil
	}

	// slow path
	h.writeMu.Lock()
	defer h.writeMu.Unlock()

	recvSignal := true
	// h.seq could be changed
	for recvSignal && !h.closed && seq >= h.seq+h.bufferSize {
		recvSignal = h.writeSignal.WaitWithTimeout(h.writeTimeout)
	}

	if h.closed {
		return newError("splithttp packet queue closed")
	}

	if !recvSignal {
		return newError("splithttp wait timeout")
	}
	return nil
}

func (h *UploadQueue) Close() error {
	h.closed = true
	h.writeSignal.Broadcast()
	h.sendReadSignal()
	return nil
}

func (h *UploadQueue) sendReadSignal() {
	// non-blocking (kind of)
	if len(h.readSignal) < h.readSignalSize {
		h.readSignal <- struct{}{}
	}
}

func (h *UploadQueue) Read(b []byte) (int, error) {
	for {
		// try to read from cache
		if l := len(h.cache); l > 0 {
			n := copy(b, h.cache)
			if n < l {
				h.cache = h.cache[n:]
			} else {
				h.cache = nil
			}
			return n, nil
		}

		// try to load from buffer
		idx := h.seq % h.bufferSize
		if p := h.buffers[idx]; p != nil {
			h.cache = p
			h.buffers[idx] = nil
			h.seq++
			h.writeSignal.Broadcast()
		} else if h.closed {
			return 0, io.EOF
		} else {
			// await writer
			<-h.readSignal
		}
	}
}
