package service

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/core/repository"
	"github.com/stretchr/testify/suite"
	_ "modernc.org/sqlite"
)

type AccessTokenServiceSuite struct {
	suite.Suite
	client    *ent.Client
	tokenRepo repository.AccessTokenRepository
	svc       AccessTokenService
	ctx       context.Context
}

func createTestDB(dsn string) *ent.Client {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		panic(err)
	}
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := ent.NewClient(ent.Driver(drv))
	if err := client.Schema.Create(context.Background()); err != nil {
		panic(err)
	}
	return client
}

func (s *AccessTokenServiceSuite) SetupSuite() {
	s.client = createTestDB("file:access_token_svc_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	s.tokenRepo = repository.NewAccessTokenRepository(s.client)
	s.svc = NewAccessTokenService(s.tokenRepo)
	s.ctx = context.Background()
}

func (s *AccessTokenServiceSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *AccessTokenServiceSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(context.Background())
	_, _ = s.client.PersonalAccessToken.Delete().Exec(ctx)
	_, _ = s.client.User.Delete().Exec(ctx)
	if impl, ok := s.svc.(*accessTokenService); ok {
		impl.now = time.Now
	}
}

func (s *AccessTokenServiceSuite) createTestUser(name, email string) *ent.User {
	ctx := context.Background()
	u, err := s.client.User.Create().
		SetName(name).
		SetEmail(email).
		Save(ctx)
	s.Require().NoError(err)
	return u
}

func (s *AccessTokenServiceSuite) Test_Create_ReturnsPlaintextOnceAndStoresOnlyHash() {
	user := s.createTestUser("Alice", "alice@example.com")

	tok, plaintext, err := s.svc.Create(s.ctx, user.ID, model.CreateAccessTokenRequest{
		Name: "my-cli-token",
	})
	s.Require().NoError(err)
	s.NotZero(tok.ID)
	s.Equal("my-cli-token", tok.Name)
	s.True(strings.HasPrefix(plaintext, AccessTokenPrefix))

	// Find in repository by sha256 hash of plaintext
	expectedHash := hashToken(plaintext)
	stored, err := s.tokenRepo.FindByHash(s.ctx, expectedHash)
	s.Require().NoError(err)
	s.Equal(expectedHash, stored.TokenHash)

	// No stored field in DB should equal the plaintext
	s.NotEqual(plaintext, stored.Name)
	s.NotEqual(plaintext, stored.TokenHash)
	s.NotEqual(plaintext, stored.Prefix)

	// Prefix is a strict prefix of the plaintext
	s.Equal(12, len(stored.Prefix))
	s.True(strings.HasPrefix(plaintext, stored.Prefix))
	s.Less(len(stored.Prefix), len(plaintext))
}

func (s *AccessTokenServiceSuite) Test_Create_ExpiryDays() {
	user := s.createTestUser("Bob", "bob@example.com")
	fixedNow := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	s.svc.(*accessTokenService).now = func() time.Time { return fixedNow }

	// 0 gives default 90 days
	tok0, _, err := s.svc.Create(s.ctx, user.ID, model.CreateAccessTokenRequest{
		Name:          "token-default",
		ExpiresInDays: 0,
	})
	s.Require().NoError(err)
	s.True(tok0.ExpiresAt.Equal(fixedNow.AddDate(0, 0, 90)))

	// 30 days
	tok30, _, err := s.svc.Create(s.ctx, user.ID, model.CreateAccessTokenRequest{
		Name:          "token-30",
		ExpiresInDays: 30,
	})
	s.Require().NoError(err)
	s.True(tok30.ExpiresAt.Equal(fixedNow.AddDate(0, 0, 30)))

	// 90 days
	tok90, _, err := s.svc.Create(s.ctx, user.ID, model.CreateAccessTokenRequest{
		Name:          "token-90",
		ExpiresInDays: 90,
	})
	s.Require().NoError(err)
	s.True(tok90.ExpiresAt.Equal(fixedNow.AddDate(0, 0, 90)))

	// 365 days
	tok365, _, err := s.svc.Create(s.ctx, user.ID, model.CreateAccessTokenRequest{
		Name:          "token-365",
		ExpiresInDays: 365,
	})
	s.Require().NoError(err)
	s.True(tok365.ExpiresAt.Equal(fixedNow.AddDate(0, 0, 365)))

	// 45 days returns ErrTokenInvalidExpiry
	_, _, err = s.svc.Create(s.ctx, user.ID, model.CreateAccessTokenRequest{
		Name:          "token-45",
		ExpiresInDays: 45,
	})
	s.Require().ErrorIs(err, domain.ErrTokenInvalidExpiry)
}

func (s *AccessTokenServiceSuite) Test_Create_Name() {
	user := s.createTestUser("Charlie", "charlie@example.com")

	// Whitespace-only rejected
	_, _, err := s.svc.Create(s.ctx, user.ID, model.CreateAccessTokenRequest{
		Name: "   ",
	})
	s.Require().ErrorIs(err, domain.ErrTokenInvalidName)

	// Empty string rejected
	_, _, err = s.svc.Create(s.ctx, user.ID, model.CreateAccessTokenRequest{
		Name: "",
	})
	s.Require().ErrorIs(err, domain.ErrTokenInvalidName)

	// Name is trimmed
	tokTrimmed, _, err := s.svc.Create(s.ctx, user.ID, model.CreateAccessTokenRequest{
		Name: "  Token Berkah  ",
	})
	s.Require().NoError(err)
	s.Equal("Token Berkah", tokTrimmed.Name)

	// 100 runes mixing emoji and Indonesian text accepted (RF2)
	prefixText := "Kunci Akses Dompet Keuangan 🚀 "
	prefixRunes := utf8.RuneCountInString(prefixText)
	name100 := prefixText + strings.Repeat("a", 100-prefixRunes)
	s.Require().Equal(100, utf8.RuneCountInString(name100))

	tok100, _, err := s.svc.Create(s.ctx, user.ID, model.CreateAccessTokenRequest{
		Name: name100,
	})
	s.Require().NoError(err)
	s.Equal(name100, tok100.Name)

	// 101 runes rejected
	name101 := name100 + "b"
	s.Require().Equal(101, utf8.RuneCountInString(name101))

	_, _, err = s.svc.Create(s.ctx, user.ID, model.CreateAccessTokenRequest{
		Name: name101,
	})
	s.Require().ErrorIs(err, domain.ErrTokenInvalidName)
}

func (s *AccessTokenServiceSuite) Test_Create_LimitReached() {
	user := s.createTestUser("Dave", "dave@example.com")
	fixedNow := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	s.svc.(*accessTokenService).now = func() time.Time { return fixedNow }

	var firstTokID int64
	// Create 20 active tokens
	for i := 1; i <= MaxActiveAccessTokens; i++ {
		tok, _, err := s.svc.Create(s.ctx, user.ID, model.CreateAccessTokenRequest{
			Name: "token-active",
		})
		s.Require().NoError(err)
		if i == 1 {
			firstTokID = tok.ID
		}
	}

	// 21st active token returns ErrTokenLimitReached
	_, _, err := s.svc.Create(s.ctx, user.ID, model.CreateAccessTokenRequest{
		Name: "token-21",
	})
	s.Require().ErrorIs(err, domain.ErrTokenLimitReached)

	// Revoke one token
	err = s.svc.Revoke(s.ctx, user.ID, firstTokID)
	s.Require().NoError(err)

	// Now create succeeds
	tokAfterRevoke, _, err := s.svc.Create(s.ctx, user.ID, model.CreateAccessTokenRequest{
		Name: "token-after-revoke",
	})
	s.Require().NoError(err)
	s.NotZero(tokAfterRevoke.ID)

	// Expired tokens do not count against limit
	// Advance time past expiry for all tokens
	s.svc.(*accessTokenService).now = func() time.Time { return fixedNow.AddDate(0, 0, 100) }

	// User should be able to create again without hitting limit
	tokAfterExpiry, _, err := s.svc.Create(s.ctx, user.ID, model.CreateAccessTokenRequest{
		Name: "token-after-expiry",
	})
	s.Require().NoError(err)
	s.NotZero(tokAfterExpiry.ID)
}

func (s *AccessTokenServiceSuite) Test_Verify() {
	user := s.createTestUser("Eve", "eve@example.com")
	fixedNow := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	s.svc.(*accessTokenService).now = func() time.Time { return fixedNow }

	// 1. Active returns the token
	created, plaintext, err := s.svc.Create(s.ctx, user.ID, model.CreateAccessTokenRequest{
		Name:          "active-token",
		ExpiresInDays: 30,
	})
	s.Require().NoError(err)

	verified, err := s.svc.Verify(s.ctx, plaintext)
	s.Require().NoError(err)
	s.Equal(created.ID, verified.ID)
	s.Equal(user.ID, verified.UserID)

	// 2. Expired returns ErrTokenExpired
	s.svc.(*accessTokenService).now = func() time.Time { return fixedNow.AddDate(0, 0, 31) }
	_, err = s.svc.Verify(s.ctx, plaintext)
	s.Require().ErrorIs(err, domain.ErrTokenExpired)

	// 3. Revoked returns ErrTokenInvalid
	s.svc.(*accessTokenService).now = func() time.Time { return fixedNow }
	createdRev, plaintextRev, err := s.svc.Create(s.ctx, user.ID, model.CreateAccessTokenRequest{
		Name:          "to-revoke",
		ExpiresInDays: 30,
	})
	s.Require().NoError(err)

	err = s.svc.Revoke(s.ctx, user.ID, createdRev.ID)
	s.Require().NoError(err)

	_, err = s.svc.Verify(s.ctx, plaintextRev)
	s.Require().ErrorIs(err, domain.ErrTokenInvalid)

	// 4. Revoked also returns ErrTokenInvalid when expired
	s.svc.(*accessTokenService).now = func() time.Time { return fixedNow.AddDate(0, 0, 35) }
	_, err = s.svc.Verify(s.ctx, plaintextRev)
	s.Require().ErrorIs(err, domain.ErrTokenInvalid)

	// 5. Unknown, empty, no-prefix, and omed_pat_ plus garbage return ErrTokenInvalid
	s.svc.(*accessTokenService).now = func() time.Time { return fixedNow }
	_, err = s.svc.Verify(s.ctx, "")
	s.Require().ErrorIs(err, domain.ErrTokenInvalid)

	_, err = s.svc.Verify(s.ctx, "not_omed_prefix_random_token")
	s.Require().ErrorIs(err, domain.ErrTokenInvalid)

	_, err = s.svc.Verify(s.ctx, "omed_pat_garbage1234567890")
	s.Require().ErrorIs(err, domain.ErrTokenInvalid)

	_, err = s.svc.Verify(s.ctx, "omed_pat_")
	s.Require().ErrorIs(err, domain.ErrTokenInvalid)
}

func (s *AccessTokenServiceSuite) Test_Revoke_OtherUsers_ReturnsNotFound() {
	frank := s.createTestUser("Frank", "frank@example.com")
	grace := s.createTestUser("Grace", "grace@example.com")

	tok, _, err := s.svc.Create(s.ctx, frank.ID, model.CreateAccessTokenRequest{
		Name: "frank-token",
	})
	s.Require().NoError(err)

	// Grace tries to revoke Frank's token
	err = s.svc.Revoke(s.ctx, grace.ID, tok.ID)
	s.Require().ErrorIs(err, base.ErrNotFound)

	// Non-existent token ID returns ErrNotFound
	err = s.svc.Revoke(s.ctx, frank.ID, 999999)
	s.Require().ErrorIs(err, base.ErrNotFound)
}

func (s *AccessTokenServiceSuite) Test_List_ReturnsOnlyOwnWithoutHash() {
	alice := s.createTestUser("AliceList", "alicelist@example.com")
	bob := s.createTestUser("BobList", "boblist@example.com")
	charlie := s.createTestUser("CharlieList", "charlielist@example.com")

	_, _, err := s.svc.Create(s.ctx, alice.ID, model.CreateAccessTokenRequest{Name: "alice-1"})
	s.Require().NoError(err)
	_, _, err = s.svc.Create(s.ctx, alice.ID, model.CreateAccessTokenRequest{Name: "alice-2"})
	s.Require().NoError(err)

	_, _, err = s.svc.Create(s.ctx, bob.ID, model.CreateAccessTokenRequest{Name: "bob-1"})
	s.Require().NoError(err)

	aliceTokens, err := s.svc.List(s.ctx, alice.ID)
	s.Require().NoError(err)
	s.Require().Len(aliceTokens, 2)
	for _, t := range aliceTokens {
		s.Equal(alice.ID, t.UserID)
		s.Empty(t.TokenHash, "TokenHash must be cleared in List")
	}

	bobTokens, err := s.svc.List(s.ctx, bob.ID)
	s.Require().NoError(err)
	s.Require().Len(bobTokens, 1)
	s.Equal(bob.ID, bobTokens[0].UserID)
	s.Empty(bobTokens[0].TokenHash, "TokenHash must be cleared in List")

	charlieTokens, err := s.svc.List(s.ctx, charlie.ID)
	s.Require().NoError(err)
	s.NotNil(charlieTokens)
	s.Empty(charlieTokens)
}

func TestAccessTokenServiceSuite(t *testing.T) {
	suite.Run(t, new(AccessTokenServiceSuite))
}
