package main

import (
	"encoding/json"
	"log"
	"net"
	"os"

	"net/http"
	_ "net/http/pprof"
)

// Config for the pool server.
type Config struct {
	// CoinbaseAddress is the address used in the coinbase.
	CoinbaseAddress string `json:"coinbase_address"`

	// Host to listen on.
	Host string `json:"host"`

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

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalln("Could not load configuration:", err)
	}

	// Create the mining server
	z, err := NewZero(cfg)
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
