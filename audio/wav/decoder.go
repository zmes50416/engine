// Copyright 2014 The Azul3D Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wav

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"sync"

	"azul3d.org/engine/audio"
)

const (
	// Data format codes

	// PCM
	wave_FORMAT_PCM = 0x0001

	// IEEE float
	wave_FORMAT_IEEE_FLOAT = 0x0003

	// 8-bit ITU-T G.711 A-law
	wave_FORMAT_ALAW = 0x0006

	// 8-bit ITU-T G.711 µ-law
	wave_FORMAT_MULAW = 0x0007

	// Determined by SubFormat
	wave_FORMAT_EXTENSIBLE = 0xFFFE
)

type decoder struct {
	access sync.RWMutex

	format, bitsPerSample   uint16
	chunkSize, currentCount uint32
	dataChunkBegin          int32

	r        interface{}
	rd       io.Reader
	smallBuf []byte // Buffer used for small reads.
	config   *audio.Config
}

// advance advances the byte counter by sz. If the chunk size is known and
// after advancement the byte counter is larger than the chunk size, then
// audio.EOS is returned.
//
// If the chunk size is not known, the data chunk marker is extended by sz as
// well.
func (d *decoder) advance(sz int) error {
	if d.chunkSize > 0 {
		d.currentCount += uint32(sz)
		if d.currentCount > d.chunkSize {
			return audio.EOS
		}
	} else {
		d.dataChunkBegin += int32(sz)
	}
	return nil
}

func (d *decoder) bRead(data interface{}, sz int) error {
	err := d.advance(sz)
	if err != nil {
		return err
	}
	return binary.Read(d.rd, binary.LittleEndian, data)
}

// smallRead performs a small read of N bytes from the decoder's reader. It is
// said to be a small read because the buffer does not shrink.
func (d *decoder) smallRead(n int) ([]byte, error) {
	if len(d.smallBuf) < n {
		d.smallBuf = make([]byte, n)
	}
	_, err := io.ReadFull(d.rd, d.smallBuf[:n])
	return d.smallBuf[:n], err
}

// Reads and returns the next RIFF chunk, note that always len(ident) == 4
// E.g.
//
//  "fmt " (notice space).
//
// Length is length of chunk data.
//
// Returns any read errors.
func (d *decoder) nextChunk() (ident string, length uint32, err error) {
	// Read chunk identity, like "RIFF" or "fmt "
	var chunkIdent [4]byte
	err = d.bRead(&chunkIdent, binary.Size(chunkIdent))
	if err != nil {
		return "", 0, err
	}
	ident = string(chunkIdent[:])

	// Read chunk length
	err = d.bRead(&length, binary.Size(length))
	if err != nil {
		return "", 0, err
	}
	return
}

