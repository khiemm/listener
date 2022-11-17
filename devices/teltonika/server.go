package teltonika

import (
	"bytes"
	"encoding/binary"
	"io"

	"time"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/thejerf/suture"
	"github.com/khiemm/listener/devices/common"
	"github.com/khiemm/listener/util"
)

func MakeServer(addr string, connSupervisor *suture.Supervisor) suture.Service {
	s := new(common.Server)
	s.Name = "teltonika"
	s.Addr = addr
	s.ConnectionSupervisor = connSupervisor
	s.InteractorGenerator = func(s *common.Server) common.Interactor { return Interactor{} }
	return s
}

type Interactor struct{}

// InitializeConnection reads device IMEI from connection, loads the corresponding
// vehicle from the database, saves its information to fmConnection and
// sends a confirmation byte to the device.
// If the device sends an IMEI that's not found in the database,
// 00 is sent to the device, connection is closed and ErrUnauthorizedDevice
// is returned.
func (_ Interactor) InitializeConnection(h *common.Handler) (err error) {
	h.Conn.SetDeadline(util.MakeTimeout(viper.GetInt("teltonika.timeout")))

	var buff = make([]byte, 10)
	_, err = io.ReadFull(h.Conn, buff)

	reader := bytes.NewReader(buff)

	var imei string

	tsm232 := isTSM232(reader)

	if tsm232 {

		h.Log().Debug("---isTSM232")
		imei, err = parseTSM232IMEI(h.Conn)

		if err != nil {
			return
		}
	} else {
		var buff2 = make([]byte, 7)
		_, err = io.ReadFull(h.Conn, buff2)
		buff = append(buff, buff2...)
		reader2 := bytes.NewReader(buff)
		imei, err = parseIMEI(reader2)
	}
	h.Log().Debug(imei)

	if err != nil {
		return
	}

	// vehicle, err := h.Store.StorageService().GetVehicleByIMEI(context.TODO(), imei)
	// if err != nil {
	// 	return
	// }

	// if vehicle == nil {
	// 	_, sendErr := h.Conn.Write([]byte{0})
	// 	if sendErr != nil {
	// 		h.Log().WithError(err).Error("Error when sending refusal byte to an unauthorized Teltonika")
	// 	}
	// 	h.Conn.Close()
	// 	return common.ErrUnauthorizedDevice
	// } else {
	// 	h.ID = vehicle.Id
	// 	h.Debug = vehicle.Debug

	// 	if tsm232 {
	// 		records, _ := ParseForTSM232(h.Conn)
	// 		recordsSlice := make([]interface{}, len(records))
	// 		for i, r := range records {
	// 			recordsSlice[i] = r
	// 		}
	// 		err = h.SaveRecords(recordsSlice)

	// 		err = binary.Write(h.Conn, binary.BigEndian, uint32(len(records)))
	// 		h.Conn.Close()

	// 	} else {
	// 		_, err = h.Conn.Write([]byte{1})
	// 	}

	// 	if err != nil {
	// 		return
	// 	}
	// }
	return
}

func (_ Interactor) ParseMessage(h *common.Handler) (result interface{}, err error) {
	return Parse(h.Conn)
}

func (_ Interactor) HandleMessage(h *common.Handler, msg interface{}) (err error) {
	if h.Debug {
		h.Log().Debugf("Raw message: %x", h.GetLastRawMessage())
	}
	if records, ok := msg.([]*Record); ok {
		records = msg.([]*Record)
		recordsSlice := make([]interface{}, len(records))
		for i, r := range records {
			recordsSlice[i] = r
		}
		// err = h.SaveRecords(recordsSlice)
		// if err != nil {
		// 	return errors.Wrap(err, "saving records failed")
		// }

		err = binary.Write(h.Conn, binary.BigEndian, uint32(len(records)))
		if err != nil {
			return errors.Wrap(err, "couldn't confirm receipt")
		}
	}
	if records, ok := msg.([]*Record8e); ok {
		records = msg.([]*Record8e)
		recordsSlice := make([]interface{}, len(records))
		for i, r := range records {
			recordsSlice[i] = r
		}
		// err = h.SaveRecords(recordsSlice)
		// if err != nil {
		// 	return errors.Wrap(err, "saving records failed")
		// }

		err = binary.Write(h.Conn, binary.BigEndian, uint32(len(records)))
		if err != nil {
			return errors.Wrap(err, "couldn't confirm receipt")
		}
	}
	return nil
}

func (_ Interactor) HandleError(h *common.Handler, _ error) (terminate bool) {
	sendError(h.Conn)
	return true
}

