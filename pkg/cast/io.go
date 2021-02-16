package cast

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/golang/protobuf/proto"
)

var IncompleteReadError = errors.New("Failed to read all the data")
var IncompleteWriteError = errors.New("Failed to write all the data")

func Read(r io.Reader) (*CastMessage, error) {
	data, err := ReadMessage(r)
	if err != nil && err != io.EOF || data == nil {
		return nil, err
	}

	message := &CastMessage{}
	if err := proto.Unmarshal(data, message); err != nil {
		return nil, err
	}

	// We can have a non-nil message + io.EOF error
	return message, err
}

func Write(w io.Writer, message *CastMessage) error {
	proto.SetDefaults(message)

	data, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	return WriteMessage(w, data)
}

func ReadMessage(r io.Reader) ([]byte, error) {
	length := new(uint32)
	err := binary.Read(r, binary.BigEndian, length)
	if err != nil && err != io.EOF || length == nil {
		return nil, err
	}

	if *length > 0 {
		buf := make([]byte, *length)

		i, err := r.Read(buf)
		if err != nil && err != io.EOF || i <= 0 {
			return nil, err
		}

		if uint32(i) != *length {
			if err == nil {
				err = IncompleteReadError
			}
			return nil, err
		}

		// We can have a non-nil message + io.EOF error
		return buf, err
	}

	return nil, io.ErrNoProgress
}

func WriteMessage(w io.Writer, data []byte) error {
	length := len(data)
	err := binary.Write(w, binary.BigEndian, uint32(length))
	if err != nil {
		return err
	}

	written, err := w.Write(data)
	if err != nil {
		return err
	} else if written != length {
		return IncompleteWriteError
	}

	return nil
}
