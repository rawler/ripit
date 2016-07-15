package main

import (
	"io"
	"log"
	"os"
	"regexp"
)

type MediaTrack struct {
	handlerType TrackType
	sizes       SampleSizeTable
	chunkMap    SampleToChunkTable
	offsets     ChunkOffsetTable
}

func (t MediaTrack) Read(r io.ReadSeeker, tgt io.Writer) error {
	sampleIndex := 0
	for chunkIndex := 0; chunkIndex < t.offsets.Count(); chunkIndex++ {
		samples := t.chunkMap.ChunkSamples(chunkIndex)
		chunkSize := t.sizes.SampleSizes(sampleIndex, samples)
		chunkOffset := t.offsets.Offset(chunkIndex)

		_, err := r.Seek(int64(chunkOffset), os.SEEK_SET)
		if err != nil {
			return err
		}
		io.CopyN(tgt, r, int64(chunkSize))

		sampleIndex += samples
	}
	return nil
}

func extractTrack(r io.ReadSeeker, track *MediaTrack) error {
	return ScanBoxes(r, func(h MP4BoxHeader, payload io.ReadSeeker) error {
		var err error
		switch h.FourCC {
		case FOURCC_MINF, FOURCC_STBL, FOURCC_MDIA:
			extractTrack(payload, track) // Flatten exactly-once-hierarchy
		case FOURCC_HDLR:
			var hdlr HandlerBox
			hdlr, err = ParseHandler(payload)
			track.handlerType = hdlr.HandlerType
		case FOURCC_STSZ:
			track.sizes, err = ParseSampleSizeBox(payload)
		case FOURCC_STSC:
			track.chunkMap, err = ParseSampleToChunkBox(payload)
		case FOURCC_STCO, FOURCC_CO64:
			track.offsets, err = ParseChunkOffsetBox(payload, h.FourCC)
		}
		return err
	})
}

func AudioTracksFromFile(r io.ReadSeeker) (tracks []MediaTrack, err error) {
	err = ScanBoxes(r, func(h MP4BoxHeader, payload io.ReadSeeker) error {
		var err error
		switch h.FourCC {
		case FOURCC_MOOV:
			tracks, err = AudioTracksFromFile(payload) // Flatten exactly-once-hierarchy
		case FOURCC_TRAK:
			var track MediaTrack
			err = extractTrack(payload, &track)
			if track.handlerType == TrackTypeSound {
				tracks = append(tracks, track)
			}
		}
		return err
	})
	return tracks, err
}

func AssertOK(err error, format string, args ...interface{}) {
	if err == nil {
		return
	}
	args = append(args, err)
	log.Fatalf(format+": %s", args...)
}

var urlRegex = regexp.MustCompile("^[a-z0-9]+://.*$")

func Open(url string) (io.ReadSeeker, error) {
	if urlRegex.MatchString(url) {
		return HTTPSeekReader(url)
	}
	return os.Open(url)
}

func main() {
	if len(os.Args) == 1 {
		log.Fatalf("Usage: %s <fname>", os.Args[0])
	}

	r, err := Open(os.Args[1])
	AssertOK(err, "Error on open")

	tracks, err := AudioTracksFromFile(r)
	AssertOK(err, "Failed to scan AudioTracks from MOOV")
	log.Printf("%d tracks", len(tracks))

	for i, track := range tracks {
		AssertOK(track.Read(r, os.Stdout), "Failed to Read track %d", i)
	}
}
