package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/pkg/errors"
)

type dataRecord8e struct {
	Timestamp  uint64
	Priority   uint8
	Longitude  int32
	Latitude   int32
	Altitude   int16
	Angle      uint16
	Satellites uint8
	Speed      uint16
	Event      uint16
	IOCount    uint16
}

type ioRecord8e struct {
	ID    uint16
	Value []byte
}

type Record8e struct {
	dataRecord8e
	IO []ioRecord8e
}

func main() {
	data := `0000016B69B0C9510000000000000000000000000000000001810001
	000000000000000000010181002D11216B817F8A274D4FBDB62D33E1842F8DF8014D022BBF21A579
	723675064DC396A7C3520129F61900000000BF0100003E5D`

	// reader := strings.NewReader(data)

	// buf := make([]byte, 100)
	// for {
	// 	n, err := reader.Read(buf)
	// 	fmt.Println(n, err, buf[:n])
	// 	if err == io.EOF {
	// 		break
	// 	}
	// }

	// var b bytes.Buffer
	b := new(bytes.Buffer)

	b.WriteString(data)

	var lol *Record8e
	lol, err := parseRecord8e(b)
	if err != nil {
		fmt.Println(lol)
	}
}

func parseRecord8e(r io.Reader) (rec *Record8e, err error) {
	rec = new(Record8e)
	dr := dataRecord8e{}
	err = binary.Read(r, binary.BigEndian, &dr)
	if err != nil {
		return nil, errors.Wrap(err, "read failed")
	}
	rec.dataRecord8e = dr
	rec.IO = make([]ioRecord8e, 0, rec.IOCount)

	for i := 0; i < 5; i++ {
		var count uint16
		err = binary.Read(r, binary.BigEndian, &count)
		if err != nil {
			return
		}

		if i == 4 {
			// value length is not fixed
			for j := 0; j < int(count); j++ {
				var idx uint16
				var valueLength uint16
				err = binary.Read(r, binary.BigEndian, &idx)
				if err != nil {
					return nil, errors.Wrap(err, "IO id read failed")
				}
				err = binary.Read(r, binary.BigEndian, &valueLength)
				if err != nil {
					return nil, errors.Wrap(err, "IO value length read failed")
				}
				data := make([]byte, valueLength)
				err = binary.Read(r, binary.BigEndian, &data)
				if err != nil {
					return nil, errors.Wrap(err, "IO value read failed")
				}
				rec.IO = append(rec.IO, ioRecord8e{ID: idx, Value: data})
			}
		} else {
			length := int(math.Pow(float64(2), float64(i)))
			for j := 0; j < int(count); j++ {
				var id uint16
				data := make([]byte, length)
				err = binary.Read(r, binary.BigEndian, &id)
				if err != nil {
					return nil, errors.Wrap(err, "IO id read failed")
				}
				err = binary.Read(r, binary.BigEndian, &data)
				if err != nil {
					return nil, errors.Wrap(err, "IO value read failed")
				}
				rec.IO = append(rec.IO, ioRecord8e{ID: id, Value: data})
			}
		}
	}
	return
}
