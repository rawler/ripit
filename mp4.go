package main

// http://l.web.umkc.edu/lizhu/teaching/2016sp.video-communication/ref/mp4.pdf

import (
	"encoding/binary"
	"fmt"
	"io"
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
	FOURCC_STSZ = FourCC{'s', 't', 's', 'z'}
	FOURCC_STTS = FourCC{'s', 't', 't', 's'}
	FOURCC_STZ2 = FourCC{'s', 't', 'z', '2'}
	FOURCC_TRAK = FourCC{'t', 'r', 'a', 'k'}
)

func (x FourCC) String() string {
	return string(x[:])
}

type MP4BoxHeader struct {
	Size   uint32
	FourCC FourCC
}

const MP4BoxHeaderBytes = 8

func (h MP4BoxHeader) PayloadSize() int64 {
	return int64(h.Size - MP4BoxHeaderBytes)
}

func ParseBoxHeader(r io.Reader) (res MP4BoxHeader, err error) {
	err = Read(r, &res)
	return res, err
}

type FullBox struct {
	Version byte
	Flags   [3]byte
}

type HandlerBox struct {
	FullBox
	Predefined  uint32
	HandlerType [4]byte
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

type SampleSizeTableHeader struct {
	FullBox
	ConstantSize uint32
	EntryCount   uint32
}

type SampleSizeTable struct {
	SampleSizeTableHeader
	Entries []uint32
}

func (t SampleSizeTable) SampleSize(n uint32) uint32 {
	if t.ConstantSize != 0 {
		return t.ConstantSize
	}
	return t.Entries[n]
}

func ParseSampleSizeBox(r io.Reader) (res SampleSizeTable, err error) {
	err = Read(r, &res.SampleSizeTableHeader)
	if err != nil {
		return res, err
	}
	res.Entries = make([]uint32, res.EntryCount)
	return res, Read(r, res.Entries)
}

type SampleToChunkEntry struct {
	FirstChunk             uint32
	SamplesPerChunk        uint32
	SampleDescriptionIndex uint32
}

type SampleToChunkTable struct {
	SimpleTableHeader
	Entries []SampleToChunkEntry
}

func ParseSampleToChunkBox(r io.Reader) (res SampleToChunkTable, err error) {
	err = Read(r, &res.SimpleTableHeader)
	if err != nil {
		return res, err
	}
	res.Entries = make([]SampleToChunkEntry, res.EntryCount)
	return res, Read(r, res.Entries)
}

type ChunkOffsetTable interface {
	Count() int
	Offset(n uint32) uint64
}

type ChunkOffset32Table struct {
	Entries []uint32
}

func (t ChunkOffset32Table) Count() int {
	return len(t.Entries)
}
func (t ChunkOffset32Table) Offset(n uint32) uint64 {
	return uint64(t.Entries[n])
}

type ChunkOffset64Table struct {
	Entries []uint64
}

func (t ChunkOffset64Table) Count() int {
	return len(t.Entries)
}
func (t ChunkOffset64Table) Offset(n uint32) uint64 {
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