package teltonika

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"sort"

	"github.com/pkg/errors"
	"github.com/khiemm/listener/util"
)

var (
	errTcpHeader         = errors.New("Malformed transmission header")
	errUnrecognizedCodec = errors.New("Unrecognized data codec")
	errChecksumMismatch  = errors.New("Checksumming failed")
)

type tcpHeader struct {
	Zeros   uint32
	DataLen uint32
}

type tsm232Header struct {
	Protocol       uint8
	MessageLength  uint16
	MOHeaderIEI    uint8
	MOHeaderLength uint16
	CDRRef         uint32
}

type packetHeader struct {
	Codec uint8
	Count uint8
}

type dataRS232Record struct {
	Timestamp uint32
	Longitude [3]byte
	Latitude  [3]byte
	Din1      uint8
	Unused    [2]byte
	Speed     float64
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

type footer struct {
	Count uint8
	_     uint16
	CRC   uint16
}

type recordSlice []*Record

func (s recordSlice) Len() int {
	return len(s)
}

func (s recordSlice) Less(i, j int) bool {
	return s[i].Timestamp < s[j].Timestamp
}

func (s recordSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func Parse(r io.Reader) (records interface{}, err error) {
	tcph, err := parseTCPHeader(r)

	if err != nil {
		return nil, errors.Wrap(err, "TCP header read failed")
	}
	// The single subtracted byte is the final record count.
	// We still leave space for it in cap, because it's needed
	// when calculating the checksum.
	avl := make([]byte, tcph.DataLen-1, tcph.DataLen)
	_, err = io.ReadFull(r, avl)

	if err != nil {
		return nil, errors.Wrap(err, "packet body read failed")
	}

	f, err := parseFooter(r)

	if err != nil {
		return nil, errors.Wrap(err, "packet footer read failed")
	}

	avl = append(avl, f.Count)
	checksum := util.Crc16(avl, 0xA001)
	if checksum != f.CRC {
		return nil, errChecksumMismatch
	}

	buf := new(bytes.Buffer)
	buf.Write(avl[0 : len(avl)-1])

	ph, err := parsePacketHeader(buf)
	if err != nil {
		return nil, errors.Wrap(err, "packet header parsing failed")
	}
	if ph.Codec == 8 {
		records8 := make([]*Record, ph.Count, ph.Count)
		for i := 0; i < int(ph.Count); i++ {
			var record *Record
			record, err = parseRecord(buf)
			if err != nil {
				return nil, errors.Wrap(err, "location record parsing failed")
			}
			records8[i] = record
		}
		records = records8
	} else {
		records8e := make([]*Record8e, ph.Count, ph.Count)
		for i := 0; i < int(ph.Count); i++ {
			var record *Record8e
			record, err = parseRecord8e(buf)
			if err != nil {
				return nil, errors.Wrap(err, "location record parsing failed")
			}
			records8e[i] = record
		}
		records = records8e
	}

	// Teltonika sometimes sends records inside a packet
	// out of order. It doesn't really matter but historically
	// we sorted them so that it would look better in the database.
	// sort.Stable(recordSlice(records8))
	return records, nil
}

func parseTCPHeader(r io.Reader) (tcph *tcpHeader, err error) {
	tcph = new(tcpHeader)
	err = binary.Read(r, binary.BigEndian, tcph)
	if err != nil {
		return nil, errors.Wrap(err, "read failed")
	}
	if tcph.Zeros != 0 {
		return nil, errTcpHeader
	}
	return
}

func parsePacketHeader(r io.Reader) (ph *packetHeader, err error) {
	ph = new(packetHeader)
	err = binary.Read(r, binary.BigEndian, ph)
	if err != nil {
		return nil, errors.Wrap(err, "read failed")
	}
	if ph.Codec != 8 && ph.Codec != 142 {
		return nil, errUnrecognizedCodec
	}
	return
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

func parseTSM232Record(r io.Reader) (rec *Record, err error) {
	rec = new(Record)
	dr := dataRecord{}
	tsm232dr := dataRS232Record{}

	err = binary.Read(r, binary.BigEndian, &tsm232dr)

	if err != nil {
		return nil, errors.Wrap(err, "read failed")
	}

	unCalculatedLong := int(uint(tsm232dr.Longitude[2]) | uint(tsm232dr.Longitude[1])<<8 | uint(tsm232dr.Longitude[0])<<16)

	unCalculatedLat := int(uint(tsm232dr.Latitude[2]) | uint(tsm232dr.Latitude[1])<<8 | uint(tsm232dr.Latitude[0])<<16)

	dr.Timestamp = uint64(tsm232dr.Timestamp) * 1000
	dr.Longitude = int32((float64(unCalculatedLong)/46603.375 - 180) * 10000000)
	dr.Latitude = int32((float64(unCalculatedLat)/93206.75 - 90) * 10000000)
	dr.Speed = uint16(tsm232dr.Speed)
	rec.dataRecord = dr

	rec.IO = make([]ioRecord, 1, 1)
	din1Bytes := make([]byte, 1)
	din1Bytes[0] = byte(tsm232dr.Din1)
	rec.IO = append(rec.IO, ioRecord{ID: 1, Value: din1Bytes})

	return
}

func parseFooter(r io.Reader) (f *footer, err error) {
	f = new(footer)
	err = binary.Read(r, binary.BigEndian, f)
	if err != nil {
		return nil, errors.Wrap(err, "read failed")
	}
	return
}

func parseIMEI(r io.Reader) (imei string, err error) {
	var length uint16
	err = binary.Read(r, binary.BigEndian, &length)

	if err != nil {
		return
	}

	imeiBytes := make([]byte, length)
	_, err = io.ReadFull(r, imeiBytes)
	if err != nil {
		return "", errors.Wrap(err, "read failed")
	}
	return string(imeiBytes), nil
}

func isTSM232(r io.Reader) bool {
	tcph, _ := parseTSM232Header(r)
	if tcph == nil {
		return false
	}

	return tcph.Protocol == 1
}

func ParseForTSM232(r io.Reader) (records []*Record, err error) {

	unusedLen := make([]byte, 25)
	_, err = io.ReadFull(r, unusedLen)

	var dataLen uint8

	binary.Read(r, binary.BigEndian, &dataLen)

	recordsCount := dataLen / 14

	records = make([]*Record, recordsCount, recordsCount)

	for i := 0; i < int(recordsCount); i++ {
		var record *Record
		record, err = parseTSM232Record(r)
		if err != nil {
			return nil, errors.Wrap(err, "location record parsing failed")
		}
		records[i] = record
	}

	sort.Stable(recordSlice(records))

	return
}

func parseTSM232Header(r io.Reader) (tcph *tsm232Header, err error) {
	tcph = new(tsm232Header)
	err = binary.Read(r, binary.BigEndian, tcph)

	if err != nil {
		return nil, errors.Wrap(err, "read failed")
	}
	if tcph.Protocol != 1 {
		return nil, errors.Wrap(err, "not a TSM232 data")
	}
	return tcph, err
}

func parseTSM232IMEI(r io.Reader) (imei string, err error) {

	imeiBytes := make([]byte, 15)
	_, err = io.ReadFull(r, imeiBytes)
	if err != nil {
		return "", errors.Wrap(err, "read failed")
	}
	return string(imeiBytes), nil
}
