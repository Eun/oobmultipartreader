package oobmultipartreader_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"testing"

	"github.com/Eun/oobmultipartreader"
)

func TestReader(T *testing.T) {
	buffer := bytes.Buffer{}

	i := 0

	reader := oobmultipartreader.OOBMultipartReader{
		Boundary: "TestBoundary",
		NextField: oobmultipartreader.NextField(func(reader *oobmultipartreader.OOBMultipartReader, field *oobmultipartreader.Field) error {
			// finish after 3 fields
			if i >= 3 {
				return io.EOF
			}
			field.Reader = bytes.NewBuffer([]byte(fmt.Sprintf("Hello %d", i)))
			field.Header.Add("Content-Disposition", fmt.Sprintf(`form-data; name="File%d"`, i))
			field.Header.Add("Custom-Header", fmt.Sprintf(`%d`, i))
			i++
			return nil
		}),
	}

	_, err := io.Copy(&buffer, &reader)
	if err != nil {
		panic(err)
	}
	if reader.WrittenFields != 3 {
		panic(fmt.Sprintf("expected WrittenFields to be 3 but was %d", reader.WrittenFields))
	}

	// check if the result matches the expected

	expected :=
		"--TestBoundary\r\nContent-Disposition: form-data; name=\"File0\"\r\nCustom-Header: 0\r\n\r\n" +
			"Hello 0" +
			"\r\n--TestBoundary\r\nContent-Disposition: form-data; name=\"File1\"\r\nCustom-Header: 1\r\n\r\n" +
			"Hello 1" +
			"\r\n--TestBoundary\r\nContent-Disposition: form-data; name=\"File2\"\r\nCustom-Header: 2\r\n\r\n" +
			"Hello 2" +
			"\r\n--TestBoundary--"

	if expected != string(buffer.Bytes()) {
		panic(fmt.Sprintf("invalid result, got %v, expected %v", buffer.Bytes(), []byte(expected)))
	}

	// try to read it with golang multipart reader

	multipartReader := multipart.NewReader(&buffer, reader.Boundary)

	i = 0
	for {
		_, err := multipartReader.NextPart()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		i++
	}

	if reader.WrittenFields != i {
		panic(fmt.Sprintf("expected Counter to be %d but was %d", reader.WrittenFields, i))
	}

}

func TestRandomBoundary(T *testing.T) {
	buffer := bytes.Buffer{}

	reader := oobmultipartreader.OOBMultipartReader{
		NextField: oobmultipartreader.NextField(func(reader *oobmultipartreader.OOBMultipartReader, field *oobmultipartreader.Field) error {
			return io.EOF
		}),
	}

	_, err := io.Copy(&buffer, &reader)
	if err != nil {
		panic(err)
	}

	if reader.WrittenFields != 0 {
		panic(fmt.Sprintf("expected WrittenFields to be 0, but was %d", reader.WrittenFields))
	}

	if len(reader.Boundary) == 0 {
		panic(fmt.Sprintf("boundary was empty"))
	}
}

func TestNoNextField(T *testing.T) {
	buffer := bytes.Buffer{}

	reader := oobmultipartreader.OOBMultipartReader{}

	_, err := io.Copy(&buffer, &reader)
	if err == nil {
		panic("Error should be set")
	}

	if noNextFieldErr, ok := err.(oobmultipartreader.NoNextFieldError); !ok {
		panic(fmt.Sprintf("expected error to be a type of NoNextFieldError, but was not"))
	} else {
		a := oobmultipartreader.NoNextFieldError{}
		if noNextFieldErr.Error() != a.Error() {
			panic("NoNextFieldError mismatch")
		}
	}

	if reader.WrittenFields != 0 {
		panic(fmt.Sprintf("expected WrittenFields to be 0, but was %d", reader.WrittenFields))
	}

}

func TestNextFieldError(T *testing.T) {
	buffer := bytes.Buffer{}

	reader := oobmultipartreader.OOBMultipartReader{NextField: oobmultipartreader.NextField(func(reader *oobmultipartreader.OOBMultipartReader, field *oobmultipartreader.Field) error {
		return errors.New("Custom Error")
	}),
	}

	_, err := io.Copy(&buffer, &reader)
	if err == nil {
		panic("Error should be set")
	}
	if err.Error() != "Custom Error" {
		panic("Error mismatch")
	}

}
