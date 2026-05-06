package connection

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"

	"infinispan.org/go-client/internal/codec"
	"infinispan.org/go-client/internal/operation"
)

type pendingEntry struct {
	op        operation.Operation
	ch        chan *response
	cacheName string
}

type response struct {
	value any
	err   error
}

type TopologyCallback func(update *codec.TopologyUpdate, cacheName string)

type ListenerEntry struct {
	Ch        chan<- *operation.CacheEntryEvent
	CustomCh  chan<- []byte
	CounterCh chan<- *operation.CounterEvent
}

type Conn struct {
	addr               string
	netConn            net.Conn
	reader             *bufio.Reader
	writeMu            sync.Mutex
	pending            sync.Map
	listeners          sync.Map // hex(listenerID) → *ListenerEntry
	nextMsgID          atomic.Int64
	topologyID         atomic.Int32
	closed             atomic.Bool
	draining           atomic.Bool
	onTopology         TopologyCallback
	clientIntelligence byte
	protocolVersion    byte
	logger             *slog.Logger
	done               chan struct{}
}

func Dial(ctx context.Context, addr string, onTopology TopologyCallback, clientIntelligence byte, protocolVersion byte, logger *slog.Logger) (*Conn, error) {
	var d net.Dialer
	netConn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	return NewConn(netConn, addr, onTopology, clientIntelligence, protocolVersion, logger), nil
}

func NewConn(netConn net.Conn, addr string, onTopology TopologyCallback, clientIntelligence byte, protocolVersion byte, logger *slog.Logger) *Conn {
	c := &Conn{
		addr:               addr,
		netConn:            netConn,
		reader:             bufio.NewReaderSize(netConn, 64*1024),
		onTopology:         onTopology,
		clientIntelligence: clientIntelligence,
		protocolVersion:    protocolVersion,
		logger:             logger,
		done:               make(chan struct{}),
	}
	go c.readLoop()
	return c
}

func (c *Conn) Addr() string { return c.addr }

func (c *Conn) Execute(ctx context.Context, op operation.Operation) (any, error) {
	if c.closed.Load() || c.draining.Load() {
		return nil, errors.New("connection closed")
	}

	msgID := c.nextMsgID.Add(1)
	ch := make(chan *response, 1)
	c.pending.Store(msgID, &pendingEntry{op: op, ch: ch, cacheName: string(op.CacheName())})

	if err := c.writeRequest(msgID, op); err != nil {
		c.pending.Delete(msgID)
		return nil, fmt.Errorf("write request: %w", err)
	}

	select {
	case resp := <-ch:
		return resp.value, resp.err
	case <-ctx.Done():
		c.pending.Delete(msgID)
		return nil, ctx.Err()
	case <-c.done:
		return nil, errors.New("connection closed")
	}
}

func (c *Conn) writeRequest(msgID int64, op operation.Operation) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	var buf bytes.Buffer
	h := &codec.RequestHeader{
		MessageID:          msgID,
		OpCode:             op.RequestOpCode(),
		Version:            c.protocolVersion,
		CacheName:          op.CacheName(),
		Flags:              op.Flags(),
		ClientIntelligence: c.clientIntelligence,
		TopologyID:         c.topologyID.Load(),
		KeyMediaType:       op.KeyMediaType(),
		ValueMediaType:     op.ValueMediaType(),
	}
	if err := codec.WriteRequestHeader(&buf, h); err != nil {
		return err
	}
	if err := op.WriteBody(&buf); err != nil {
		return err
	}
	_, err := c.netConn.Write(buf.Bytes())
	return err
}

func (c *Conn) readLoop() {
	defer close(c.done)
	for {
		header, err := codec.ReadResponseHeader(c.reader, c.clientIntelligence)
		if err != nil {
			if !c.closed.Load() {
				c.logger.Warn("read loop error", "addr", c.addr, "err", err)
			}
			c.closeAllPending(err)
			return
		}

		if codec.IsEvent(header.OpCode) {
			c.handleEvent(header)
			continue
		}

		val, ok := c.pending.LoadAndDelete(header.MessageID)
		if !ok {
			if header.TopologyUpdate != nil && c.onTopology != nil {
				c.logger.Debug("topology update on orphan message", "messageId", header.MessageID, "topoID", header.TopologyUpdate.ID, "servers", len(header.TopologyUpdate.Servers))
				c.topologyID.Store(header.TopologyUpdate.ID)
				c.onTopology(header.TopologyUpdate, "")
			}
			c.logger.Warn("no pending request for message", "messageId", header.MessageID)
			continue
		}
		entry := val.(*pendingEntry)

		if header.TopologyUpdate != nil && c.onTopology != nil {
			addrs := make([]string, len(header.TopologyUpdate.Servers))
			for i, s := range header.TopologyUpdate.Servers {
				addrs[i] = s.Addr()
			}
			c.logger.Debug("topology update received", "messageId", header.MessageID, "topoID", header.TopologyUpdate.ID, "servers", addrs, "hashVersion", header.TopologyUpdate.HashFunctionVersion, "numSegments", header.TopologyUpdate.NumSegments)
			c.topologyID.Store(header.TopologyUpdate.ID)
			c.onTopology(header.TopologyUpdate, entry.cacheName)
		}

		if codec.IsError(header.Status) {
			msg, _ := codec.ReadErrorMessage(c.reader)
			entry.ch <- &response{err: &ServerError{
				Status:    header.Status,
				MessageID: header.MessageID,
				Message:   msg,
			}}
			continue
		}

		result, err := entry.op.DecodeResponse(header.Status, c.reader)
		entry.ch <- &response{value: result, err: err}

		if c.draining.Load() && !c.hasPending() && !c.hasListeners() {
			c.Close()
			return
		}
	}
}

