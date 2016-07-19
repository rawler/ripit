package main

// http://l.web.umkc.edu/lizhu/teaching/2016sp.video-communication/ref/mp4.pdf

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
)

var MP4Endian = binary.BigEndian

func Read(r io.Reader, tgt interface{}) error {
	return binary.Read(r, MP4Endian, tgt)
}

type FourCC [4]byte

var (
	FOURCC_CO64 = FourCC{'c', 'o', '6', '4'}
	FOURCC_HDLR = FourCC{'h', 'd', 'l', 'r'}
	FOURCC_MDIA = FourCC{'m', 'd', 'i', 'a'}
	FOURCC_MINF = FourCC{'m', 'i', 'n', 'f'}
	FOURCC_MOOV = FourCC{'m', 'o', 'o', 'v'}
	FOURCC_STBL = FourCC{'s', 't', 'b', 'l'}
	FOURCC_STCO = FourCC{'s', 't', 'c', 'o'}
	FOURCC_STSC = FourCC{'s', 't', 's', 'c'}
	FOURCC_STSD = FourCC{'s', 't', 's', 'd'}
	FOURCC_STSZ = FourCC{'s', 't', 's', 'z'}
	FOURCC_STTS = FourCC{'s', 't', 't', 's'}
	FOURCC_STZ2 = FourCC{'s', 't', 'z', '2'}
	FOURCC_TRAK = FourCC{'t', 'r', 'a', 'k'}
)

func (x FourCC) String() string {
	return string(x[:])
}

type MP4BoxHeader struct {
	Size   uint64
	FourCC FourCC
}

const MP4BoxHeaderBytes = 8

func (h MP4BoxHeader) PayloadSize() int64 {
	return int64(h.Size - MP4BoxHeaderBytes)
}

func ParseBoxHeader(r io.Reader) (res MP4BoxHeader, err error) {
	var simpleSize uint32
	if err = Read(r, &simpleSize); err != nil {
		return res, err
	}
	if err = Read(r, &res.FourCC); err != nil {
		return res, err
	}
	if simpleSize == 1 {
		err = Read(r, &res.Size)
	} else {
		res.Size = uint64(simpleSize)
	}
	return res, err
}

