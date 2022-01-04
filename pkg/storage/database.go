// Package storage handles connection pool for MySQL
// and defines some utility interfaces.
package storage

import (
	"database/sql"
	"errors"
	"fmt"

	log "github.com/Sirupsen/logrus"
	_ "github.com/go-sql-driver/mysql" //Just registering the driver
	"gopkg.in/gorp.v1"
)

var (
	// Db is the global MySQL connection pool
	Db                      *gorp.DbMap
	ErrAlreadyConnected     = errors.New("storage: connection pools have already been initialized")
	ErrNoDatabaseConnection = errors.New("storage: couldn't connect to mysql database")
)

// var db *sql.DB

type Album struct {
	ID     int64
	Title  string
	Artist string
	Price  float32
}

// Connect initializes global connection pool. The connection info
// is loaded from config.
func Connect() (err error) {
	if Db != nil {
		return ErrAlreadyConnected
	}

	err = mysqlConnect("root", "root", "localhost:3306", "listener", 16, false)
	if err != nil {
		return
	}
	// // Capture connection properties.
	// cfg := mysql.Config{
	// 	User:   "root",
	// 	Passwd: "root",
	// 	Net:    "tcp",
	// 	Addr:   "localhost:3306",
	// 	DBName: "listener",
	// }
	// // Get a database handle.
	// db, err = sql.Open("mysql", cfg.FormatDSN())
	// if err != nil {
	// 	log.Fatal(err)
	// }

	return
}

// Disconnect drains the connection pool and releases their associated resources.
func Disconnect() (err error) {
	if Db != nil {
		err = Db.Db.Close()
		if err != nil {
			return
		}
		Db = nil
	}
	return
}

func mysqlConnect(username, password, addr, dbName string, poolSize int, trace bool) (err error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?multiStatements=true",
		username, password, addr, dbName)
	sqlDb, err := sql.Open("mysql", dsn)
	if err != nil {
		return
	}
	err = sqlDb.Ping()
	if err != nil {
		return
	}
	sqlDb.SetMaxOpenConns(poolSize)
	Db = &gorp.DbMap{Db: sqlDb, Dialect: gorp.MySQLDialect{}}
	if trace {
		Db.TraceOn("gorp:", log.StandardLogger())
	}
	return
}

// albumsByArtist queries for albums that have the specified artist name.
func AlbumsByArtist(name string) (albums []Album, err error) {
	// An albums slice to hold data from returned rows.
	// var albums []Album

	// rows, err := db.Query("SELECT * FROM album WHERE artist = ?", name)
	// if err != nil {
	// 	return nil, fmt.Errorf("albumsByArtist %q: %v", name, err)
	// }
	// defer rows.Close()
	// // Loop through rows, using Scan to assign column data to struct fields.
	// for rows.Next() {
	// 	var alb Album
	// 	if err := rows.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
	// 		return nil, fmt.Errorf("albumsByArtist %q: %v", name, err)
	// 	}
	// 	albums = append(albums, alb)
	// }
	// if err := rows.Err(); err != nil {
	// 	return nil, fmt.Errorf("albumsByArtist %q: %v", name, err)
	// }
	// return albums, nil

	// rs = make([]*Album, 0)
	_, err = Db.Select(&albums, "SELECT * FROM album WHERE artist = ?", name)
	if gorp.NonFatalError(err) {
		err = nil
	}
	return

}

// addAlbum adds the specified album to the database,
// returning the album ID of the new entry
func addAlbum(alb Album) (int64, error) {
	result, err := Db.Exec("INSERT INTO album (title, artist, price) VALUES (?, ?, ?)", alb.Title, alb.Artist, alb.Price)
	if err != nil {
		return 0, fmt.Errorf("addAlbum: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("addAlbum: %v", err)
	}
	return id, nil
}
