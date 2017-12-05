package oobmultipartreader

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"net/textproto"
	"sort"
)

type Field struct {
	Header textproto.MIMEHeader
	Reader io.Reader
}

type NextField func(reader *OOBMultipartReader, field *Field) (err error)

type NoNextFieldError struct{}

func (NoNextFieldError) Error() string { return "NextField is nil" }

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
		reader.headerBuffer = &bytes.Buffer{}
		// write the multipart header
		if reader.WrittenFields == 0 {
			fmt.Fprintf(reader.headerBuffer, "--%s\r\n", reader.Boundary)
		} else {
			fmt.Fprintf(reader.headerBuffer, "\r\n--%s\r\n", reader.Boundary)
		}
		keys := make([]string, 0, len(field.Header))
		for k := range field.Header {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			for _, v := range field.Header[k] {
				fmt.Fprintf(reader.headerBuffer, "%s: %s\r\n", k, v)
			}
		}
		fmt.Fprintf(reader.headerBuffer, "\r\n")
		reader.currentReader = field.Reader
	}

	if reader.headerBuffer != nil {
		n, err := reader.headerBuffer.Read(p)
		if err == io.EOF {
			reader.WrittenFields++
			reader.headerBuffer = nil
			return n, nil
		}
		return n, err
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

func (reader *OOBMultipartReader) writeFinalBuffer(p []byte) (int, error) {
	n, err := reader.finalBuffer.Read(p)
	if err == io.EOF {
		reader.finalBuffer = nil
		reader.initialized = false
	}
	return n, err

}

func randomBoundary() string {
	var buf [30]byte
	_, err := io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", buf[:])
}
