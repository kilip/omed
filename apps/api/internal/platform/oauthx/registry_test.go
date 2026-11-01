package oauthx_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/kilip/omed/api/internal/platform/oauthx"
)

type fakeDriver struct {
	id  string
	tag string
}

func (f fakeDriver) ID() string {
	return f.id
}

func (f fakeDriver) Scopes() []string {
	return nil
}

func (f fakeDriver) GetAuthURL(state, redirectURL string) string {
	return ""
}

func (f fakeDriver) ExchangeCode(ctx context.Context, code, redirectURL string) (*oauth2.Token, error) {
	return nil, nil
}

func (f fakeDriver) FetchUserProfile(ctx context.Context, token *oauth2.Token) (*oauthx.Profile, error) {
	return nil, nil
}

var _ oauthx.Driver = fakeDriver{}

func TestRegistry_GetReturnsRegisteredDriver(t *testing.T) {
	driverA := fakeDriver{id: "a"}
	reg := oauthx.NewRegistry(driverA)

	got, err := reg.Get("a")
	require.NoError(t, err)
	assert.Equal(t, driverA, got)
}

func TestRegistry_GetUnknownReturnsErrUnsupportedProvider(t *testing.T) {
	reg := oauthx.NewRegistry()

	_, errMissing := reg.Get("missing")
	require.Error(t, errMissing)
	assert.True(t, errors.Is(errMissing, oauthx.ErrUnsupportedProvider))

	_, errEmpty := reg.Get("")
	require.Error(t, errEmpty)
	assert.True(t, errors.Is(errEmpty, oauthx.ErrUnsupportedProvider))
}

func TestRegistry_RegisterReplacesSameID(t *testing.T) {
	first := fakeDriver{id: "a", tag: "first"}
	second := fakeDriver{id: "a", tag: "second"}
	reg := oauthx.NewRegistry(first)

	reg.Register(second)

	got, err := reg.Get("a")
	require.NoError(t, err)
	assert.Equal(t, second, got)
}

func TestRegistry_IDsSorted(t *testing.T) {
	emptyReg := oauthx.NewRegistry()
	assert.Empty(t, emptyReg.IDs())

	reg := oauthx.NewRegistry(fakeDriver{id: "google"}, fakeDriver{id: "github"})
	assert.Equal(t, []string{"github", "google"}, reg.IDs())
}

func TestRegistry_ConcurrentRegisterAndGet(t *testing.T) {
	reg := oauthx.NewRegistry()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(id string) {
			defer wg.Done()
			reg.Register(fakeDriver{id: id})
		}("id")

		go func() {
			defer wg.Done()
			_, _ = reg.Get("id")
		}()
	}

	wg.Wait()
}
