package connection

import (
	"context"
	"fmt"

	"infinispan.org/go-client/internal/auth"
	"infinispan.org/go-client/internal/operation"
)

func (c *Conn) Authenticate(ctx context.Context, mech auth.Mechanism) error {
	mechListResult, err := c.Execute(ctx, &operation.AuthMechListOp{})
	if err != nil {
		return fmt.Errorf("auth mech list: %w", err)
	}
	mechs := mechListResult.([]string)
	found := false
	for _, m := range mechs {
		if m == mech.Name() {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("mechanism %q not supported by server (available: %v)", mech.Name(), mechs)
	}

	var responseData []byte
	if mech.HasInitialResponse() {
		responseData, err = mech.InitialResponse()
		if err != nil {
			return fmt.Errorf("initial response: %w", err)
		}
	}

	for {
		authResult, err := c.Execute(ctx, &operation.AuthOp{
			Mechanism:    mech.Name(),
			ResponseData: responseData,
		})
		if err != nil {
			return fmt.Errorf("auth: %w", err)
		}
		resp := authResult.(*operation.AuthResponse)
		if resp.Completed {
			if !mech.Complete() {
				_, err = mech.EvaluateChallenge(resp.Challenge)
				if err != nil {
					return fmt.Errorf("verify server: %w", err)
				}
			}
			return nil
		}
		responseData, err = mech.EvaluateChallenge(resp.Challenge)
		if err != nil {
			return fmt.Errorf("evaluate challenge: %w", err)
		}
		if mech.Complete() {
			return nil
		}
	}
}
