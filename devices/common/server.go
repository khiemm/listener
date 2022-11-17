// Package common provides a common infrastructure for devices communicating
// using a TCP based protocol.
//
// It is based on three primitives. Server and Handler are implemented by this
// package while Interactor is an interface that implements the protocol-specific
// parts.
package common

import (
	"bytes"
	"net"

	log "github.com/Sirupsen/logrus"
	"github.com/thejerf/suture"
)

// Server accepts connections on Addr and creates Handlers which
// use Interactors returned by InteractorGenerator.
type Server struct {
	Name                 string
	Addr                 string
	ConnectionSupervisor *suture.Supervisor
	stop                 chan struct{}
	Handler              suture.Service
	InteractorGenerator  func(*Server) Interactor
}

// Serve starts the server and makes it accept connections
// on Addr.
func (s *Server) Serve() {
	s.stop = make(chan struct{})
	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"src":   s.Name,
		}).Error("Couldn't create listener")
		return
	}
	log.WithFields(log.Fields{
		"addr": s.Addr,
		"src":  s.Name,
	}).Info("Listening")
	connChan, errChan := Acceptor(listener)
	for {
		select {
		case conn := <-connChan:
			log.WithFields(log.Fields{
				"addr": conn.RemoteAddr(),
				"src":  s.Name,
			}).Info("New connection")
			msgBuf := new(bytes.Buffer)
			handler := &Handler{
				Name:           s.Name,
				Conn:           TeeConn(conn, msgBuf),
				Logger:         log.StandardLogger(),
				Interactor:     s.InteractorGenerator(s),
				lastRawMessage: msgBuf,
			}
			token := s.ConnectionSupervisor.Add(handler)
			unregister := func() { s.ConnectionSupervisor.Remove(token) }
			handler.Unregister = unregister
		case acceptErr := <-errChan:
			log.WithFields(log.Fields{
				"error": acceptErr,
				"src":   s.Name,
			}).Error("Error when accepting connection. Restarting listener.")
			listener.Close()
		case <-s.stop:
			if closeErr := listener.Close(); closeErr != nil {
				log.WithFields(log.Fields{
					"error": closeErr,
					"src":   s.Name,
				}).Error("Error when closing listener")
			}
			return
		}
	}
}

// Stop gracefully terminates all the connections to the Server
// and all the goroutines it spun up.
func (s *Server) Stop() {
	s.stop <- struct{}{}
}

// Acceptor listens for connections on the given listener using a separate goroutine
// and sends them through connChan.
func Acceptor(l net.Listener) (connChan <-chan net.Conn, errChan <-chan error) {
	cc := make(chan net.Conn, 1)
	ec := make(chan error, 1)

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				ec <- err
				return
			}
			cc <- conn
		}
	}()
	return cc, ec
}
