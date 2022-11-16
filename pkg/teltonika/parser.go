package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math"
	"strings"

	"github.com/pkg/errors"
)

type tcpHeader struct {
	Zeros   uint32
	DataLen uint32
}

type packetHeader struct {
	Codec uint8
	Count uint8
}

type footer struct {
	Count uint8
	_     uint16
	CRC   uint16
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

// type BeaconData struct {
// 	uuid [16]byte
// 	major [2]byte
// 	minor [2]byte
// 	rssi [1]byte
// }

func main() {
	// NOTE: byte array <-> string, base on ASCII
	// byteArray := []byte{'G', 'O', 'L', 'A', 'N', 'G'} // =[]byte("GOLANG")
	// fmt.Println("String as byte array:", byteArray)
	// str1 := string(byteArray)
	// fmt.Println("String:", str1)

	// sample codec8
	data := "000000000000003608010000016B40D8EA30010000000000000000000000000000000105021503010101425E0F01F10000601A014E0000000000000000010000C7CF"

	// sample codec8e
	// data := "000000000000004A8E010000016B412CEE000100000000000000000000000000000000010005000100010100010011001D00010010015E2C880002000B000000003544C87A000E000000001DD7E06A00000100002994"

	// beacon sensor
	// data := `000000000000005A8E010000016B69B0C9510000000000000000000000000000000001810001000000000000000000010181002D11216B817F8A274D4FBDB62D33E1842F8DF8014D022BBF21A579723675064DC396A7C3520129F61900000000BF0100003E5D`

	// handbrake, chua chinh xac
	// data := "00000000000000688E010000016B69B0C9510000000000000000000000000000000000840001000000000000000000010084000c000000000800000100003E5D"

	// NOTE: Reader: string to reader
	// r := strings.NewReader(data)
	// fmt.Println("Reader:", r)

	// NOTE: hex string to byte array
	decoded, err := hex.DecodeString(data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%x\n", decoded)
	fmt.Println("hex to decimal:", decoded)

	// convert to Reader, to read a stream of data
	// why need convert a byte slice into an io.Reader, because real data is stream, not string
	// Reader contains current reading index
	// NOTE: byte array to reader
	reader := bytes.NewReader(decoded)
	fmt.Println("reader", reader)

	// read header
	tcph := new(tcpHeader)
	// read1: read by structured binary data
	err = binary.Read(reader, binary.BigEndian, tcph)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("header by hex: %x\n", tcph)
	fmt.Println("header by decimal:", tcph)
	fmt.Println("DataLen:", tcph.DataLen)
	fmt.Println("reader", reader)

	// read body
	// why not use binary.Read, because not struct yet for avl?
	// because need checksum in footer first
	avl := make([]byte, tcph.DataLen-1)
	// read2: read len bytes
	_, err4 := io.ReadFull(reader, avl)
	if err4 != nil {
		log.Fatal(err4)
	}
	fmt.Println("Body", avl)
	fmt.Println("reader", reader)

	// read footer
	f := new(footer)
	err = binary.Read(reader, binary.BigEndian, f)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Footer %x\n", f)
	fmt.Println("Footer decimal", f)

	// avl = append(avl, f.Count)
	// TODO: do this to checksum
	// fmt.Println("new body", avl)

	// because reader has already been read
	// so need create a new reader, but why need if I have got the avl data already
	// TODO: why use buffer instead of reader, dung nhu o duoi van cho ket qua dung, Reader vs Buffer: https://pkg.go.dev/bytes#Reader
	buffer := new(bytes.Buffer)
	// buffer := bytes.NewReader(avl)
	// read3: to write into buffer
	// buf.Write(avl[0 : len(avl)-1])
	buffer.Write(avl[0:])
	fmt.Printf("buffer %x\n", buffer)
	fmt.Printf("buffer decimal %d\n", buffer)

	// read codec and number of records
	// FIXME: thu tu doc dang ko hop li, cai nay nen doc ngay sau tcpHeader
	ph := new(packetHeader)
	err = binary.Read(buffer, binary.BigEndian, ph)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("packetHeader", ph)

	if ph.Codec == 8 {
		records8 := make([]*Record, ph.Count)
		for i := 0; i < int(ph.Count); i++ {
			var record *Record
			record, err = parseRecord(buffer)
			if err != nil {
				log.Fatal(err)
			}
			records8[i] = record
		}
		fmt.Println("records", records8[0])
	} else {
		records8e := make([]*Record8e, ph.Count)
		for i := 0; i < int(ph.Count); i++ {
			var record *Record8e
			record, err = parseRecord8e(buffer)
			if err != nil {
				log.Fatal(err)
			}
			records8e[i] = record
		}
		fmt.Println("records", records8e[0])
	}

	// TODO: print all records
	// TODO: name record data, see fleetlog
	// fmt.Println("records", records[0])
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

func parseBeacon() {
	s := "11210102030405060708090a0b0c0d0e0f1023262326bf210102030405060708090a0b0c0d0e0f1023532353c0210102030405060708090a0b0c0d0e0f1023502350c1210102030405060708090a0b0c0d0e0f1023512351bf210102030405060708090a0b0c0d0e0f1023282328bc210102030405060708090a0b0c0d0e0f1020d120d1c02110190d0c0b0a0908070605040302010000020018a1"
	x := s[2:]
	y := strings.Split(x, "21")
	v := y[1:]
	for _, z := range v {
		uuid := z[:32]
		major := z[32:36]
		minor := z[36:40]
		rssi := z[40:]
		fmt.Println(uuid, major, minor, rssi)
		// decoded, err := hex.DecodeString(z)
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// fmt.Println(decoded)
		// 	reader := bytes.NewReader(decoded)
		// 	b := new(BeaconData)
		// err = binary.Read(reader, binary.BigEndian, b)
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// fmt.Println(b)
	}
}
