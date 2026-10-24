package rewind

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"sort"
)

// A minimal WebM muxer for VP8 and Opus. Each segment file is a complete
// WebM: header, tracks and clusters. Timecodes are milliseconds on the
// timeline of the room's buffer, not from zero, so segments line up when
// a player appends them one after another.

const (
	idEBML               = 0x1A45DFA3
	idEBMLVersion        = 0x4286
	idEBMLReadVersion    = 0x42F7
	idEBMLMaxIDLength    = 0x42F2
	idEBMLMaxSizeLength  = 0x42F3
	idDocType            = 0x4282
	idDocTypeVersion     = 0x4287
	idDocTypeReadVersion = 0x4285
	idSegment            = 0x18538067
	idInfo               = 0x1549A966
	idTimecodeScale      = 0x2AD7B1
	idMuxingApp          = 0x4D80
	idWritingApp         = 0x5741
	idTracks             = 0x1654AE6B
	idTrackEntry         = 0xAE
	idTrackNumber        = 0xD7
	idTrackUID           = 0x73C5
	idTrackType          = 0x83
	idCodecID            = 0x86
	idCodecPrivate       = 0x63A2
	idSeekPreRoll        = 0x56BB
	idVideo              = 0xE0
	idPixelWidth         = 0xB0
	idPixelHeight        = 0xBA
	idAudio              = 0xE1
	idSamplingFrequency  = 0xB5
	idChannels           = 0x9F
	idCluster            = 0x1F43B675
	idTimecode           = 0xE7
	idSimpleBlock        = 0xA3
)

const (
	trackVideo = 1
	trackAudio = 2
)

// frame is one encoded video frame or audio packet
type frame struct {
	video bool
	key   bool
	ts    int64 // ms on the buffer timeline
	data  []byte
}

func putID(b *bytes.Buffer, id uint32) {
	switch {
	case id >= 1<<24:
		b.Write([]byte{byte(id >> 24), byte(id >> 16), byte(id >> 8), byte(id)})
	case id >= 1<<16:
		b.Write([]byte{byte(id >> 16), byte(id >> 8), byte(id)})
	case id >= 1<<8:
		b.Write([]byte{byte(id >> 8), byte(id)})
	default:
		b.WriteByte(byte(id))
	}
}

// putSize writes an element size as the shortest EBML variable size integer
func putSize(b *bytes.Buffer, n uint64) {
	l := 1
	for l < 8 && n >= (1<<(7*l))-1 {
		l++
	}
	for i := l - 1; i >= 0; i-- {
		v := byte(n >> (8 * i))
		if i == l-1 {
			v |= 0x80 >> (l - 1)
		}
		b.WriteByte(v)
	}
}

func element(b *bytes.Buffer, id uint32, data []byte) {
	putID(b, id)
	putSize(b, uint64(len(data)))
	b.Write(data)
}

func elementUint(b *bytes.Buffer, id uint32, v uint64) {
	var d []byte
	for v > 0 || len(d) == 0 {
		d = append([]byte{byte(v)}, d...)
		v >>= 8
	}
	element(b, id, d)
}

func elementFloat(b *bytes.Buffer, id uint32, v float64) {
	var d [8]byte
	binary.BigEndian.PutUint64(d[:], math.Float64bits(v))
	element(b, id, d[:])
}

func master(b *bytes.Buffer, id uint32, f func(*bytes.Buffer)) {
	var inner bytes.Buffer
	f(&inner)
	element(b, id, inner.Bytes())
}

// opusHead is the Opus identification header, the codec private data
func opusHead() []byte {
	h := []byte("OpusHead")
	h = append(h, 1, 2) // version, channels
	// no pre-skip: the stream is cut from a running encoder, and each segment
	// would otherwise lose its first few milliseconds
	h = binary.LittleEndian.AppendUint16(h, 0)
	h = binary.LittleEndian.AppendUint32(h, 48000)
	h = append(h, 0, 0, 0) // gain, mapping family
	return h
}

