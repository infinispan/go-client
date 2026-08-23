package auth

type Mechanism interface {
	Name() string
	HasInitialResponse() bool
	InitialResponse() ([]byte, error)
	EvaluateChallenge(challenge []byte) ([]byte, error)
	Complete() bool
}
