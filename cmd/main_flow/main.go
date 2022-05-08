package main

import (
	"fmt"
	"net"
	"time"

	// "example.com/storage"
	log "github.com/Sirupsen/logrus"
	"github.com/khiemm/listener/pkg/storage"
)

func init() {
	log.WithFields(log.Fields{
		"animal": "walrus",
	}).Info("A walrus appears")
}

func main() {
	err := storage.Connect()
	if err != nil {
		panic(err)
	}

	// NOTE: must Upper case first letter to allow export function
	albums, err := storage.AlbumsByArtist("John Coltrane")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Albums found: %v\n", albums)
}

func startHealthCheckServer(addr string) (err error) {
	srv, err := net.Listen("tcp", addr)
	if err != nil {
		return
	}
	log.WithField("addr", addr).Info("Health Check listening")
	go func() {
		for {
			conn, err := srv.Accept()
			if err != nil {
				log.WithError(err).Error("Couldn't accept health check connection")
				continue
			}
			go healthCheckConnectionHandler(conn)
		}
	}()
	return nil
}

func healthCheckConnectionHandler(conn net.Conn) {
	log.WithField("addr", conn.RemoteAddr()).Debug("Health check connection")
	time.Sleep(time.Duration(5) * time.Second)
	err := conn.Close()
	if err != nil {
		log.WithError(err).Error("Couldn't close health check connection")
	}
}