// writeHeader writes the EBML header, the start of a live segment and the
// tracks
func writeHeader(w io.Writer, width, height int) error {
	var b bytes.Buffer
	master(&b, idEBML, func(b *bytes.Buffer) {
		elementUint(b, idEBMLVersion, 1)
		elementUint(b, idEBMLReadVersion, 1)
		elementUint(b, idEBMLMaxIDLength, 4)
		elementUint(b, idEBMLMaxSizeLength, 8)
		element(b, idDocType, []byte("webm"))
		elementUint(b, idDocTypeVersion, 4)
		elementUint(b, idDocTypeReadVersion, 2)
	})
	// a segment of unknown size, clusters follow until the file ends
	putID(&b, idSegment)
	b.Write([]byte{0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})
	master(&b, idInfo, func(b *bytes.Buffer) {
		elementUint(b, idTimecodeScale, 1000000) // ms
		element(b, idMuxingApp, []byte("neko-rooms"))
		element(b, idWritingApp, []byte("neko-rooms"))
	})
	master(&b, idTracks, func(b *bytes.Buffer) {
		master(b, idTrackEntry, func(b *bytes.Buffer) {
			elementUint(b, idTrackNumber, trackVideo)
			elementUint(b, idTrackUID, trackVideo)
			elementUint(b, idTrackType, 1)
			element(b, idCodecID, []byte("V_VP8"))
			master(b, idVideo, func(b *bytes.Buffer) {
				elementUint(b, idPixelWidth, uint64(width))
				elementUint(b, idPixelHeight, uint64(height))
			})
		})
		master(b, idTrackEntry, func(b *bytes.Buffer) {
			elementUint(b, idTrackNumber, trackAudio)
			elementUint(b, idTrackUID, trackAudio)
			elementUint(b, idTrackType, 2)
			element(b, idCodecID, []byte("A_OPUS"))
			element(b, idCodecPrivate, opusHead())
			elementUint(b, idSeekPreRoll, 80000000) // 80ms
			master(b, idAudio, func(b *bytes.Buffer) {
				elementFloat(b, idSamplingFrequency, 48000)
				elementUint(b, idChannels, 2)
			})
		})
	})
	_, err := w.Write(b.Bytes())
	return err
}

// maxClusterSpan keeps block timecodes, relative to their cluster, in int16
const maxClusterSpan = 30000

// writeCluster writes frames as one cluster, ordered by time
func writeCluster(w io.Writer, frames []frame) error {
	if len(frames) == 0 {
		return nil
	}
	sort.SliceStable(frames, func(i, j int) bool { return frames[i].ts < frames[j].ts })
	tc := frames[0].ts

	var b bytes.Buffer
	master(&b, idCluster, func(b *bytes.Buffer) {
		elementUint(b, idTimecode, uint64(tc))
		for _, f := range frames {
			track := byte(trackAudio)
			if f.video {
				track = trackVideo
			}
			rel := f.ts - tc
			if rel > math.MaxInt16 {
				rel = math.MaxInt16
			}
			flags := byte(0)
			if f.key || !f.video {
				flags = 0x80
			}
			putID(b, idSimpleBlock)
			putSize(b, uint64(4+len(f.data)))
			b.Write([]byte{0x80 | track, byte(uint16(rel) >> 8), byte(uint16(rel)), flags})
			b.Write(f.data)
		}
	})
	_, err := w.Write(b.Bytes())
	return err
}

// vp8Key says whether a VP8 frame is a keyframe and its size
func vp8Key(data []byte) (key bool, width, height int) {
	if len(data) < 10 || data[0]&0x01 != 0 {
		return false, 0, 0
	}
	// keyframes have a start code after the 3 byte frame tag
	if data[3] != 0x9d || data[4] != 0x01 || data[5] != 0x2a {
		return false, 0, 0
	}
	width = int(binary.LittleEndian.Uint16(data[6:8]) & 0x3fff)
	height = int(binary.LittleEndian.Uint16(data[8:10]) & 0x3fff)
	return true, width, height
}
