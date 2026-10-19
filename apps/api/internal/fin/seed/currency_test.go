package seed_test

import (
	"context"
	"testing"

	"github.com/kilip/omed/api/ent/currency"
	"github.com/kilip/omed/api/internal/fin/seed"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/require"
)

func TestSeedCurrencies(t *testing.T) {
	client := testutil.CreateTestDB("file:currency_seed_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	defer client.Close()

	ctx := context.Background()

	// Initial seed
	err := seed.SeedCurrencies(ctx, client)
	require.NoError(t, err)

	// Verify all default currencies exist
	count, err := client.Currency.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, len(seed.DefaultCurrencies), count)

	// Verify specific currencies and decimals
	idr, err := client.Currency.Query().Where(currency.CodeEQ("IDR")).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, idr.Decimals)
	require.Equal(t, "Rp", idr.Symbol)

	usd, err := client.Currency.Query().Where(currency.CodeEQ("USD")).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, usd.Decimals)
	require.Equal(t, "$", usd.Symbol)

	kwd, err := client.Currency.Query().Where(currency.CodeEQ("KWD")).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, kwd.Decimals)

	// Second seed should be idempotent
	err = seed.SeedCurrencies(ctx, client)
	require.NoError(t, err)

	countAfter, err := client.Currency.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, count, countAfter)
}
