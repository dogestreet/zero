package main

import (
	"encoding/json"
	"log"

	"github.com/garyburd/redigo/redis"
)

// DB used to handle all the share handing.
type DB struct {
	pool *redis.Pool

	SubmitChan chan Share
}

// A Share submitted to redis.
type Share struct {
	Submitter     string  `json:"addr"`
	Difficulty    float64 `json:"diff"`
	NetDifficulty float64 `json:"net_diff"`
	Subsidy       float64 `json:"sub"`
	Host          string  `json:"host"`
	Server        string  `json:"srv"`
	Valid         bool    `json:"valid"`
}

// NewDB from redis.
func NewDB(redisHost, redisPass string) (*DB, error) {
	pool := redis.NewPool(func() (redis.Conn, error) {
		c, err := redis.Dial("tcp", redisHost)
		if err != nil {
			return nil, err
		}

		// Authenticate with the auth.
		if redisPass != "" {
			if _, err := c.Do("AUTH", redisPass); err != nil {
				c.Close()
				return nil, err
			}
		}

		return c, err
	}, 10)

	conn := pool.Get()
	defer conn.Close()

	// Test the connection
	if _, err := conn.Do("PING"); err != nil {
		return nil, err
	}

	db := DB{
		pool:       pool,
		SubmitChan: make(chan Share, 1024),
	}

	go db.serve()

	return &db, nil
}

// serve runs a loop that handles share submissions.
func (db *DB) serve() {
	for share := range db.SubmitChan {
		func() {
			conn := db.pool.Get()
			defer conn.Close()

			// Encode the share
			data, _ := json.Marshal(share)

			_, err := conn.Do("PUBLISH", "shares", data)
			if err != nil {
				log.Println("[db] ! Could not publish share: " + string(data))
			}
		}()
	}
}
