What is this
============
A simple stupid program, parsing and extracting audio tracks from an MP4, over HTTP.

Basic use-case: extract mp3 from a remote video, and pipe it on stdout.

Running
=======

```bash
go run *.go <url> | mpg123 -
```

Design
======
The code is designed around 3 abstractions;

LimitedSeekReader
-----------------
A windowed io.SeekReader, helpful for reading sections of files

HTTPSeekReader
--------------
An HTTP-implementation of an io.SeekReader, using range-requests for seek

MP4
---
Simple library helping to parse Boxes in MP4-files, with a Scanner that takes a Box (as a SeekReader), and emits all inner Boxes (As Header + Payload in a SeekReader) to its recieved function

The rest
--------
Fairly simple. See main.go for the code. Basically;

  1. Seek through file, looking for MOOV-box, and all audio-tracks matching the desired type
  2. For each found track, loop through chunks and copy them to stdout

(Dump.go)
---------
Unused. Initially developed to test MP4-parsing. Left since helpful to debug new files

Limitations / TODO
==================
  - Only parts of ISO base media file format supported (as described at http://l.web.umkc.edu/lizhu/teaching/2016sp.video-communication/ref/mp4.pdf)
  - The code is designed with as few assumptions as possible. (For example regarding Box order). That said, due to the complexity of the specification, there is no guarantee all possible streams can be handled.
  - Will not handle HUGE files gracefully. (Stores some tables in RAM)
  - Not designed to work as a library. No separation between public and private methods (yet)
  - No fallback for non-Range supporting HTTP servers
  - Currently wastes a good bit of bandwidth over HTTP. Could be severely optimized.
