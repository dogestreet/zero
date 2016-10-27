package main

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dogestreet/zero/rpc"
	"github.com/dogestreet/zero/stratum"
)

type (
	// ZeroClientID is the type for the client ID.
	ZeroClientID uint64

	// Zero mining server.
	Zero struct {
		idCount  uint64
		jobCount uint64

		clients struct {
			m map[ZeroClientID]*ZeroClient
			w *Work
			sync.RWMutex
		}

		// The server ID part of the nonce.
		NoncePart1a [8]byte

		// Configuration.
		Config Config

		// RPC client.
		daemon *rpc.Client
	}

	// A ZeroClient represents a miner.
	ZeroClient struct {
		ID       ZeroClientID
		WorkChan chan *Work

		z    *Zero
		conn net.Conn
		lrw  *LineReadWriter

		closeOnce sync.Once
	}

	// Work is a piece of the block that being worked on by the pool server.
	Work struct {
		stratum.ResponseNotify

		Height   int
		Target   stratum.Uint256
		RawBlock []byte
		SWork    []byte
		N        int
		K        int
		At       time.Time

		z *Zero
	}

	// workTemplate used with the patched RPC call.
	workTemplate struct {
		Height int `json:"height"`
		N      int `json:"n"`
		K      int `json:"k"`

		Header struct {
			Version        uint32 `json:"version"`
			HashPrevBlock  string `json:"prevblock"`
			HashMerkleRoot string `json:"merkleroot"`
			Reserved       string `json:"reserved"`
			Time           uint32 `json:"time"`
			Bits           string `json:"bits"`
		} `json:"header"`

		Raw string `json:"raw"`
	}
)

// Pool timeout settings
const (
	InitTimeout = 10 * time.Second
	AuthTimeout = 10 * time.Second

	WriteTimeout = 15 * time.Second

	InactivityTimeout = 3 * time.Minute
	KeepAliveInterval = 30 * time.Second
)

// NewZero creates a new instance of the mining server.
func NewZero(cfg Config) (*Zero, error) {
	z := Zero{
		idCount: 0,

		clients: struct {
			m map[ZeroClientID]*ZeroClient
			w *Work
			sync.RWMutex
		}{
			m: make(map[ZeroClientID]*ZeroClient),
		},

		Config: cfg,
		daemon: nil,
	}

	if _, err := rand.Read(z.NoncePart1a[:]); err != nil {
		return nil, err
	}

	var err error
	if z.daemon, err = rpc.NewClient(cfg.DaemonURL); err != nil {
		return nil, err
	}

	// Try and get work
	if err := z.GetWork(); err != nil {
		return nil, err
	}

	return &z, nil
}

// ShareStatus defines the status of a share.
type ShareStatus string

// ShareStatus constants.
const (
	ShareInvalid ShareStatus = "invalid"
	ShareOK      ShareStatus = "ok"
	ShareBlock   ShareStatus = "block"
)

// Check the proof of work.
// Returns the difficulty, return error if invalid.
func (w *Work) Check(nTime uint32, noncePart1, noncePart2, solution []byte, shareTarget stratum.Uint256, dead bool) ShareStatus {

	buffer := BuildBlockHeader(w.Version, w.HashPrevBlock[:], w.HashMerkleRoot[:], w.HashReserved[:], w.NTime, w.NBits, noncePart1, noncePart2)

	result := Validate(w.N, w.K, buffer.Bytes(), solution, shareTarget, w.Target)
	if result == ShareBlock {
		if dead {
			return result
		}

		_, _ = buffer.Write([]byte{0xfd, 0x40, 0x05})
		_, _ = buffer.Write(solution)

		// The buffer now contains the completed block header
		// Fill in the rest of the block
		_, _ = buffer.Write(w.RawBlock[buffer.Len():])
		go w.z.SubmitBlock(buffer.Bytes())
	}

	return result
}

func reverse(b []byte) {
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
}

// GetWork from the coin daemon.
func (z *Zero) GetWork() error {
	var workTemplate workTemplate
	if err := z.daemon.Call("zero_getblocktemplate", []interface{}{z.Config.CoinbaseAddress}, &workTemplate); err != nil {
		return err
	}

	w, err := makeWork(atomic.AddUint64(&z.jobCount, 1), workTemplate)
	if err != nil {
		return err
	}

	// SWork has a newline on the end
	log.Print("[server] got new work: ", string(w.SWork))

	z.SendWorkAll(w)
	return nil
}

