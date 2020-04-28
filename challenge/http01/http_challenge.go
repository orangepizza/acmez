package http01

import (
	"context"
	"fmt"

	"github.com/mholt/acme/acme"
	"github.com/mholt/acme/acme/api"
	"github.com/mholt/acme/challenge"
	"github.com/mholt/acme/log"
)

type ValidateFunc func(core *api.Core, domain string, chlng acme.Challenge) error

// ChallengePath returns the URL path for the `http-01` challenge
func ChallengePath(token string) string {
	return "/.well-known/acme-challenge/" + token
}

type Challenge struct {
	core     *api.Core
	validate ValidateFunc
	provider challenge.Provider
}

func NewChallenge(core *api.Core, validate ValidateFunc, provider challenge.Provider) *Challenge {
	return &Challenge{
		core:     core,
		validate: validate,
		provider: provider,
	}
}

func (c *Challenge) SetProvider(provider challenge.Provider) {
	c.provider = provider
}

func (c *Challenge) Solve(ctx context.Context, authz acme.Authorization) error {
	domain := challenge.GetTargetedDomain(authz)
	log.Infof("[%s] acme: Trying to solve HTTP-01", domain)

	chlng, err := challenge.FindChallenge(challenge.HTTP01, authz)
	if err != nil {
		return err
	}

	// Generate the Key Authorization for the challenge
	keyAuth, err := c.core.GetKeyAuthorization(chlng.Token)
	if err != nil {
		return err
	}

	err = c.provider.Present(ctx, challenge.Info{Domain: authz.Identifier.Value, Token: chlng.Token, KeyAuth: keyAuth})
	if err != nil {
		return fmt.Errorf("[%s] acme: error presenting token: %w", domain, err)
	}
	defer func() {
		err := c.provider.CleanUp(ctx, challenge.Info{Domain: authz.Identifier.Value, Token: chlng.Token, KeyAuth: keyAuth})
		if err != nil {
			log.Warnf("[%s] acme: cleaning up failed: %v", domain, err)
		}
	}()

	chlng.KeyAuthorization = keyAuth
	return c.validate(c.core, domain, chlng)
}