func (c *Conn) hasPending() bool {
	has := false
	c.pending.Range(func(_, _ any) bool {
		has = true
		return false
	})
	return has
}

func (c *Conn) hasListeners() bool {
	has := false
	c.listeners.Range(func(_, _ any) bool {
		has = true
		return false
	})
	return has
}

func (c *Conn) closeAllPending(err error) {
	c.pending.Range(func(key, value any) bool {
		entry := value.(*pendingEntry)
		entry.ch <- &response{err: err}
		c.pending.Delete(key)
		return true
	})
	c.listeners.Range(func(key, value any) bool {
		entry := value.(*ListenerEntry)
		if entry.Ch != nil {
			close(entry.Ch)
		}
		if entry.CustomCh != nil {
			close(entry.CustomCh)
		}
		if entry.CounterCh != nil {
			close(entry.CounterCh)
		}
		c.listeners.Delete(key)
		return true
	})
}

func (c *Conn) handleEvent(header *codec.ResponseHeader) {
	if header.OpCode == codec.OpCounterEvent {
		c.handleCounterEvent()
		return
	}

	listenerID, err := operation.ReadEventListenerID(c.reader)
	if err != nil {
		c.logger.Warn("read listener ID from event", "err", err)
		return
	}
	key := hex.EncodeToString(listenerID)
	raw, ok := c.listeners.Load(key)
	if !ok {
		c.logger.Warn("event for unknown listener", "listenerID", key, "opCode", header.OpCode)
		operation.SkipEventBody(header.OpCode, c.reader)
		return
	}
	entry := raw.(*ListenerEntry)
	decoded, err := operation.DecodeEventBody(header.OpCode, c.reader)
	if err != nil {
		c.logger.Warn("decode event error", "err", err)
		return
	}
	if decoded == nil {
		return
	}

	switch ev := decoded.(type) {
	case *operation.CacheEntryEvent:
		if entry.Ch != nil {
			select {
			case entry.Ch <- ev:
			default:
				c.logger.Warn("listener event channel full, dropping event")
			}
		}
	case *operation.CustomEvent:
		if entry.CustomCh != nil {
			select {
			case entry.CustomCh <- ev.Data:
			default:
				c.logger.Warn("custom event channel full, dropping event")
			}
		}
	}
}

func (c *Conn) handleCounterEvent() {
	ev, listenerID, err := operation.DecodeCounterEventBody(c.reader)
	if err != nil {
		c.logger.Warn("decode counter event error", "err", err)
		return
	}
	key := hex.EncodeToString(listenerID)
	raw, ok := c.listeners.Load(key)
	if !ok {
		c.logger.Warn("counter event for unknown listener", "listenerID", key)
		return
	}
	entry := raw.(*ListenerEntry)
	if entry.CounterCh != nil {
		select {
		case entry.CounterCh <- ev:
		default:
			c.logger.Warn("counter event channel full, dropping event")
		}
	}
}

func (c *Conn) ExecuteListener(ctx context.Context, op operation.Operation, listenerID []byte, entry *ListenerEntry) error {
	if c.closed.Load() || c.draining.Load() {
		return errors.New("connection closed")
	}

	key := hex.EncodeToString(listenerID)
	c.listeners.Store(key, entry)

	msgID := c.nextMsgID.Add(1)
	ch := make(chan *response, 1)
	c.pending.Store(msgID, &pendingEntry{op: op, ch: ch, cacheName: string(op.CacheName())})

	if err := c.writeRequest(msgID, op); err != nil {
		c.pending.Delete(msgID)
		c.listeners.Delete(key)
		return fmt.Errorf("write request: %w", err)
	}

	select {
	case resp := <-ch:
		if resp.err != nil {
			c.listeners.Delete(key)
			return resp.err
		}
		return nil
	case <-ctx.Done():
		c.pending.Delete(msgID)
		c.listeners.Delete(key)
		return ctx.Err()
	case <-c.done:
		c.listeners.Delete(key)
		return errors.New("connection closed")
	}
}

func (c *Conn) UnregisterListener(listenerKey string) {
	c.listeners.Delete(listenerKey)
}

// Drain marks the connection so that it will close itself once all pending
// operations have been serviced by the readLoop.  New writes are rejected
// immediately (Execute checks the draining flag).
func (c *Conn) Drain() {
	c.draining.Store(true)
}

func (c *Conn) Close() error {
	if c.closed.CompareAndSwap(false, true) {
		return c.netConn.Close()
	}
	return nil
}

type ServerError struct {
	Status    byte
	MessageID int64
	Message   string
}

func (e *ServerError) Error() string {
	return fmt.Sprintf("server error (status=0x%02x, msgId=%d): %s", e.Status, e.MessageID, e.Message)
}
