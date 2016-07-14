package main

// Code to dump Box structure of an MP4

import (
	"encoding/hex"
	"io"
	"log"
)

type OutputContext string

func (c OutputContext) Logf(fmt string, args ...interface{}) {
	args = append([]interface{}{c}, args...)
	log.Printf("%s"+fmt, args...)
}

func (c OutputContext) Nested() OutputContext {
	return c + " "
}

func DumpHandler(r io.Reader, o OutputContext) {
	h, err := ParseHandler(r)
	AssertOK(err, "Handler Read Error")

	o.Logf("Format: %s", string(h.HandlerType[:]))
}

func DumpSampleToChunks(r io.Reader, o OutputContext) {
	t, err := ParseSampleToChunkBox(r)
	AssertOK(err, "SampleToChunks Read Error")
	o.Logf("Table of %d entries", t.EntryCount)
}

func DumpChunkOffsets(r io.Reader, fourCC FourCC, o OutputContext) {
	t, err := ParseChunkOffsetBox(r, fourCC)
	AssertOK(err, "ChunkOffset Read Error")
	o.Logf("Table of %d entries", t.Count())
}

func DumpSampleSizes(r io.Reader, o OutputContext) {
	t, err := ParseSampleSizeBox(r)
	AssertOK(err, "SampleSize Read Error")
	o.Logf("Table of %d entries", t.EntryCount)
}

func DumpUnknown(r io.Reader, o OutputContext) {
	var dump [128]byte
	count, _ := r.Read(dump[:])
	o.Logf("Data:\n%s", hex.Dump(dump[0:count]))
}

func DumpBox(r io.Reader, o OutputContext) {
	nested := o.Nested()
	err := ScanBoxes(r, func(h MP4BoxHeader, payload io.Reader) error {
		o.Logf("Box Type: %s, Size: %d", string(h.FourCC[:]), h.Size)

		switch h.FourCC {
		case FOURCC_MOOV, FOURCC_TRAK, FOURCC_MDIA, FOURCC_MINF:
			DumpBox(payload, nested)
		case FOURCC_HDLR:
			DumpHandler(payload, nested)
		case FOURCC_STSZ:
			DumpSampleSizes(payload, nested)
		case FOURCC_STSC:
			DumpSampleToChunks(payload, nested)
		case FOURCC_STCO, FOURCC_CO64:
			DumpChunkOffsets(payload, h.FourCC, nested)
		case FOURCC_STZ2:
			log.Fatalf("Compact Sample Size box not supported")
		case FOURCC_STBL:
			DumpBox(payload, nested)
		default:
			DumpUnknown(payload, nested)
		}
		return nil
	})
	AssertOK(err, "Failed scanning in box")
}
