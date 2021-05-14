// Copyright 2020 lesismal. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package nbio

import (
	"net"
	"time"

	"github.com/lesismal/nbio/loging"
)

// Dial wraps net.Dial
func Dial(network string, address string) (*Conn, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		loging.Error("Dial failed: %v", err)
		return nil, err
	}
	return NBConn(conn)
}

// DialTimeout wraps net.DialTimeout
func DialTimeout(network string, address string, timeout time.Duration) (*Conn, error) {
	conn, err := net.DialTimeout(network, address, timeout)
	if err != nil {
		return nil, err
	}
	return NBConn(conn)
}
