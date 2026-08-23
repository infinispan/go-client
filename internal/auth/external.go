package auth

type External struct {
	complete bool
}

func NewExternal() *External {
	return &External{}
}

func (e *External) Name() string            { return "EXTERNAL" }
func (e *External) HasInitialResponse() bool { return true }
func (e *External) Complete() bool         { return e.complete }

func (e *External) InitialResponse() ([]byte, error) {
	e.complete = true
	return []byte{}, nil
}

func (e *External) EvaluateChallenge([]byte) ([]byte, error) {
	return nil, nil
}