func makeWork(count uint64, workTemplate workTemplate) (*Work, error) {
	prevBlock, err := stratum.HexToUint256(workTemplate.Header.HashPrevBlock)
	if err != nil {
		return nil, err
	}
	reverse(prevBlock[:])

	merkleRoot, err := stratum.HexToUint256(workTemplate.Header.HashMerkleRoot)
	if err != nil {
		return nil, err
	}

	reserved, err := stratum.HexToUint256(workTemplate.Header.Reserved)
	if err != nil {
		return nil, err
	}

	bits, err := stratum.HexToUint32(workTemplate.Header.Bits)
	if err != nil {
		return nil, err
	}

	// Reverse the version
	workTemplate.Header.Version = (uint32(workTemplate.Header.Version)&0xff000000)>>24 |
		(uint32(workTemplate.Header.Version)&0x00ff0000)>>16 |
		(uint32(workTemplate.Header.Version)&0x0000ff00)<<16 |
		(uint32(workTemplate.Header.Version)&0x000000ff)<<24

	w := Work{
		ResponseNotify: stratum.ResponseNotify{
			Job:            fmt.Sprint(count),
			Version:        workTemplate.Header.Version,
			HashPrevBlock:  prevBlock,
			HashMerkleRoot: merkleRoot,
			HashReserved:   reserved,
			NTime:          workTemplate.Header.Time,
			NBits:          bits,
			CleanJobs:      true,
		},

		Height:   workTemplate.Height,
		RawBlock: nil,
		SWork:    nil,
		N:        workTemplate.N,
		K:        workTemplate.K,
		At:       time.Now(),
	}

	w.SWork, err = json.Marshal(w.ResponseNotify)
	if err != nil {
		return nil, err
	}
	w.SWork = append(w.SWork, '\n')

	w.RawBlock, err = hex.DecodeString(workTemplate.Raw)
	if err != nil {
		return nil, err
	}

	if len(w.RawBlock) < 4+32+32+32+4+4+32+1334 {
		return nil, errors.New("invalid block template")
	}

	w.Target, err = CompactToTarget(w.NBits)
	if err != nil {
		return nil, err
	}

	return &w, nil
}

// Handle a new client connection.
// This handler is executed in a goroutine.
func (z *Zero) Handle(conn *net.TCPConn) error {
	log.Println("[server] new connection from", conn.RemoteAddr())

	if err := conn.SetKeepAlive(true); err != nil {
		return err
	}

	if err := conn.SetKeepAlivePeriod(KeepAliveInterval); err != nil {
		return err
	}

	client := ZeroClient{
		ID:       ZeroClientID(atomic.AddUint64(&z.idCount, 1)),
		WorkChan: make(chan *Work, 32),

		z:    z,
		conn: conn,
		lrw:  NewLineReadWriter(conn),
	}

	return client.Serve()
}

// Subscribe a client onto work notifications.
func (z *Zero) Subscribe(zc *ZeroClient) {
	z.clients.Lock()
	z.clients.m[zc.ID] = zc

	// This should never block
	select {
	case zc.WorkChan <- z.clients.w:
	default:
	}

	z.clients.Unlock()
}

// Unsubscribe removes a client from work notifications.
func (z *Zero) Unsubscribe(zc *ZeroClient) {
	z.clients.Lock()
	delete(z.clients.m, zc.ID)
	_ = zc.Close()
	z.clients.Unlock()
}

// BlockNotify is used to notify the server that a new block has been found and that new work should be sent.
func (z *Zero) BlockNotify() error {
	log.Println("[server]", "new block detected")
	return z.GetWork()
}

// SubmitBlock to the RPC client.
// NOTE: this function is to be called via goroutine.
func (z *Zero) SubmitBlock(rawBlock []byte) (err error) {
	block := hex.EncodeToString(rawBlock)
	var status string

	if err := func() (err error) {
		if err := z.daemon.Call("submitblock", []interface{}{block}, &status); err != nil {
			return err
		}

		if status != "valid?" {
			return errors.New("not valid: " + status)
		}

		return nil
	}(); err != nil {
		log.Printf("Block '%v' submission failed: %v\n", block, err)
	}

	// Either way, get new work
	return z.GetWork()
}