func (_ Interactor) GetConnectionTimeout(h common.Handler) time.Duration {
	return time.Duration(viper.GetInt("teltonika.timeout")) * time.Second
}

// func (_ Interactor) MakeDBRecord(h common.Handler, record interface{}) (dbr *models.Record, ior []*models.Parameter) {
// 	if r, ok := record.(*Record); ok {
// 		if !verifyRecord(r) {
// 			return nil, nil
// 		}
// 		dbr = &models.Record{
// 			Vehicle:    h.ID,
// 			Datetime:   mysql.NullTime{Valid: true, Time: time.Unix(0, int64(r.Timestamp*1000000)).In(time.UTC)},
// 			Longitude:  fmt.Sprintf("%f", float64(r.Longitude)/10000000.0),
// 			Latitude:   fmt.Sprintf("%f", float64(r.Latitude)/10000000.0),
// 			Altitude:   float64(r.Altitude),
// 			Angle:      int32(r.Angle),
// 			Satellites: int32(r.Satellites),
// 			Speed:      int32(r.Speed),
// 			Created:    mysql.NullTime{Valid: true, Time: time.Now()},
// 		}
// 		ior = make([]*models.Parameter, 0)
// 		for _, param := range r.IO {
// 			switch param.ID {
// 			case 1:
// 				dbr.Ignition = param.Value[0] != byte(0)
// 			case 78:
// 				dbr.Ibutton = sql.NullString{Valid: true, String: hex.EncodeToString(param.Value)}
// 			case 199:
// 				value := binary.BigEndian.Uint32(param.Value)
// 				dbr.Distance = sql.NullInt64{Valid: true, Int64: int64(value)}
// 			default:
// 				val, _ := util.HexNumberFromBytes(param.Value)
// 				ior = append(ior, &models.Parameter{
// 					Parameter: int64(param.ID),
// 					Value:     val,
// 				})
// 			}
// 		}
// 	}
// 	if r, ok := record.(*Record8e); ok {
// 		if !verifyRecord(r) {
// 			return nil, nil
// 		}
// 		dbr = &models.Record{
// 			Vehicle:    h.ID,
// 			Datetime:   mysql.NullTime{Valid: true, Time: time.Unix(0, int64(r.Timestamp*1000000)).In(time.UTC)},
// 			Longitude:  fmt.Sprintf("%f", float64(r.Longitude)/10000000.0),
// 			Latitude:   fmt.Sprintf("%f", float64(r.Latitude)/10000000.0),
// 			Altitude:   float64(r.Altitude),
// 			Angle:      int32(r.Angle),
// 			Satellites: int32(r.Satellites),
// 			Speed:      int32(r.Speed),
// 			EventID:    sql.NullInt64{Valid: true, Int64: int64(r.Event)},
// 			Created:    mysql.NullTime{Valid: true, Time: time.Now()},
// 		}
// 		ior = make([]*models.Parameter, 0)
// 		for _, param := range r.IO {
// 			switch param.ID {
// 			case 1:
// 				dbr.Ignition = param.Value[0] != byte(0)
// 			case 78:
// 				dbr.Ibutton = sql.NullString{Valid: true, String: hex.EncodeToString(param.Value)}
// 			case 199:
// 				value := binary.BigEndian.Uint32(param.Value)
// 				dbr.Distance = sql.NullInt64{Valid: true, Int64: int64(value)}
// 			case 385:
// 				h.Log().Debugf("event 385 from id %v: %x, %x", h.ID, r.Event, r.IO)
// 				ior = append(ior, &models.Parameter{
// 					Parameter: int64(param.ID),
// 					Value:     hex.EncodeToString(param.Value),
// 				})
// 			default:
// 				val, _ := util.HexNumberFromBytes(param.Value)
// 				ior = append(ior, &models.Parameter{
// 					Parameter: int64(param.ID),
// 					Value:     val,
// 				})
// 			}
// 		}
// 	}
// 	return
// }

// verifyRecord checks whether the record should be saved into database
// using a set of basic sanity checks.
func verifyRecord(record interface{}) bool {
	if r, ok := record.(*Record8e); ok {
		if r.Latitude == 0 && r.Longitude == 0 {
			return false
		}
		if r.Timestamp > uint64(time.Now().Unix()*1000+2*60*60000) {
			return false
		}
	}
	if r, ok := record.(*Record); ok {
		if r.Latitude == 0 && r.Longitude == 0 {
			return false
		}
		if r.Timestamp > uint64(time.Now().Unix()*1000+2*60*60000) {
			return false
		}
	}
	return true
}

func (_ Interactor) CloseConnection(h common.Handler) (err error) { return nil }

func sendError(w io.Writer) {
	w.Write([]byte{0, 0, 0, 0})
}
