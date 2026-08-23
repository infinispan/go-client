package hotrod

import (
	"context"
	"crypto/rand"
	"log/slog"

	"infinispan.org/go-client/internal/operation"
	"infinispan.org/go-client/internal/protostream"
)

const cqFactoryName = "continuous-query-filter-converter-factory"

// CQResultType identifies whether a continuous query event is a join, update, or leave.
type CQResultType = protostream.CQResultType

const (
	CQJoining = protostream.CQJoining
	CQUpdated = protostream.CQUpdated
	CQLeaving = protostream.CQLeaving
)

// CQEvent represents a single continuous query result event.
type CQEvent struct {
	Type        CQResultType
	Key         []byte
	Value       []byte
	Projections [][]byte
}

// ContinuousQuery represents an active continuous query whose Events channel receives matching entries.
type ContinuousQuery struct {
	Events <-chan *CQEvent
	id     []byte
	rawCh  chan []byte
	evCh   chan *CQEvent
	done   chan struct{}
	logger *slog.Logger
}

// CQOption configures a ContinuousQuery operation.
type CQOption func(*cqConfig)

type cqConfig struct {
	channelSize int
	params      []cqParam
}

type cqParam struct {
	name  string
	value any
}

// WithCQParam adds a named parameter binding to the continuous query.
func WithCQParam(name string, value any) CQOption {
	return func(c *cqConfig) {
		c.params = append(c.params, cqParam{name: name, value: value})
	}
}

// WithCQChannelSize sets the buffer size of the continuous query event channel.
func WithCQChannelSize(n int) CQOption {
	return func(c *cqConfig) {
		c.channelSize = n
	}
}

// ContinuousQuery registers a continuous Ickle query and returns a ContinuousQuery whose Events channel receives matching entries.
func (rc *RemoteCache) ContinuousQuery(ctx context.Context, query string, opts ...CQOption) (*ContinuousQuery, error) {
	cfg := &cqConfig{channelSize: 64}
	for _, o := range opts {
		o(cfg)
	}

	listenerID := make([]byte, 16)
	if _, err := rand.Read(listenerID); err != nil {
		return nil, err
	}

	factoryParams := buildCQParams(query, cfg.params)

	rawCh := make(chan []byte, cfg.channelSize)
	evCh := make(chan *CQEvent, cfg.channelSize)
	done := make(chan struct{})

	op := &operation.AddClientListenerOp{
		Cache:            rc.name,
		ListenerID:       listenerID,
		IncludeState:     true,
		Interests:        operation.InterestAll,
		FilterFactory:    cqFactoryName,
		ConverterFactory: cqFactoryName,
		FilterParams:     factoryParams,
		ConverterParams:  factoryParams,
	}

	if err := rc.client.pool.AddCustomListener(ctx, op, rawCh); err != nil {
		return nil, err
	}

	cq := &ContinuousQuery{
		Events: evCh,
		id:     listenerID,
		rawCh:  rawCh,
		evCh:   evCh,
		done:   done,
		logger: rc.client.logger,
	}

	go cq.decodeLoop()

	return cq, nil
}

// RemoveContinuousQuery unregisters an active continuous query.
func (rc *RemoteCache) RemoveContinuousQuery(ctx context.Context, cq *ContinuousQuery) error {
	close(cq.done)

	op := &operation.RemoveClientListenerOp{
		Cache:      rc.name,
		ListenerID: cq.id,
	}
	return rc.client.pool.RemoveListener(ctx, op)
}

func (cq *ContinuousQuery) decodeLoop() {
	defer close(cq.evCh)
	for {
		select {
		case data, ok := <-cq.rawCh:
			if !ok {
				return
			}
			inner, err := protostream.UnwrapBytes(data)
			if err != nil {
				cq.logger.Warn("unwrap CQ event", "err", err)
				continue
			}
			result, err := protostream.DecodeCQResult(inner)
			if err != nil {
				cq.logger.Warn("decode CQ result", "err", err)
				continue
			}
			ev := &CQEvent{
				Type:        result.ResultType,
				Key:         result.Key,
				Value:       result.Value,
				Projections: result.Projections,
			}
			select {
			case cq.evCh <- ev:
			case <-cq.done:
				return
			}
		case <-cq.done:
			return
		}
	}
}

func buildCQParams(query string, params []cqParam) [][]byte {
	result := [][]byte{protostream.WrapString(query)}
	for _, p := range params {
		result = append(result, protostream.WrapString(p.name))
		switch v := p.value.(type) {
		case string:
			result = append(result, protostream.WrapString(v))
		case int32:
			result = append(result, protostream.WrapInt32(v))
		case int:
			result = append(result, protostream.WrapInt32(int32(v)))
		case int64:
			result = append(result, protostream.WrapInt64(v))
		case float32:
			result = append(result, protostream.WrapFloat(v))
		case float64:
			result = append(result, protostream.WrapDouble(v))
		case bool:
			result = append(result, protostream.WrapBool(v))
		case []float32:
			result = append(result, protostream.WrapFloatArray(v))
		}
	}
	return result
}
