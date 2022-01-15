package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math"

	"github.com/pkg/errors"
)

type tcpHeader struct {
	Zeros   uint32
	DataLen uint32
}

type footer struct {
	Count uint8
	_     uint16
	CRC   uint16
}

type packetHeader struct {
	Codec uint8
	Count uint8
}

type dataRecord struct {
	Timestamp  uint64
	Priority   uint8
	Longitude  int32
	Latitude   int32
	Altitude   int16
	Angle      uint16
	Satellites uint8
	Speed      uint16
	Event      uint8
	IOCount    uint8
}

type ioRecord struct {
	ID    uint8
	Value []byte
}

type Record struct {
	dataRecord
	IO []ioRecord
}

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
	// data := "000000000000003608010000016B40D8EA30010000000000000000000000000000000105021503010101425E0F01F10000601A014E0000000000000000010000C7CF"
	// data := "000000000000004A8E010000016B412CEE000100000000000000000000000000000000010005000100010100010011001D00010010015E2C880002000B000000003544C87A000E000000001DD7E06A00000100002994"
	// beacon sensor
	data := `000000000000005A8E010000016B69B0C9510000000000000000000000000000000001810001000000000000000000010181002D11216B817F8A274D4FBDB62D33E1842F8DF8014D022BBF21A579723675064DC396A7C3520129F61900000000BF0100003E5D`

	// r := strings.NewReader(data)
	// fmt.Println(r)
	// reader2 := []byte(data)
	// fmt.Println(reader2)
	// get byte array
	decoded, err := hex.DecodeString(data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%x\n", decoded)
	fmt.Println(decoded)

	// convert to Reader
	reader := bytes.NewReader(decoded)

	// read header
	tcph := new(tcpHeader)
	err = binary.Read(reader, binary.BigEndian, tcph)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Header %x\n", tcph)
	fmt.Println("Header", tcph)
	fmt.Println("reader", reader)

	// get body
	avl := make([]byte, tcph.DataLen-1)
	_, err4 := io.ReadFull(reader, avl)
	if err4 != nil {
		log.Fatal(err4)
	}
	fmt.Println("Body", avl)

	// read footer
	f := new(footer)
	err = binary.Read(reader, binary.BigEndian, f)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Footer", f)

	// TODO: not understand yet
	avl = append(avl, f.Count)

	// because reader has already been read
	// so need create a new reader
	// TODO: user reader instead of buffer
	buf := new(bytes.Buffer)
	buf.Write(avl[0 : len(avl)-1])
	fmt.Printf("buf %x\n", buf)

	// read codec and number of records
	ph := new(packetHeader)
	err = binary.Read(buf, binary.BigEndian, ph)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("packetHeader", ph)

	// TODO: check codec 8 or 8e
	records := make([]*Record8e, ph.Count, ph.Count)
	for i := 0; i < int(ph.Count); i++ {
		var record *Record8e
		record, err = parseRecord8e(buf)
		if err != nil {
			log.Fatal(err)
		}
		records[i] = record
	}
	// TODO: print all records
	// TODO: name record data, see fleetlog
	fmt.Println("records", records[0])
}

func parseRecord(r io.Reader) (rec *Record, err error) {
	rec = new(Record)
	dr := dataRecord{}
	err = binary.Read(r, binary.BigEndian, &dr)
	if err != nil {
		return nil, errors.Wrap(err, "read failed")
	}
	rec.dataRecord = dr
	rec.IO = make([]ioRecord, 0, rec.IOCount)

	for i := 0; i < 4; i++ {
		var count uint8
		err = binary.Read(r, binary.BigEndian, &count)
		if err != nil {
			return
		}

		length := int(math.Pow(float64(2), float64(i)))

		for j := 0; j < int(count); j++ {
			var id uint8
			data := make([]byte, length)
			err = binary.Read(r, binary.BigEndian, &id)
			if err != nil {
				return nil, errors.Wrap(err, "IO id read failed")
			}
			err = binary.Read(r, binary.BigEndian, &data)
			if err != nil {
				return nil, errors.Wrap(err, "IO value read failed")
			}
			rec.IO = append(rec.IO, ioRecord{ID: id, Value: data})
		}
	}
	return
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
