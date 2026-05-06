package codec

const (
	RequestMagic  byte = 0xA0
	ResponseMagic byte = 0xA1
	Version41     byte = 41

	// Request opcodes.
	OpPut          byte = 0x01
	OpGet          byte = 0x03
	OpPutIfAbsent  byte = 0x05
	OpReplace      byte = 0x07
	OpReplaceIfUnmodified byte = 0x09
	OpRemove             byte = 0x0B
	OpRemoveIfUnmodified byte = 0x0D
	OpContainsKey        byte = 0x0F
	OpGetWithVersion     byte = 0x11
	OpClear        byte = 0x13
	OpStats        byte = 0x15
	OpPing         byte = 0x17
	OpGetWithMetadata byte = 0x1B
	OpQuery          byte = 0x1F
	OpAuthMechList   byte = 0x21
	OpAuth               byte = 0x23
	OpAddClientListener  byte = 0x25
	OpRemoveClientListener byte = 0x27
	OpExec               byte = 0x2B
	OpSize               byte = 0x29
	OpPutAll             byte = 0x2D
	OpGetAll             byte = 0x2F
	// Iteration request opcodes.
	OpIterationStart byte = 0x31
	OpIterationNext  byte = 0x33
	OpIterationEnd   byte = 0x35
	// Multimap request opcodes.
	OpMultimapGet              byte = 0x67
	OpMultimapGetWithMetadata  byte = 0x69
	OpMultimapPut              byte = 0x6B
	OpMultimapRemoveKey        byte = 0x6D
	OpMultimapRemoveEntry      byte = 0x6F
	OpMultimapSize             byte = 0x71
	OpMultimapContainsEntry    byte = 0x73
	OpMultimapContainsKey      byte = 0x75
	OpMultimapContainsValue    byte = 0x77

	// Transaction request opcodes.
	OpCommitTx    byte = 0x3D
	OpRollbackTx  byte = 0x3F
	OpPrepareTx2  byte = 0x7D

	OpAddBloomNearCacheListener byte = 0x41
	OpUpdateBloomFilter         byte = 0x43

	// Counter request opcodes.
	OpCounterCreate           byte = 0x4B
	OpCounterGetConfiguration byte = 0x4D
	OpCounterIsDefined        byte = 0x4F
	OpCounterAddAndGet        byte = 0x52
	OpCounterReset            byte = 0x54
	OpCounterGet              byte = 0x56
	OpCounterCAS              byte = 0x58
	OpCounterAddListener      byte = 0x5A
	OpCounterRemoveListener   byte = 0x5C
	OpCounterRemove           byte = 0x5E
	OpCounterGetNames         byte = 0x64
	OpCounterGetAndSet        byte = 0x7F

	// Response opcodes.
	OpPutResponse          byte = 0x02
	OpGetResponse          byte = 0x04
	OpPutIfAbsentResponse  byte = 0x06
	OpReplaceResponse      byte = 0x08
	OpReplaceIfUnmodifiedResponse byte = 0x0A
	OpRemoveResponse             byte = 0x0C
	OpRemoveIfUnmodifiedResponse byte = 0x0E
	OpContainsKeyResponse        byte = 0x10
	OpGetWithVersionResponse     byte = 0x12
	OpClearResponse        byte = 0x14
	OpStatsResponse        byte = 0x16
	OpPingResponse         byte = 0x18
	OpGetWithMetadataResponse byte = 0x1C
	OpQueryResponse        byte = 0x20
	OpAuthMechListResponse byte = 0x22
	OpAuthResponse               byte = 0x24
	OpAddClientListenerResponse  byte = 0x26
	OpRemoveClientListenerResponse byte = 0x28
	OpExecResponse               byte = 0x2C
	OpSizeResponse               byte = 0x2A
	OpPutAllResponse             byte = 0x2E
	OpGetAllResponse             byte = 0x30
	// Iteration response opcodes.
	OpIterationStartResponse byte = 0x32
	OpIterationNextResponse  byte = 0x34
	OpIterationEndResponse   byte = 0x36
	// Multimap response opcodes.
	OpMultimapGetResponse              byte = 0x68
	OpMultimapGetWithMetadataResponse  byte = 0x6A
	OpMultimapPutResponse              byte = 0x6C
	OpMultimapRemoveKeyResponse        byte = 0x6E
	OpMultimapRemoveEntryResponse      byte = 0x70
	OpMultimapSizeResponse             byte = 0x72
	OpMultimapContainsEntryResponse    byte = 0x74
	OpMultimapContainsKeyResponse      byte = 0x76
	OpMultimapContainsValueResponse    byte = 0x78

	// Transaction response opcodes.
	OpCommitTxResponse    byte = 0x3E
	OpRollbackTxResponse  byte = 0x40
	OpPrepareTx2Response  byte = 0x7E

	OpAddBloomNearCacheListenerResponse byte = 0x42
	OpUpdateBloomFilterResponse         byte = 0x44

	// Counter response opcodes.
	OpCounterCreateResponse           byte = 0x4C
	OpCounterGetConfigurationResponse byte = 0x4E
	OpCounterIsDefinedResponse        byte = 0x51
	OpCounterAddAndGetResponse        byte = 0x53
	OpCounterResetResponse            byte = 0x55
	OpCounterGetResponse              byte = 0x57
	OpCounterCASResponse              byte = 0x59
	OpCounterAddListenerResponse      byte = 0x5B
	OpCounterRemoveListenerResponse   byte = 0x5D
	OpCounterRemoveResponse           byte = 0x5F
	OpCounterGetNamesResponse         byte = 0x65
	OpCounterGetAndSetResponse        byte = 0x80

	OpErrorResponse              byte = 0x50

	// Cache entry event opcodes (server → client).
	OpCacheEntryCreated  byte = 0x60
	OpCacheEntryModified byte = 0x61
	OpCacheEntryRemoved  byte = 0x62
	OpCacheEntryExpired  byte = 0x63
	OpCounterEvent       byte = 0x66

	// Response status codes.
	StatusSuccess              byte = 0x00
	StatusNotExecuted          byte = 0x01
	StatusKeyDoesNotExist      byte = 0x02
	StatusSuccessWithPrevious  byte = 0x03
	StatusNotExecWithPrevious  byte = 0x04
	StatusCounterBoundReached   byte = 0x04
	StatusInvalidIteration     byte = 0x05
	StatusInvalidMagic         byte = 0x81
	StatusUnknownCommand       byte = 0x82
	StatusUnknownVersion       byte = 0x83
	StatusParseError           byte = 0x84
	StatusServerError          byte = 0x85
	StatusCommandTimeout       byte = 0x86
	StatusNodeSuspected        byte = 0x87
	StatusIllegalLifecycleState byte = 0x88

	// Client intelligence levels.
	IntelligenceBasic            byte = 0x01
	IntelligenceTopologyAware    byte = 0x02
	IntelligenceHashDistAware    byte = 0x03

	// MediaType kind indicators.
	MediaTypeNone       byte = 0x00
	MediaTypePredefined byte = 0x01
	MediaTypeCustom     byte = 0x02

	// Predefined media type IDs.
	MediaIDJavaObject   int32 = 1
	MediaIDJSON         int32 = 2
	MediaIDOctetStream  int32 = 3
	MediaIDProtostream  int32 = 12
	MediaIDTextPlain    int32 = 13

	// Time unit codes (used in the nibble encoding).
	TimeUnitSeconds      byte = 0x00
	TimeUnitMilliseconds byte = 0x01
	TimeUnitNanoseconds  byte = 0x02
	TimeUnitMicroseconds byte = 0x03
	TimeUnitMinutes      byte = 0x04
	TimeUnitHours        byte = 0x05
	TimeUnitDays         byte = 0x06
	TimeUnitDefault      byte = 0x07
	TimeUnitInfinite     byte = 0x08

	// XA return codes.
	XaOk     int32 = 0
	XaRdOnly int32 = 3

	DefaultPort = 11222
)

func IsError(status byte) bool {
	return status >= 0x81
}

func IsEvent(opCode byte) bool {
	return (opCode >= OpCacheEntryCreated && opCode <= OpCacheEntryExpired) || opCode == OpCounterEvent
}
