package main

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"time"

	"net/http"
	_ "net/http/pprof"
)

// Config for the pool server.
type Config struct {
	// CoinbaseAddress is the address used in the coinbase.
	CoinbaseAddress string `json:"coinbase_address"`

	// Name of this server.
	Name string `json:"name"`

	// Host to listen on.
	Host string `json:"host"`

	// RedisHost to connect to.
	RedisHost string `json:"redis_host"`

	// RedisPass for the server.
	RedisPassword string `json:"redis_password"`

	// Host for pprof.
	PProfHost string `json:"pprof_host"`

	// Daemon url.
	DaemonURL string `json:"daemon_url"`

	// VardiffRetargetTime (seconds).
	VardiffRetargetTime int `json:"vardiff_retarget_time"`

	// VardiffTarget (Target shares every VardiffRetargetTime).
	VardiffTarget int `json:"vardiff_target"`

	// VardiffAllowance (How much under/over target is allowed).
	VardiffAllowance float64 `json:"vardiff_allowance"`

	// VardiffMin (Minimum difficulty).
	VardiffMin float64 `json:"vardiff_min"`

	// VardiffInitial (Initial difficulty).
	VardiffInitial float64 `json:"vardiff_initial"`

	// VardiffShares (Number of shared submitted before an adjustment is forced).
	VardiffShares int `json:"vardiff_shares"`

	// Testnet or not.
	Testnet bool `json:"testnet"`

	// UpdateKey used to notify the server of updates.
	UpdateKey string `json:"update_key"`
}

func loadConfig() (cfg Config, err error) {
	configPath := os.Getenv("CONFIG")
	if configPath == "" {
		configPath = "config.json"
	}

	file, err := os.Open(configPath)
	if err != nil {
		return cfg, err
	}

	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return cfg, err
	}

	// Allow the HOST environmental variable to override the host setting
	cfg.Host = os.Getenv("HOST")
	if cfg.Host == "" {
		cfg.Host = "0.0.0.0:5555"
	}

	if cfg.PProfHost == "" {
		cfg.PProfHost = "localhost:5556"
	}

	return cfg, nil
}

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)

	if len(os.Args) > 2 && os.Args[1] == "notify" {
		conn, err := net.Dial("tcp", os.Args[2])
		if err != nil {
			log.Fatalln("Could not connect to server", err)
		}
		defer conn.Close()

		if err := conn.SetWriteDeadline(time.Now().Add(WriteTimeout)); err != nil {
			log.Fatalln(err)
		}

		if _, err := conn.Write([]byte(`{"id": 1, "method": "mining.subscribe", "params": ["` + os.Args[3] + `","` + os.Args[4] + `"]}` + "\n")); err != nil {
			log.Fatalln(err)
		}

		var buf [1]byte
		if _, err := conn.Read(buf[:]); err != nil {
			// Wait for EOF
		}

		return
	}

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalln("Could not load configuration:", err)
	}

	db, err := NewDB(cfg.RedisHost, cfg.RedisPassword)
	if err != nil {
		log.Fatalln("Could not connect to DB", err)
	}

	// Create the mining server
	z, err := NewZero(cfg, db)
	if err != nil {
		log.Fatalln("Could not start server", err)
	}

	// Enable profiling
	go func() {
		log.Println("Listening on: http://"+cfg.PProfHost, "(pprof)")
		log.Println(http.ListenAndServe(cfg.PProfHost, nil))
	}()

	// Set up the tcp server for stratum
	addr, err := net.ResolveTCPAddr("tcp", cfg.Host)
	if err != nil {
		log.Fatalln("Could not resolve address:", err)
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatalln("Could not create listener:", err)
	}

	log.Println("Listening on:", addr)
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Fatalln("Failed to accept socket:", err)
		}

		go z.Handle(conn)
	}
}