func (d *decoder) Seek(sample uint64) error {
	rs, ok := d.r.(io.ReadSeeker)
	if ok {
		offset := int64(sample * (uint64(d.bitsPerSample) / 8))
		_, err := rs.Seek(int64(d.dataChunkBegin)+offset, 0)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *decoder) readUint8(b audio.Slice) (read int, err error) {
	bb, bbOk := b.(audio.Uint8)

	var (
		sample uint8
		length = b.Len()
		buf    []byte
	)
	for read = 0; read < length; read++ {
		// Advance the reader.
		err = d.advance(1) // 1 == binary.Size(sample)
		if err != nil {
			return
		}

		// Pull one sample from the reader.
		buf, err = d.smallRead(1) // 1 == binary.Size(sample)
		if err != nil {
			return
		}
		sample = buf[0]

		if bbOk {
			bb[read] = sample
		} else {
			f64 := audio.Uint8ToFloat64(sample)
			b.Set(read, f64)
		}
	}

	return
}

func (d *decoder) readInt16(b audio.Slice) (read int, err error) {
	bb, bbOk := b.(audio.Int16)

	var (
		sample int16
		length = b.Len()
		buf    []byte
	)
	for read = 0; read < length; read++ {
		// Advance the reader.
		err = d.advance(2) // 2 == binary.Size(sample)
		if err != nil {
			return
		}

		// Pull one sample from the reader.
		buf, err = d.smallRead(2) // 2 == binary.Size(sample)
		if err != nil {
			return
		}
		sample = int16(binary.LittleEndian.Uint16(buf))

		if bbOk {
			bb[read] = sample
		} else {
			f64 := audio.Int16ToFloat64(sample)
			b.Set(read, f64)
		}
	}

	return
}

func (d *decoder) readInt24(b audio.Slice) (read int, err error) {
	bb, bbOk := b.(audio.Int32)

	var (
		sample []byte
		length = b.Len()
	)
	for read = 0; read < length; read++ {
		// Advance the reader.
		err = d.advance(3) // 3 == binary.Size(sample)
		if err != nil {
			return
		}

		// Pull one sample from the reader.
		sample, err = d.smallRead(3) // 3 == binary.Size(sample)
		if err != nil {
			return
		}

		var ss int32
		ss = int32(sample[0]) | int32(sample[1])<<8 | int32(sample[2])<<16
		if (ss & 0x800000) > 0 {
			ss |= ^0xffffff
		}

		if bbOk {
			bb[read] = ss
		} else {
			f64 := audio.Int32ToFloat64(ss)
			b.Set(read, f64)
		}
	}

	return
}

func (d *decoder) readInt32(b audio.Slice) (read int, err error) {
	bb, bbOk := b.(audio.Int32)

	var (
		sample int32
		length = b.Len()
		buf    []byte
	)
	for read = 0; read < length; read++ {
		// Advance the reader.
		err = d.advance(4) // 4 == binary.Size(sample)
		if err != nil {
			return
		}

		// Pull one sample from the reader.
		buf, err = d.smallRead(4) // 4 == binary.Size(sample)
		if err != nil {
			return
		}
		sample = int32(binary.LittleEndian.Uint32(buf))

		if bbOk {
			bb[read] = sample
		} else {
			f64 := audio.Int32ToFloat64(sample)
			b.Set(read, f64)
		}
	}

	return
}

func (d *decoder) readFloat32(b audio.Slice) (read int, err error) {
	bb, bbOk := b.(audio.Float32)

	var (
		sample uint32
		length = b.Len()
		buf    []byte
	)
	for read = 0; read < length; read++ {
		// Advance the reader.
		err = d.advance(4) // 4 == binary.Size(sample)
		if err != nil {
			return
		}

		// Pull one sample from the reader.
		buf, err = d.smallRead(4) // 4 == binary.Size(sample)
		if err != nil {
			return
		}
		sample = binary.LittleEndian.Uint32(buf)

		if bbOk {
			bb[read] = math.Float32frombits(sample)
		} else {
			b.Set(read, float64(math.Float32frombits(sample)))
		}
	}

	return
}

func (d *decoder) readFloat64(b audio.Slice) (read int, err error) {
	var (
		sample uint64
		length = b.Len()
		buf    []byte
	)
	for read = 0; read < length; read++ {
		// Advance the reader.
		err = d.advance(8) // 8 == binary.Size(sample)
		if err != nil {
			return
		}

		// Pull one sample from the reader.
		buf, err = d.smallRead(8) // 8 == binary.Size(sample)
		if err != nil {
			return
		}
		sample = binary.LittleEndian.Uint64(buf)

		b.Set(read, math.Float64frombits(sample))
	}

	return
}

func (d *decoder) readMuLaw(b audio.Slice) (read int, err error) {
	bb, bbOk := b.(audio.MuLaw)

	var (
		sample uint8
		length = b.Len()
		buf    []byte
	)
	for read = 0; read < length; read++ {
		// Advance the reader.
		err = d.advance(1) // 1 == binary.Size(sample)
		if err != nil {
			return
		}

		// Pull one sample from the reader.
		buf, err = d.smallRead(1) // 1 == binary.Size(sample)
		if err != nil {
			return
		}
		sample = buf[0]

		if bbOk {
			bb[read] = sample
		} else {
			p16 := audio.MuLawToInt16(sample)
			b.Set(read, audio.Int16ToFloat64(p16))
		}
	}

	return
}

func (d *decoder) readALaw(b audio.Slice) (read int, err error) {
	bb, bbOk := b.(audio.ALaw)

	var (
		sample uint8
		length = b.Len()
		buf    []byte
	)
	for read = 0; read < length; read++ {
		// Advance the reader.
		err = d.advance(1) // 1 == binary.Size(sample)
		if err != nil {
			return
		}

		// Pull one sample from the reader.
		buf, err = d.smallRead(1) // 1 == binary.Size(sample)
		if err != nil {
			return
		}
		sample = buf[0]

		if bbOk {
			bb[read] = sample
		} else {
			p16 := audio.ALawToInt16(sample)
			b.Set(read, audio.Int16ToFloat64(p16))
		}
	}

	return
}

func (d *decoder) Read(b audio.Slice) (read int, err error) {
	if b.Len() == 0 {
		return
	}

	d.access.Lock()
	defer d.access.Unlock()

	switch d.format {
	case wave_FORMAT_PCM:
		switch d.bitsPerSample {
		case 8:
			return d.readUint8(b)
		case 16:
			return d.readInt16(b)
		case 24:
			return d.readInt24(b)
		case 32:
			return d.readInt32(b)
		default:
			panic("invalid bits per sample")
		}

	case wave_FORMAT_IEEE_FLOAT:
		switch d.bitsPerSample {
		case 32:
			return d.readFloat32(b)
		case 64:
			return d.readFloat64(b)
		default:
			panic("invalid bits per sample")
		}

	case wave_FORMAT_MULAW:
		return d.readMuLaw(b)
	case wave_FORMAT_ALAW:
		return d.readALaw(b)
	default:
		panic("invalid format")
	}
	return
}

func (d *decoder) Config() audio.Config {
	d.access.RLock()
	defer d.access.RUnlock()

	return *d.config
}

// ErrUnsupported defines an error for decoding wav data that is valid (by the
// wave specification) but not supported by the decoder in this package.
//
// This error only happens for audio files containing extensible wav data.
var ErrUnsupported = errors.New("wav: data format is valid but not supported by decoder")

// NewDecoder returns a new initialized audio decoder for the io.Reader or
// io.ReadSeeker, r.
func newDecoder(r interface{}) (audio.Decoder, error) {
	d := new(decoder)
	d.r = r

	switch t := r.(type) {
	case io.Reader:
		d.rd = t
	case io.ReadSeeker:
		d.rd = io.Reader(t)
	default:
		panic("NewDecoder(): Invalid reader type; must be io.Reader or io.ReadSeeker!")
	}

	var (
		complete bool

		c16 fmtChunk16
		c18 fmtChunk18
		c40 fmtChunk40
	)
	for !complete {
		ident, length, err := d.nextChunk()
		if err != nil {
			return nil, err
		}

		switch ident {
		case "RIFF":
			var format [4]byte
			err = d.bRead(&format, binary.Size(format))
			if string(format[:]) != "WAVE" {
				return nil, audio.ErrInvalidData
			}

		case "fmt ":
			// Always contains the 16-byte chunk
			err = d.bRead(&c16, binary.Size(c16))
			if err != nil {
				return nil, err
			}
			d.bitsPerSample = c16.BitsPerSample

			// Sometimes contains extensive 18/40 total byte chunks
			switch length {
			case 18:
				err = d.bRead(&c18, binary.Size(c18))
				if err != nil {
					return nil, err
				}
			case 40:
				err = d.bRead(&c40, binary.Size(c40))
				if err != nil {
					return nil, err
				}
			}

			// Verify format tag
			ft := c16.FormatTag
			switch {
			case ft == wave_FORMAT_PCM && (d.bitsPerSample == 8 || d.bitsPerSample == 16 || d.bitsPerSample == 24 || d.bitsPerSample == 32):
				break
			case ft == wave_FORMAT_IEEE_FLOAT && (d.bitsPerSample == 32 || d.bitsPerSample == 64):
				break
			case ft == wave_FORMAT_ALAW && d.bitsPerSample == 8:
				break
			case ft == wave_FORMAT_MULAW && d.bitsPerSample == 8:
				break
			// We don't support extensible wav files
			//case wave_FORMAT_EXTENSIBLE:
			//	break
			default:
				return nil, ErrUnsupported
			}

			// Assign format tag for later (See Read() method)
			d.format = c16.FormatTag

			// We now have enough information to build the audio configuration
			d.config = &audio.Config{
				Channels:   int(c16.Channels),
				SampleRate: int(c16.SamplesPerSec),
			}

		case "fact":
			// We need to scan fact chunk first.
			var fact factChunk
			err = d.bRead(&fact, binary.Size(fact))
			if err != nil {
				return nil, err
			}

		case "data":
			// Read the data chunk header now
			d.chunkSize = length
			complete = true
		}
	}

	return d, nil
}

func init() {
	audio.RegisterFormat("wav", "RIFF", newDecoder)
}