// ScanBoxes scans the inside of an MP4 Box, for inner boxes.
//
// Boxes are given to f(header, payload) for inspection, which can yield io.EOF to end
// iteration, or some other error which will simply propagate up
func ScanBoxes(r io.ReadSeeker, f func(header MP4BoxHeader, payload io.ReadSeeker) error) error {
	for {
		header, err := ParseBoxHeader(r)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("Header Parse Error: %s", err)
		}

		var payload io.ReadSeeker
		payload, err = LimitReadSeeker(r, header.PayloadSize())
		if err != nil {
			return err
		}
		err = f(header, payload)

		// Silently skip remaining inner box.
		pos, err := payload.Seek(0, SEEK_END)
		if err != nil {
			fmt.Errorf("Seek To end of Box (%d) failed: %s", pos, err)
		}

		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

type FullBox struct {
	Version byte
	Flags   [3]byte
}

type TrackType FourCC

var (
	TrackTypeSound = TrackType{'s', 'o', 'u', 'n'}
	TrackTypeVideo = TrackType{'v', 'i', 'd', 'e'}
	TrackTypeHint  = TrackType{'h', 'i', 'n', 't'}
)

func (t TrackType) String() string {
	return string(t[:])
}

type HandlerBox struct {
	FullBox
	Predefined  uint32
	HandlerType TrackType
	Reserved    [3]uint32
}

func ParseHandler(r io.Reader) (res HandlerBox, err error) {
	err = Read(r, &res)
	return res, err
}

type SimpleTableHeader struct {
	FullBox
	EntryCount uint32
}

type SampleDescriptionBox struct {
	SimpleTableHeader
	Entries []MP4BoxHeader // Highly simplified
}

func (sd SampleDescriptionBox) MatchesTypes(t []FourCC) bool {
	for _, entry := range sd.Entries {
		for _, m := range t {
			if entry.FourCC == m {
				return true
			}
		}
		log.Printf("Track of wrong type: %s", entry.FourCC)
	}
	return false
}

func ParseSampleDescriptionBox(r io.Reader) (res SampleDescriptionBox, err error) {
	err = Read(r, &res.SimpleTableHeader)
	if err != nil {
		return res, err
	}

	for {
		header, err := ParseBoxHeader(r)
		if err == io.EOF {
			return res, nil
		}
		if err != nil {
			return res, err
		}
		res.Entries = append(res.Entries, header)
		_, err = io.CopyN(ioutil.Discard, r, header.PayloadSize())
		if err != nil {
			return res, err
		}
	}
}

type SampleSizeTable struct {
	SampleSizeTableHeader
	Entries []uint32
}

type SampleSizeTableHeader struct {
	FullBox
	ConstantSize uint32
	EntryCount   uint32
}

func (t SampleSizeTable) SampleSize(n int) uint32 {
	if t.ConstantSize != 0 {
		return t.ConstantSize
	}
	return t.Entries[n]
}

func (t SampleSizeTable) SampleSizes(n int, count int) uint64 {
	if t.ConstantSize != 0 {
		return uint64(t.ConstantSize) * uint64(count)
	}
	res := uint64(0)
	for i := n; i < (n + count); i++ {
		res += uint64(t.SampleSize(i))
	}
	return res
}

func ParseSampleSizeBox(r io.Reader) (res SampleSizeTable, err error) {
	err = Read(r, &res.SampleSizeTableHeader)
	if err != nil {
		return res, err
	}
	if res.ConstantSize == 0 {
		res.Entries = make([]uint32, res.EntryCount)
	}
	return res, Read(r, res.Entries)
}

type SampleToChunkTable struct {
	SimpleTableHeader
	Entries []SampleToChunkEntry
}

type SampleToChunkEntry struct {
	FirstChunk             uint32
	SamplesPerChunk        uint32
	SampleDescriptionIndex uint32
}

func ParseSampleToChunkBox(r io.Reader) (res SampleToChunkTable, err error) {
	err = Read(r, &res.SimpleTableHeader)
	if err != nil {
		return res, err
	}
	res.Entries = make([]SampleToChunkEntry, res.EntryCount)
	return res, Read(r, res.Entries)
}

func (t SampleToChunkTable) ChunkSamples(chunkIndex int) int {
	tableIndex := len(t.Entries) - 1
	limit := uint32(chunkIndex + 1)
	for ; t.Entries[tableIndex].FirstChunk > limit; tableIndex-- {
	}
	return int(t.Entries[tableIndex].SamplesPerChunk)
}

// ChunkOffsetTable is a unifying interface for the two versions
type ChunkOffsetTable interface {
	Count() int
	Offset(n int) uint64
}

type ChunkOffset32Table struct {
	Entries []uint32
}

func (t ChunkOffset32Table) Count() int {
	return len(t.Entries)
}
func (t ChunkOffset32Table) Offset(n int) uint64 {
	return uint64(t.Entries[n])
}

type ChunkOffset64Table struct {
	Entries []uint64
}

func (t ChunkOffset64Table) Count() int {
	return len(t.Entries)
}
func (t ChunkOffset64Table) Offset(n int) uint64 {
	return t.Entries[n]
}

func ParseChunkOffsetBox(r io.Reader, fourCC FourCC) (res ChunkOffsetTable, err error) {
	var h SimpleTableHeader
	err = Read(r, &h)
	if err != nil {
		return nil, err
	}

	switch fourCC {
	case FOURCC_STCO:
		res := ChunkOffset32Table{
			Entries: make([]uint32, h.EntryCount),
		}
		return res, Read(r, res.Entries)
	case FOURCC_CO64:
		res := ChunkOffset64Table{
			Entries: make([]uint64, h.EntryCount),
		}
		return res, Read(r, res.Entries)
	}
	return nil, fmt.Errorf("Unsupported Four-CC for ChunkOffset: %s", fourCC)
}
