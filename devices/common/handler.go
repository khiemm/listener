package common

import (
	"bytes"
	goerr "errors"
	"io"
	"io/ioutil"
	"net"
	"time"

	"github.com/Sirupsen/logrus"
	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
)

var (
	ErrUnauthorizedDevice = goerr.New("device unauthorized")
	discardLogger         = &log.Logger{
		Out:       ioutil.Discard,
		Formatter: new(emptyFormatter),
		Hooks:     make(log.LevelHooks),
		Level:     logrus.FatalLevel,
	}
)

type emptyFormatter struct{}

func (f emptyFormatter) Format(_ *log.Entry) ([]byte, error) {
	return []byte{}, nil
}

type Handler struct {
	Name       string
	Conn       net.Conn
	ID         int64
	Debug      bool
	Unregister func()
	stop       chan struct{}
	Logger     *log.Logger
	Interactor
	lastRawMessage *bytes.Buffer
	MessageData string
}

type Interactor interface {
	InitializeConnection(h *Handler) (err error)
	ParseMessage(h *Handler) (result interface{}, err error)
	HandleMessage(h *Handler, msg interface{}) (err error)
	HandleError(h *Handler, err error) (terminate bool)
	GetConnectionTimeout(h Handler) time.Duration
	CloseConnection(h Handler) (err error)
}

func (h *Handler) Serve() {
	defer func() {
		h.Conn = nil
		h.Unregister()
	}()
	h.stop = make(chan struct{})

	h.Conn.SetDeadline(makeTimeout(h.GetConnectionTimeout(*h)))

	authorized := true
	err := h.InitializeConnection(h)
	if err != nil {
		h.Log().WithError(err).Info("Couldn't initialize connection")
		if errors.Cause(err) == ErrUnauthorizedDevice {
		}
		authorized = false
	}

	if authorized {
		h.Log().Info("Connection initialized")
		err = h.Loop()
		if err != nil {
			h.Log().WithError(err).Error("Error in connection")
		}

		err = h.CloseConnection(*h)
		if err != nil {
			h.Log().WithError(err).Error("Connection couldn't be closed")
		}
	}

	closeErr := h.Conn.Close()
	if closeErr != nil {
		h.Log().WithError(err).Error("Error when closing connection")
	} else {
		h.Log().Info("Connection closed")
	}
	h.Conn = nil
}

func (h *Handler) Stop() {
	if h.Conn != nil {
		h.stop <- struct{}{}
	}
}

func (h *Handler) Loop() (err error) {
	msgChan, errChan := h.chanParser(h.Conn)
	for {
		// lastRawMessage should only be empty when testing
		if h.lastRawMessage != nil {
			h.lastRawMessage.Reset()
		}

		select {
		case msg := <-msgChan:
			h.DebugLog().Debugf("Raw message: %x", h.GetLastRawMessage())
			err = h.HandleMessage(h, msg)
			if err != nil {
				terminate := h.HandleError(h, err)
				if terminate {
					return
				}
			} else if h.Name == "queclink" {
				// queclink handle process one time
				return
			}
		case err = <-errChan:
			h.DebugLog().Debugf("Raw message: %x", h.GetLastRawMessage())
			if neterr, ok := errors.Cause(err).(net.Error); ok && neterr.Timeout() {
				h.Log().Debug("Connection timed out")
				err = nil
				return
			} else if errors.Cause(err) == io.EOF {
				h.Log().WithField("at", err).Debug("Connection closed by other side")
				err = nil
				return
			} else if err != nil {
				h.Log().WithError(err).Error("Error when communicating")
				terminate := h.HandleError(h, err)
				if terminate {
					return
				}
			}
		case <-h.stop:
			return
		}
	}
	return
}

func (h *Handler) chanParser(c net.Conn) (recChan <-chan interface{}, errChan <-chan error) {
	mc := make(chan interface{}, 1)
	ec := make(chan error, 1)
	go func() {
		for {
			err := c.SetDeadline(makeTimeout(h.GetConnectionTimeout(*h)))
			// The OpError checking's here because we use net.Pipe
			// in tests and it doesn't support setting timeouts.
			if opErr, ok := err.(*net.OpError); err != nil && !(ok && opErr.Net == "pipe") {
				h.Log().WithError(err).Debug("net.OpError in ChanParser")
				ec <- err
				return
			}
			msg, err := h.ParseMessage(h)
			if err != nil {
				// This is a debug-only log, because if it's a true error, it will be logged.
				h.DebugLog().WithError(err).Debug("There was an error in connection")
				ec <- err
				return
			} else {
				mc <- msg
			}
		}
	}()
	return mc, ec
}

func (h Handler) DebugLog() *log.Entry {
	if h.Debug {
		return h.Log()
	}
	return log.NewEntry(discardLogger)
}

func (h Handler) Log() *log.Entry {
	return h.Logger.WithFields(log.Fields{
		"id":   h.ID,
		"addr": h.Conn.RemoteAddr(),
		"src":  h.Name,
	})
}

func (h Handler) GetLastRawMessage() []byte {
	// This condition is necessary because server tests can't initialize
	// lastRawMessage buffer
	if h.lastRawMessage == nil {
		return nil
	}
	return h.lastRawMessage.Bytes()
}

func (h Handler) metricName() string {
	return "listener." + h.Name + "."
}

func makeTimeout(timeout time.Duration) time.Time {
	return time.Now().Add(timeout)
}
