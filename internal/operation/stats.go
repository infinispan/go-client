package operation

import (
	"io"

	"infinispan.org/go-client/internal/codec"
)

type StatsOp struct {
	Cache string
}

func (o *StatsOp) RequestOpCode() byte   { return codec.OpStats }
func (o *StatsOp) ResponseOpCode() byte  { return codec.OpStatsResponse }
func (o *StatsOp) CacheName() []byte     { return []byte(o.Cache) }
func (o *StatsOp) Flags() int32          { return 0 }
func (o *StatsOp) KeyMediaType() int32   { return 0 }
func (o *StatsOp) ValueMediaType() int32 { return 0 }

func (o *StatsOp) WriteBody(_ io.Writer) error {
	return nil
}

func (o *StatsOp) DecodeResponse(_ byte, r io.Reader) (any, error) {
	count, err := codec.ReadVInt(r)
	if err != nil {
		return nil, err
	}
	stats := make(map[string]string, count)
	for i := int32(0); i < count; i++ {
		name, err := codec.ReadLPString(r)
		if err != nil {
			return nil, err
		}
		value, err := codec.ReadLPString(r)
		if err != nil {
			return nil, err
		}
		stats[name] = value
	}
	return stats, nil
}
