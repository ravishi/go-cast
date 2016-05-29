package cast

import (
	"encoding/binary"
	"errors"
	"github.com/gogo/protobuf/proto"
	"io"
)

var IncompleteReadError = errors.New("Failed to read all the data")
var IncompleteWriteError = errors.New("Failed to write all the data")

func Read(r io.Reader) (*CastMessage, error) {
	data, err := ReadMessage(r)
	if err != nil && err != io.EOF {
		return nil, err
	}

	message := &CastMessage{}

	err = proto.Unmarshal(data, message)
	if err != nil {
		return nil, err
	}

	return message, nil
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
	var length uint32
	err := binary.Read(r, binary.BigEndian, &length)
	if err != nil {
		return nil, err
	}

	if length > 0 {
		buf := make([]byte, length)

		i, err := r.Read(buf)
		if err != nil && err != io.EOF {
			return nil, err
		}

		if i < 0 || uint32(i) != length {
			return nil, IncompleteReadError
		}

		// Return err. It can be io.EOF meaning we just read
		// the last message while the connection was closing.
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
