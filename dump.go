package main

import (
	"encoding/hex"
	"io"
	"io/ioutil"
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
	if err != nil {
		log.Fatalf("Handler Read Error: %s", err)
	}

	o.Logf("Format: %s", string(h.HandlerType[:]))
}

func DumpSampleToChunks(r io.Reader, o OutputContext) {
	t, err := ParseSampleToChunkBox(r)
	if err != nil {
		log.Fatalf("SampleToChunks Read Error: %s", err)
	}
	o.Logf("Table of %d entries", t.EntryCount)
}

func DumpChunkOffsets(r io.Reader, fourCC FourCC, o OutputContext) {
	t, err := ParseChunkOffsetBox(r, fourCC)
	if err != nil {
		log.Fatalf("ChunkOffset Read Error: %s", err)
	}
	o.Logf("Table of %d entries", t.Count())
}

func DumpSampleSizes(r io.Reader, o OutputContext) {
	t, err := ParseSampleSizeBox(r)
	if err != nil {
		log.Fatalf("SampleSize Read Error: %s", err)
	}
	o.Logf("Table of %d entries", t.EntryCount)
}

func DumpUnknown(r io.Reader, o OutputContext) {
	var dump [128]byte
	count, _ := r.Read(dump[:])
	o.Logf("Data:\n%s", hex.Dump(dump[0:count]))
}

func DumpBox(r io.Reader, o OutputContext) {
	nested := o.Nested()
	for {
		h, err := ParseBoxHeader(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Header Parse Error: %s", err)
		}

		o.Logf("Box Type: %s, Size: %d", string(h.FourCC[:]), h.Size)

		payload := io.LimitReader(r, h.PayloadSize())
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

		_, err = io.Copy(ioutil.Discard, payload)
		if err != nil && err != io.EOF {
			log.Fatalf("Payload Consumption Error: %s", err)
		}
	}
}
