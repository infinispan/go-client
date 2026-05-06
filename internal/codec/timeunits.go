package codec

import (
	"io"
	"time"
)

type TimeUnits struct {
	Packed   byte
	Lifespan int64
	MaxIdle  int64
}

func EncodeTimeUnits(lifespan, maxIdle time.Duration) TimeUnits {
	lifespanCode, lifespanVal := encodeDuration(lifespan)
	maxIdleCode, maxIdleVal := encodeDuration(maxIdle)
	return TimeUnits{
		Packed:   (lifespanCode << 4) | maxIdleCode,
		Lifespan: lifespanVal,
		MaxIdle:  maxIdleVal,
	}
}

func (t TimeUnits) Write(w io.Writer) error {
	if err := WriteU1(w, t.Packed); err != nil {
		return err
	}
	if (t.Packed>>4)&0x0F < TimeUnitDefault {
		if err := WriteVLong(w, t.Lifespan); err != nil {
			return err
		}
	}
	if t.Packed&0x0F < TimeUnitDefault {
		if err := WriteVLong(w, t.MaxIdle); err != nil {
			return err
		}
	}
	return nil
}

func encodeDuration(d time.Duration) (code byte, value int64) {
	if d == 0 {
		return TimeUnitDefault, 0
	}
	if d < 0 {
		return TimeUnitInfinite, 0
	}
	switch {
	case d%time.Hour == 0 && d/time.Hour < (24*365):
		return TimeUnitHours, int64(d / time.Hour)
	case d%time.Minute == 0:
		return TimeUnitMinutes, int64(d / time.Minute)
	case d%time.Second == 0:
		return TimeUnitSeconds, int64(d / time.Second)
	case d%time.Millisecond == 0:
		return TimeUnitMilliseconds, int64(d / time.Millisecond)
	case d%time.Microsecond == 0:
		return TimeUnitMicroseconds, int64(d / time.Microsecond)
	default:
		return TimeUnitNanoseconds, int64(d)
	}
}
