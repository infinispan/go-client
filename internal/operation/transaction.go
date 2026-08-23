package operation

import (
	"fmt"
	"io"
	"time"

	"infinispan.org/go-client/internal/codec"
)

const (
	CtrlNotRead     byte = 0x01
	CtrlNonExisting byte = 0x02
	CtrlRemoveOp    byte = 0x04
)

type TxModification struct {
	Key         []byte
	Value       []byte
	VersionRead int64
	Control     byte
	Lifespan    time.Duration
	MaxIdle     time.Duration
}

type PrepareTx2Op struct {
	Cache          string
	FormatID       int32
	GlobalTxID     []byte
	BranchQual     []byte
	OnePhaseCommit bool
	TimeoutMs      int64
	Modifications  []TxModification
}

func (o *PrepareTx2Op) RequestOpCode() byte   { return codec.OpPrepareTx2 }
func (o *PrepareTx2Op) ResponseOpCode() byte  { return codec.OpPrepareTx2Response }
func (o *PrepareTx2Op) CacheName() []byte     { return []byte(o.Cache) }
func (o *PrepareTx2Op) Flags() int32          { return 0 }
func (o *PrepareTx2Op) KeyMediaType() int32   { return codec.MediaIDOctetStream }
func (o *PrepareTx2Op) ValueMediaType() int32 { return codec.MediaIDOctetStream }

func (o *PrepareTx2Op) WriteBody(w io.Writer) error {
	if err := writeXID(w, o.FormatID, o.GlobalTxID, o.BranchQual); err != nil {
		return err
	}
	onePhase := byte(0)
	if o.OnePhaseCommit {
		onePhase = 1
	}
	if err := codec.WriteU1(w, onePhase); err != nil {
		return err
	}
	if err := codec.WriteU1(w, 0); err != nil {
		return err
	}
	if err := codec.WriteLong(w, o.TimeoutMs); err != nil {
		return err
	}
	if err := codec.WriteVInt(w, int32(len(o.Modifications))); err != nil {
		return err
	}
	for _, m := range o.Modifications {
		if err := writeModification(w, &m); err != nil {
			return err
		}
	}
	return nil
}

func (o *PrepareTx2Op) DecodeResponse(_ byte, r io.Reader) (any, error) {
	code, err := codec.ReadInt32(r)
	if err != nil {
		return nil, fmt.Errorf("read xa return code: %w", err)
	}
	return code, nil
}

type CommitTxOp struct {
	FormatID   int32
	GlobalTxID []byte
	BranchQual []byte
}

func (o *CommitTxOp) RequestOpCode() byte   { return codec.OpCommitTx }
func (o *CommitTxOp) ResponseOpCode() byte  { return codec.OpCommitTxResponse }
func (o *CommitTxOp) CacheName() []byte     { return nil }
func (o *CommitTxOp) Flags() int32          { return 0 }
func (o *CommitTxOp) KeyMediaType() int32   { return 0 }
func (o *CommitTxOp) ValueMediaType() int32 { return 0 }

func (o *CommitTxOp) WriteBody(w io.Writer) error {
	return writeXID(w, o.FormatID, o.GlobalTxID, o.BranchQual)
}

func (o *CommitTxOp) DecodeResponse(_ byte, r io.Reader) (any, error) {
	code, err := codec.ReadInt32(r)
	if err != nil {
		return nil, fmt.Errorf("read xa return code: %w", err)
	}
	return code, nil
}

type RollbackTxOp struct {
	FormatID   int32
	GlobalTxID []byte
	BranchQual []byte
}

func (o *RollbackTxOp) RequestOpCode() byte   { return codec.OpRollbackTx }
func (o *RollbackTxOp) ResponseOpCode() byte  { return codec.OpRollbackTxResponse }
func (o *RollbackTxOp) CacheName() []byte     { return nil }
func (o *RollbackTxOp) Flags() int32          { return 0 }
func (o *RollbackTxOp) KeyMediaType() int32   { return 0 }
func (o *RollbackTxOp) ValueMediaType() int32 { return 0 }

func (o *RollbackTxOp) WriteBody(w io.Writer) error {
	return writeXID(w, o.FormatID, o.GlobalTxID, o.BranchQual)
}

func (o *RollbackTxOp) DecodeResponse(_ byte, r io.Reader) (any, error) {
	code, err := codec.ReadInt32(r)
	if err != nil {
		return nil, fmt.Errorf("read xa return code: %w", err)
	}
	return code, nil
}

func writeXID(w io.Writer, formatID int32, globalTxID, branchQual []byte) error {
	if err := codec.WriteSignedVInt(w, formatID); err != nil {
		return fmt.Errorf("write xid formatId: %w", err)
	}
	if err := codec.WriteLPBytes(w, globalTxID); err != nil {
		return fmt.Errorf("write xid globalTxId: %w", err)
	}
	return codec.WriteLPBytes(w, branchQual)
}

func writeModification(w io.Writer, m *TxModification) error {
	if err := codec.WriteLPBytes(w, m.Key); err != nil {
		return err
	}
	if err := codec.WriteU1(w, m.Control); err != nil {
		return err
	}
	if m.Control&CtrlNonExisting == 0 && m.Control&CtrlNotRead == 0 {
		if err := codec.WriteLong(w, m.VersionRead); err != nil {
			return err
		}
	}
	if m.Control&CtrlRemoveOp == 0 {
		tu := codec.EncodeTimeUnits(m.Lifespan, m.MaxIdle)
		if err := tu.Write(w); err != nil {
			return err
		}
		if err := codec.WriteLPBytes(w, m.Value); err != nil {
			return err
		}
	}
	return nil
}
