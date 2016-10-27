package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"net"
	"time"

	"github.com/dogestreet/zero/stratum"
)

// LineReadWriter reads lines from a net.Conn with a specified timeout.
type LineReadWriter struct {
	conn    net.Conn
	scanner *bufio.Scanner
}

// NewLineReadWriter creates a new line reader.
func NewLineReadWriter(conn net.Conn) *LineReadWriter {
	return &LineReadWriter{
		conn:    conn,
		scanner: bufio.NewScanner(conn),
	}
}

// ReadStratumTimed reads a line of Stratum RPC from the connection.
func (lrw *LineReadWriter) ReadStratumTimed(deadline time.Time) (stratum.Request, error) {
	if err := lrw.conn.SetReadDeadline(deadline); err != nil {
		return nil, err
	}

	if !lrw.scanner.Scan() {
		// Scanner will set error to nil on EOF.
		if lrw.scanner.Err() == nil {
			return nil, io.EOF
		}

		return nil, lrw.scanner.Err()
	}

	val, err := stratum.Parse(lrw.scanner.Bytes())
	if err != nil {
		return nil, err
	}

	return val, nil
}

// WaitForType waits until the client sends the expected message.
func (lrw *LineReadWriter) WaitForType(tp stratum.RequestType, deadline time.Time) (stratum.Request, error) {
	req, err := lrw.ReadStratumTimed(deadline)
	if err != nil {
		return nil, err
	}

	if req.Type() != tp {
		return nil, errors.New("unexpected message")
	}

	return req, nil
}

// WriteStratumTimed writes a Stratum response.
func (lrw *LineReadWriter) WriteStratumTimed(resp stratum.Response, deadline time.Time) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	data = append(data, byte('\n'))

	if err := lrw.conn.SetWriteDeadline(deadline); err != nil {
		return err
	}

	if _, err := lrw.conn.Write(data); err != nil {
		return err
	}

	return nil
}

// WriteStratumRaw writes a raw Stratum response.
func (lrw *LineReadWriter) WriteStratumRaw(data []byte, deadline time.Time) error {
	if err := lrw.conn.SetWriteDeadline(deadline); err != nil {
		return err
	}

	_, err := lrw.conn.Write(data)
	if err != nil {
		return err
	}

	return nil
}
