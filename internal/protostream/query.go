package protostream

import "errors"

// QueryParam holds a named query parameter with its pre-wrapped value.
type QueryParam struct {
	Name  string
	Value []byte // already-wrapped WrappedMessage bytes
}

// QueryResponse holds the decoded fields from a QueryResponse protobuf message.
type QueryResponse struct {
	NumResults     int32
	ProjectionSize int32
	Results        [][]byte // each is WrappedMessage bytes
	HitCount       int32
	HitCountExact  bool
}

// EncodeQueryRequest builds a raw QueryRequest protobuf (not wrapped in a WrappedMessage).
func EncodeQueryRequest(query string, startOffset int64, maxResults int32, params []QueryParam) []byte {
	// Field 1: queryString (string)
	dst := appendLenDelimited(nil, 1, []byte(query))

	// Field 3: startOffset (int64, omit if 0)
	if startOffset > 0 {
		dst = appendVarintField(dst, 3, uint64(startOffset))
	}

	// Field 4: maxResults (int32, omit if < 0)
	if maxResults >= 0 {
		dst = appendVarintField(dst, 4, uint64(maxResults))
	}

	// Field 5: namedParameters (repeated embedded message)
	for _, p := range params {
		paramMsg := appendLenDelimited(nil, 1, []byte(p.Name))
		paramMsg = appendLenDelimited(paramMsg, 2, p.Value)
		dst = appendLenDelimited(dst, 5, paramMsg)
	}

	return dst
}

// DecodeQueryResponse decodes a raw QueryResponse protobuf message.
func DecodeQueryResponse(data []byte) (*QueryResponse, error) {
	resp := &QueryResponse{}
	err := ScanFields(data, func(fieldNumber int, wireType int, value []byte) error {
		switch fieldNumber {
		case 1: // numResults
			v, n := decodeUvarint(value)
			if n <= 0 {
				return errors.New("QueryResponse: invalid numResults varint")
			}
			resp.NumResults = int32(v)
		case 2: // projectionSize
			v, n := decodeUvarint(value)
			if n <= 0 {
				return errors.New("QueryResponse: invalid projectionSize varint")
			}
			resp.ProjectionSize = int32(v)
		case 3: // results (repeated, length-delimited WrappedMessage)
			resp.Results = append(resp.Results, value)
		case 4: // hitCount
			v, n := decodeUvarint(value)
			if n <= 0 {
				return errors.New("QueryResponse: invalid hitCount varint")
			}
			resp.HitCount = int32(v)
		case 5: // hitCountExact
			v, n := decodeUvarint(value)
			if n <= 0 {
				return errors.New("QueryResponse: invalid hitCountExact varint")
			}
			resp.HitCountExact = v != 0
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}
