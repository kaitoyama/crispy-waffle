package domain

import (
	"crypto/rand"
	"encoding/binary"
	"sync"
	"time"
)

// Typed identifiers. All IDs are lexicographically sortable, time-prefixed
// strings (ULID-like) so that "newest" ordering falls out of string ordering.
type (
	TaskID     string
	RunID      string
	StepRunID  string
	EventID    string
	ApprovalID string
	ActorID    string
	LinkID     string
)

const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

var (
	entMu   sync.Mutex
	lastMS  uint64
	lastEnt [10]byte
)

// newULID generates a 26-char Crockford base32 ULID. Monotonic within the same
// millisecond so IDs created in a tight loop still sort by creation order.
func newULID() string {
	entMu.Lock()
	defer entMu.Unlock()

	ms := uint64(time.Now().UnixMilli())
	if ms == lastMS {
		// same millisecond: increment the entropy to preserve order
		for i := 9; i >= 0; i-- {
			lastEnt[i]++
			if lastEnt[i] != 0 {
				break
			}
		}
	} else {
		lastMS = ms
		_, _ = rand.Read(lastEnt[:])
	}

	var b [16]byte
	binary.BigEndian.PutUint64(b[0:8], ms<<16) // 48 bits of time in the top
	copy(b[6:16], lastEnt[:])

	// encode 128 bits -> 26 crockford chars (last char carries 2 bits)
	out := make([]byte, 26)
	out[0] = crockford[(b[0]&0xE0)>>5]
	out[1] = crockford[b[0]&0x1F]
	idx := 2
	var acc uint16
	var bits uint8
	for i := 1; i < 16; i++ {
		acc = acc<<8 | uint16(b[i])
		bits += 8
		for bits >= 5 {
			bits -= 5
			out[idx] = crockford[(acc>>bits)&0x1F]
			idx++
		}
	}
	if bits > 0 {
		out[idx] = crockford[(acc<<(5-bits))&0x1F]
		idx++
	}
	return string(out[:26])
}

func NewTaskID() TaskID         { return TaskID("tsk_" + newULID()) }
func NewRunID() RunID           { return RunID("run_" + newULID()) }
func NewStepRunID() StepRunID   { return StepRunID("srn_" + newULID()) }
func NewEventID() EventID       { return EventID("evt_" + newULID()) }
func NewApprovalID() ApprovalID { return ApprovalID("apr_" + newULID()) }
func NewLinkID() LinkID         { return LinkID("lnk_" + newULID()) }