// SendWorkAll sends new work to every client.
func (z *Zero) SendWorkAll(w *Work) {
	var dead []*ZeroClient

	z.clients.Lock()
	z.clients.w = w
	for _, zc := range z.clients.m {
		select {
		case zc.WorkChan <- w:
		default:
			dead = append(dead, zc)
			delete(z.clients.m, zc.ID)
		}
	}
	z.clients.Unlock()

	for _, zc := range dead {
		zc.Close()
	}
}

// Serve runs the client.
func (zc *ZeroClient) Serve() (err error) {
	log.Printf("[client %v %v] serving\n", zc.ID, zc.conn.RemoteAddr())
	defer func() {
		log.Printf("[client %v %v] disconnected with error: %v\n", zc.ID, zc.conn.RemoteAddr(), err)
	}()

	defer zc.conn.Close()

	var noncePart1 stratum.Uint128

	{ // Handle subscription
		subscribe, err := zc.lrw.WaitForType(stratum.Subscribe, time.Now().Add(InitTimeout))
		if err != nil {
			return err
		}

		sub := subscribe.(stratum.RequestSubscribe)
		if len(sub.Params) == 1 && sub.Params[0] == zc.z.Config.UpdateKey {
			zc.z.BlockNotify()
		}

		// Write in the server ID
		copy(noncePart1[:], zc.z.NoncePart1a[:])

		// Write in the client ID
		binary.PutUvarint(noncePart1[8:], uint64(zc.ID))

		if err := zc.lrw.WriteStratumTimed(stratum.ResponseSubscribeReply{
			ID:         sub.ID,
			Session:    "",
			NoncePart1: noncePart1,
		}, time.Now().Add(WriteTimeout)); err != nil {
			return err
		}
	}

	var username string

	{ // Handle auth
		auth, err := zc.lrw.WaitForType(stratum.Authorise, time.Now().Add(AuthTimeout))
		if err != nil {
			return err
		}

		authReq := auth.(stratum.RequestAuthorise)
		if ok, testnet := IsValidAddress(authReq.Username); !ok || zc.z.Config.Testnet != testnet {
			if !ok {
				if err := zc.lrw.WriteStratumTimed(stratum.ResponseGeneral{
					ID:    authReq.ID,
					Error: "Please double check your payout address, it appears to be invalid",
				}, time.Now().Add(WriteTimeout)); err != nil {
					return err
				}

				return errors.New("invalid payout address")
			}

			if zc.z.Config.Testnet != testnet {
				msg := "Please double check your payout address, it appears to be a testnet address (expecting a mainnet address)"
				if zc.z.Config.Testnet {
					msg = "Please double check your payout address, it appears to be a mainnet address (expecting a testnet address)"
				}

				if err := zc.lrw.WriteStratumTimed(stratum.ResponseGeneral{
					ID:    authReq.ID,
					Error: msg,
				}, time.Now().Add(WriteTimeout)); err != nil {
					return err
				}

				return errors.New("payout address testnet mismatch")
			}
		}

		username = authReq.Username
	}

	log.Printf("[client %v %v] authed with '%v'\n", zc.ID, zc.conn.RemoteAddr(), username)

	// Subscribe onto the mining notifications
	zc.z.Subscribe(zc)
	defer zc.z.Unsubscribe(zc)

	// Channel for reading input
	requestChan := make(chan stratum.Request, 64)
	go func() error {
		defer close(requestChan)

		for {
			req, err := zc.lrw.ReadStratumTimed(time.Now().Add(InactivityTimeout))
			if err != nil {
				log.Println("err", err)
				return err
			}

			requestChan <- req
		}
	}()

	// Vardiff channels and tickers.
	vardiffRetargetTime := zc.z.Config.VardiffRetargetTime
	vardiffRetargetTicker := time.NewTicker(time.Duration(vardiffRetargetTime) * time.Second)
	defer vardiffRetargetTicker.Stop()

	// Statistics for vardiff
	vardiffDifficulty := Difficulty(zc.z.Config.VardiffInitial)
	vardiffTarget := zc.z.Config.VardiffTarget
	vardiffAllowance := zc.z.Config.VardiffAllowance
	vardiffMin := Difficulty(zc.z.Config.VardiffMin)
	vardiffForceChan := make(chan bool)
	vardiffLastRun := time.Now()
	vardiffLastSubmit := time.Now()
	vardiffShares := 0
	vardiffSharesDead := 0

	// Vardiff/Kick thread.
	vardiff := func() error {
		elapsedTime := time.Now().Sub(vardiffLastRun)
		idleTime := time.Now().Sub(vardiffLastSubmit)

		estimatedHashPerSecond := EstimateHashPerSecond(vardiffShares, vardiffDifficulty, elapsedTime)
		currentHashPerSecond := EstimateHashPerSecond(vardiffTarget, vardiffDifficulty, elapsedTime)

		log.Println("vardiff adjustment", "est:", estimatedHashPerSecond, "curr:", currentHashPerSecond)

		if vardiffShares > vardiffTarget && float64(vardiffSharesDead)/float64(vardiffShares) > 0.9 {
			// TODO: logging
			return errors.New("too many dead shares")
		} else if idleTime >= InactivityTimeout {
			return errors.New("client idle")
		} else if currentHashPerSecond*(1-vardiffAllowance) > estimatedHashPerSecond ||
			estimatedHashPerSecond > currentHashPerSecond*(1+vardiffAllowance) {
			vardiffDifficulty = FindDifficulty(estimatedHashPerSecond, vardiffTarget, vardiffRetargetTime)

			if vardiffDifficulty < vardiffMin {
				vardiffDifficulty = Difficulty(vardiffMin)
			}

			// Send current difficulty
			if err := zc.lrw.WriteStratumTimed(stratum.ResponseSetTarget{
				Target: vardiffDifficulty.ToTarget(),
			}, time.Now().Add(WriteTimeout)); err != nil {
				return err
			}
		}

		// Reset our counters
		vardiffLastRun = time.Now()
		vardiffShares = 0
		vardiffSharesDead = 0
		return nil
	}

	// Vardiff difficulty
	diff := Difficulty(zc.z.Config.VardiffInitial)

	var currWork, prevWork *Work

	for {
		select {
		case <-vardiffRetargetTicker.C:
			if err := vardiff(); err != nil {
				return err
			}

		case <-vardiffForceChan:
			if err := vardiff(); err != nil {
				return err
			}

		case req, ok := <-requestChan:
			if !ok {
				return errors.New("read chan closed")
			}

			switch req.(type) {
			case stratum.RequestSubmit:
				req := req.(stratum.RequestSubmit)

				log.Printf("%x %x %x %x\n", req.NTime, noncePart1, req.NoncePart2, req.Solution)

				// Calculate the target
				shareStatus := currWork.Check(req.NTime, noncePart1[:], req.NoncePart2[:], req.Solution, diff.ToTarget(), false)
				if shareStatus == ShareInvalid {
					if prevWork != nil {
						// Allow stale shares in the last N seconds.
						if time.Now().Sub(prevWork.At) < 3*time.Second {
							// Check if this was a stale share.
							res := prevWork.Check(req.NTime, noncePart1[:], req.NoncePart2[:], req.Solution, diff.ToTarget(), prevWork.Height != currWork.Height)
							if res != ShareInvalid {
								// Accept it anyway
								shareStatus = ShareOK
							}
						}
					}
				}

				if shareStatus == ShareInvalid {
					vardiffSharesDead++
				}

				vardiffShares++
				vardiffLastSubmit = time.Now()

				// TODO: redis publish, bad share bans
				// TODO: logger for fail2ban
				log.Println("share", req.NoncePart2, shareStatus)

			case stratum.RequestSuggestTarget:
				// ignored, we use our own vardiff
			default:
				// impossible
			}

		case work, ok := <-zc.WorkChan:
			if !ok {
				return errors.New("work chan closed")
			}

			// Send current difficulty
			if err := zc.lrw.WriteStratumTimed(stratum.ResponseSetTarget{
				Target: vardiffDifficulty.ToTarget(),
			}, time.Now().Add(WriteTimeout)); err != nil {
				return err
			}

			if err := zc.lrw.WriteStratumRaw(work.SWork, time.Now().Add(WriteTimeout)); err != nil {
				return err
			}

			prevWork, currWork = currWork, work
		}
	}
}

// Close the client.
func (zc *ZeroClient) Close() error {
	zc.closeOnce.Do(func() {
		close(zc.WorkChan)
	})

	return zc.conn.Close()
}
