package oobmultipartreader

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"net/textproto"
	"sort"
)

// Field is a Multipart Fragment that defines would be written in the response
type Field struct {
	Header textproto.MIMEHeader
	Reader io.Reader
}

// NextField is a function that returns the next Field
type NextField func(reader *OOBMultipartReader, field *Field) (err error)

// NoNextFieldError is an error that is triggered as soon as the NextField in OOBMultipartReader is nil
type NoNextFieldError struct{}

func (NoNextFieldError) Error() string { return "NextField is nil" }

// OOBMultipartReader is a io.Reader that generates a multipart stream out of multiple other readers
type OOBMultipartReader struct {
	Boundary      string
	NextField     NextField
	WrittenFields int

	initialized   bool
	currentReader io.Reader
	headerBuffer  *bytes.Buffer
	finalBuffer   *bytes.Buffer
}

func (reader *OOBMultipartReader) Read(p []byte) (int, error) {

	if !reader.initialized {
		reader.WrittenFields = 0
		if len(reader.Boundary) <= 0 {
			reader.Boundary = randomBoundary()
		}
		if reader.NextField == nil {
			return 0, NoNextFieldError{}
		}
		reader.initialized = true
	}

	if reader.finalBuffer != nil {
		return reader.writeFinalBuffer(p)
	}

	if reader.currentReader == nil {
		field := Field{
			Header: make(textproto.MIMEHeader),
		}
		if err := reader.NextField(reader, &field); err != nil {
			if err == io.EOF {
				reader.finalBuffer = &bytes.Buffer{}
				fmt.Fprintf(reader.finalBuffer, "\r\n--%s--", reader.Boundary)
				return reader.writeFinalBuffer(p)
			}
			return 0, err
		}
		reader.fillHeaderBuffer(field.Header)
		reader.currentReader = field.Reader
	}

	if reader.headerBuffer != nil {
		return reader.writeHeaderBuffer(p)
	}

	n, err := reader.currentReader.Read(p)
	if err != nil {
		// if the reader is EOF go to the next field
		if err == io.EOF {
			reader.currentReader = nil
			return 0, nil
		}
		return 0, err
	}
	return n, nil
}

func (reader *OOBMultipartReader) writeHeaderBuffer(p []byte) (int, error) {
	n, err := reader.headerBuffer.Read(p)
	if err == io.EOF {
		reader.WrittenFields++
		reader.headerBuffer = nil
		return n, nil
	}
	return n, err
}

func (reader *OOBMultipartReader) writeFinalBuffer(p []byte) (int, error) {
	n, err := reader.finalBuffer.Read(p)
	if err == io.EOF {
		reader.finalBuffer = nil
		reader.initialized = false
	}
	return n, err
}

func (reader *OOBMultipartReader) fillHeaderBuffer(header textproto.MIMEHeader) {
	reader.headerBuffer = &bytes.Buffer{}
	// write the multipart header
	if reader.WrittenFields == 0 {
		fmt.Fprintf(reader.headerBuffer, "--%s\r\n", reader.Boundary)
	} else {
		fmt.Fprintf(reader.headerBuffer, "\r\n--%s\r\n", reader.Boundary)
	}
	keys := make([]string, 0, len(header))
	for k := range header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for _, v := range header[k] {
			fmt.Fprintf(reader.headerBuffer, "%s: %s\r\n", k, v)
		}
	}
	fmt.Fprintf(reader.headerBuffer, "\r\n")
}

func randomBoundary() string {
	var buf [30]byte
	_, err := io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", buf[:])
}
