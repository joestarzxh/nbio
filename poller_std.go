// Copyright 2020 lesismal. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build windows

package nbio

import (
	"bytes"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/lesismal/nbio/logging"
)

type poller struct {
	mux sync.Mutex

	g *Gopher

	index int

	ReadBuffer []byte

	pollType   string
	isListener bool
	listener   net.Listener
	shutdown   bool

	chStop chan struct{}
}

func (p *poller) accept() error {
	conn, err := p.listener.Accept()
	if err != nil {
		return err
	}

	c := newConn(conn)
	o := p.g.pollers[c.Hash()%len(p.g.pollers)]
	o.addConn(c)

	return nil
}

func (p *poller) readConn(c *Conn) {
	if p.g.onEvent == nil {
		for {
			buffer := p.g.borrow(c)
			n, err := c.Read(buffer)
			if n > 0 {
				p.g.onData(c, buffer[:n])
			}
			p.g.payback(c, buffer)
			if err != nil {
				c.Close()
				return
			}
		}
	} else {
		if c.Buffer == nil {
			c.Buffer = bytes.NewBuffer(make([]byte, 4096))
		}
		for {
			buffer := p.g.borrow(c)
			n, err := c.conn.Read(buffer)
			if n > 0 {
				c.mux.Lock()
				c.Buffer.Write(buffer)
				c.mux.Unlock()
				p.g.onEvent(c, PollerEventRead)
			}
			p.g.payback(c, buffer)
			if err != nil {
				p.g.onEvent(c, PollerEventError)
				c.CloseWithError(err)
				return
			}
		}
	}
}

func (p *poller) addConn(c *Conn) error {
	c.g = p.g
	p.g.mux.Lock()
	p.g.connsStd[c] = struct{}{}
	p.g.mux.Unlock()
	p.g.onOpen(c)
	go p.readConn(c)

	return nil
}

func (p *poller) deleteConn(c *Conn) {
	p.g.mux.Lock()
	delete(p.g.connsStd, c)
	p.g.mux.Unlock()
	p.g.onClose(c, c.closeErr)
}

func (p *poller) start() {
	if p.g.lockListener {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}
	defer p.g.Done()

	logging.Debug("Poller[%v_%v_%v] start", p.g.Name, p.pollType, p.index)
	defer logging.Debug("Poller[%v_%v_%v] stopped", p.g.Name, p.pollType, p.index)

	if p.isListener {
		var err error
		p.shutdown = false
		for !p.shutdown {
			err = p.accept()
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Temporary() {
					logging.Error("Poller[%v_%v_%v] Accept failed: temporary error, retrying...", p.g.Name, p.pollType, p.index)
					time.Sleep(time.Second / 20)
				} else {
					logging.Error("Poller[%v_%v_%v] Accept failed: %v, exit...", p.g.Name, p.pollType, p.index, err)
					break
				}
			}

		}
	}
	<-p.chStop
}

func (p *poller) stop() {
	logging.Debug("Poller[%v_%v_%v] stop...", p.g.Name, p.pollType, p.index)
	p.shutdown = true
	if p.isListener {
		p.listener.Close()
	}
	close(p.chStop)
}

func newPoller(g *Gopher, isListener bool, index int) (*poller, error) {
	p := &poller{
		g:          g,
		index:      index,
		isListener: isListener,
		chStop:     make(chan struct{}),
	}

	if isListener {
		var err error
		var addr = g.addrs[index%len(g.addrs)]
		p.listener, err = net.Listen(g.network, addr)
		if err != nil {
			return nil, err
		}
		p.pollType = "LISTENER"
	} else {
		p.pollType = "POLLER"
	}

	return p, nil
}
