//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func contributorProxy(id int64, name, host, countryCode, ipAddress string) ProxyWithAccountCount {
	return ProxyWithAccountCount{
		Proxy: Proxy{
			ID:        id,
			Name:      name,
			Protocol:  "http",
			Host:      host,
			Port:      8080,
			Status:    StatusActive,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
		CountryCode: countryCode,
		IPAddress:   ipAddress,
	}
}

func TestSelectContributorProxyFromList(t *testing.T) {
	proxies := []ProxyWithAccountCount{
		contributorProxy(1, "default", "us-default.example.com", "US", "1.1.1.1"),
		contributorProxy(2, "us-alt", "us-alt.example.com", "US", "1.1.1.2"),
		contributorProxy(3, "default", "jp-default.example.com", "JP", "2.2.2.2"),
		contributorProxy(4, "sg-alt", "sg-alt.example.com", "SG", "3.3.3.3"),
	}

	require.Equal(t, int64(1), selectContributorProxyFromList(proxies, "us").ID)
	require.Equal(t, int64(4), selectContributorProxyFromList(proxies[3:], "SG").ID)

	fallbackDefault := selectContributorProxyFromList(proxies[:2], "CA")
	require.Equal(t, int64(1), fallbackDefault.ID)

	noDefault := selectContributorProxyFromList([]ProxyWithAccountCount{proxies[3]}, "")
	require.Equal(t, int64(4), noDefault.ID)

	require.Nil(t, selectContributorProxyFromList(nil, "US"))
}

type contributorProxyRepoStub struct {
	mockGeminiProxyRepo
	proxies      []ProxyWithAccountCount
	updated      []*Proxy
	reservations []ContributorProxyReservation
	nextResID    int64
}

func (r *contributorProxyRepoStub) ListActiveWithAccountCount(ctx context.Context) ([]ProxyWithAccountCount, error) {
	return r.proxies, nil
}

func (r *contributorProxyRepoStub) Update(ctx context.Context, proxy *Proxy) error {
	copied := *proxy
	r.updated = append(r.updated, &copied)
	for i := range r.proxies {
		if r.proxies[i].ID == proxy.ID {
			r.proxies[i].Proxy = *proxy
		}
	}
	return nil
}

func (r *contributorProxyRepoStub) GetByID(ctx context.Context, id int64) (*Proxy, error) {
	for i := range r.proxies {
		if r.proxies[i].ID == id {
			return &r.proxies[i].Proxy, nil
		}
	}
	return nil, ErrProxyNotFound
}

func (r *contributorProxyRepoStub) ListActive(ctx context.Context) ([]Proxy, error) {
	out := make([]Proxy, 0, len(r.proxies))
	for i := range r.proxies {
		out = append(out, r.proxies[i].Proxy)
	}
	return out, nil
}

func (r *contributorProxyRepoStub) List(ctx context.Context, params pagination.PaginationParams) ([]Proxy, *pagination.PaginationResult, error) {
	proxies, err := r.ListActive(ctx)
	if err != nil {
		return nil, nil, err
	}
	return proxies, &pagination.PaginationResult{Total: int64(len(proxies))}, nil
}

func (r *contributorProxyRepoStub) ExpireContributorProxyReservations(ctx context.Context, now time.Time) error {
	for i := range r.reservations {
		if r.reservations[i].Status == "active" && !r.reservations[i].ExpiresAt.After(now) {
			r.reservations[i].Status = "expired"
			r.reservations[i].UpdatedAt = now
		}
	}
	return nil
}

func (r *contributorProxyRepoStub) GetActiveContributorProxyReservation(ctx context.Context, ownerUserID int64, now, expiresAt time.Time) (*ContributorProxyReservation, error) {
	var selected *ContributorProxyReservation
	for i := range r.reservations {
		reservation := &r.reservations[i]
		if reservation.OwnerUserID != ownerUserID || reservation.Status != "active" || !reservation.ExpiresAt.After(now) {
			continue
		}
		if selected == nil || reservation.UpdatedAt.After(selected.UpdatedAt) || (reservation.UpdatedAt.Equal(selected.UpdatedAt) && reservation.ID > selected.ID) {
			selected = reservation
		}
	}
	if selected == nil {
		return nil, nil
	}
	selected.ExpiresAt = expiresAt
	selected.UpdatedAt = now
	copied := *selected
	return &copied, nil
}

func (r *contributorProxyRepoStub) ListActiveContributorProxyReservationProxyIDs(ctx context.Context, now time.Time) (map[int64]struct{}, error) {
	out := make(map[int64]struct{})
	for i := range r.reservations {
		reservation := r.reservations[i]
		if reservation.Status == "active" && reservation.ExpiresAt.After(now) {
			out[reservation.ProxyID] = struct{}{}
		}
	}
	return out, nil
}

func (r *contributorProxyRepoStub) CreateContributorProxyReservation(ctx context.Context, ownerUserID, proxyID int64, country string, expiresAt time.Time) (*ContributorProxyReservation, error) {
	now := time.Now().UTC()
	for i := range r.reservations {
		reservation := r.reservations[i]
		if reservation.OwnerUserID == ownerUserID && reservation.Status == "active" && reservation.ExpiresAt.After(now) {
			return nil, nil
		}
		if reservation.ProxyID == proxyID && reservation.Status == "active" && reservation.ExpiresAt.After(now) {
			return nil, nil
		}
	}
	r.nextResID++
	reservation := ContributorProxyReservation{
		ID:          r.nextResID,
		ProxyID:     proxyID,
		OwnerUserID: ownerUserID,
		Country:     country,
		Status:      "active",
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	r.reservations = append(r.reservations, reservation)
	copied := reservation
	return &copied, nil
}

func (r *contributorProxyRepoStub) ConsumeContributorProxyReservation(ctx context.Context, ownerUserID, proxyID int64) error {
	now := time.Now().UTC()
	for i := range r.reservations {
		if r.reservations[i].OwnerUserID == ownerUserID && r.reservations[i].ProxyID == proxyID && r.reservations[i].Status == "active" && r.reservations[i].ExpiresAt.After(now) {
			r.reservations[i].Status = "consumed"
			r.reservations[i].UpdatedAt = now
		}
	}
	return nil
}

func (r *contributorProxyRepoStub) ReleaseContributorProxyReservations(ctx context.Context, ownerUserID int64) error {
	now := time.Now().UTC()
	for i := range r.reservations {
		if r.reservations[i].OwnerUserID == ownerUserID && r.reservations[i].Status == "active" {
			r.reservations[i].Status = "released"
			r.reservations[i].UpdatedAt = now
		}
	}
	return nil
}

type contributorAccountRepoStub struct {
	mockAccountRepoForGemini
	account *Account
}

func (r *contributorAccountRepoStub) Create(ctx context.Context, account *Account) error {
	copied := *account
	copied.ID = 100
	r.account = &copied
	account.ID = copied.ID
	return nil
}

func (r *contributorAccountRepoStub) Update(ctx context.Context, account *Account) error {
	copied := *account
	r.account = &copied
	return nil
}

func (r *contributorAccountRepoStub) GetByID(ctx context.Context, id int64) (*Account, error) {
	if r.account == nil || r.account.ID != id {
		return nil, ErrAccountNotFound
	}
	copied := *r.account
	return &copied, nil
}

func TestSelectContributorProxyReusesExistingReservation(t *testing.T) {
	proxyRepo := &contributorProxyRepoStub{
		proxies: []ProxyWithAccountCount{
			contributorProxy(1, "default", "us-default.example.com", "US", "1.1.1.1"),
			contributorProxy(2, "default", "jp-default.example.com", "JP", "2.2.2.2"),
		},
	}
	svc := &adminServiceImpl{proxyRepo: proxyRepo}

	first, err := svc.SelectContributorProxy(context.Background(), 42, "US")
	require.NoError(t, err)
	require.NotNil(t, first)

	initialExpiry := proxyRepo.reservations[0].ExpiresAt
	second, err := svc.SelectContributorProxy(context.Background(), 42, "JP")
	require.NoError(t, err)
	require.NotNil(t, second)
	require.Equal(t, first.ID, second.ID)
	require.Len(t, proxyRepo.reservations, 1)
	require.True(t, proxyRepo.reservations[0].ExpiresAt.After(initialExpiry) || proxyRepo.reservations[0].ExpiresAt.Equal(initialExpiry))
}

func TestSelectContributorProxySkipsReservedProxyForOtherUsers(t *testing.T) {
	proxyRepo := &contributorProxyRepoStub{
		proxies: []ProxyWithAccountCount{
			contributorProxy(1, "default", "us-default.example.com", "US", "1.1.1.1"),
			contributorProxy(2, "us-alt", "us-alt.example.com", "US", "1.1.1.2"),
		},
		reservations: []ContributorProxyReservation{
			{
				ID:          1,
				ProxyID:     1,
				OwnerUserID: 7,
				Country:     "US",
				Status:      "active",
				ExpiresAt:   time.Now().UTC().Add(time.Hour),
				CreatedAt:   time.Now().UTC(),
				UpdatedAt:   time.Now().UTC(),
			},
		},
		nextResID: 1,
	}
	svc := &adminServiceImpl{proxyRepo: proxyRepo}

	selected, err := svc.SelectContributorProxy(context.Background(), 42, "US")
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, int64(2), selected.ID)
	require.Len(t, proxyRepo.reservations, 2)
	require.Equal(t, int64(2), proxyRepo.reservations[1].ProxyID)
}

func TestSelectContributorProxyAllowsExpiredReservationReuse(t *testing.T) {
	now := time.Now().UTC()
	proxyRepo := &contributorProxyRepoStub{
		proxies: []ProxyWithAccountCount{
			contributorProxy(1, "default", "us-default.example.com", "US", "1.1.1.1"),
		},
		reservations: []ContributorProxyReservation{
			{
				ID:          1,
				ProxyID:     1,
				OwnerUserID: 7,
				Country:     "US",
				Status:      "active",
				ExpiresAt:   now.Add(-time.Minute),
				CreatedAt:   now.Add(-time.Hour),
				UpdatedAt:   now.Add(-time.Hour),
			},
		},
		nextResID: 1,
	}
	svc := &adminServiceImpl{proxyRepo: proxyRepo}

	selected, err := svc.SelectContributorProxy(context.Background(), 42, "US")
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, int64(1), selected.ID)
	require.Equal(t, "expired", proxyRepo.reservations[0].Status)
	require.Len(t, proxyRepo.reservations, 2)
	require.Equal(t, "active", proxyRepo.reservations[1].Status)
}

func TestSelectContributorProxyReturnsNilWhenAllProxiesReserved(t *testing.T) {
	now := time.Now().UTC()
	proxyRepo := &contributorProxyRepoStub{
		proxies: []ProxyWithAccountCount{
			contributorProxy(1, "default", "us-default.example.com", "US", "1.1.1.1"),
		},
		reservations: []ContributorProxyReservation{
			{
				ID:          1,
				ProxyID:     1,
				OwnerUserID: 7,
				Country:     "US",
				Status:      "active",
				ExpiresAt:   now.Add(time.Hour),
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
		nextResID: 1,
	}
	svc := &adminServiceImpl{proxyRepo: proxyRepo}

	selected, err := svc.SelectContributorProxy(context.Background(), 42, "US")
	require.NoError(t, err)
	require.Nil(t, selected)
}

func TestCreateContributorAccountRenamesConsumedDefaultProxy(t *testing.T) {
	proxyID := int64(7)
	now := time.Now().UTC()
	proxyRepo := &contributorProxyRepoStub{
		proxies: []ProxyWithAccountCount{
			contributorProxy(proxyID, "default", "proxy-host.example.com", "JP", "203.0.113.7"),
		},
		reservations: []ContributorProxyReservation{
			{
				ID:          1,
				ProxyID:     proxyID,
				OwnerUserID: 42,
				Country:     "US",
				Status:      "active",
				ExpiresAt:   now.Add(time.Hour),
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
		nextResID: 1,
	}
	accountRepo := &contributorAccountRepoStub{}
	svc := &adminServiceImpl{accountRepo: accountRepo, proxyRepo: proxyRepo}

	account, err := svc.CreateContributorAccount(context.Background(), 42, &CreateAccountInput{
		Name:               "Claude OAuth",
		Platform:           PlatformAnthropic,
		Type:               AccountTypeSetupToken,
		Credentials:        map[string]any{"access_token": "token"},
		ProxyID:            &proxyID,
		ContributorCountry: "us",
	})

	require.NoError(t, err)
	require.NotNil(t, account)
	require.Len(t, proxyRepo.updated, 1)
	require.Equal(t, "US203.0.113.7", proxyRepo.updated[0].Name)
	require.Equal(t, "consumed", proxyRepo.reservations[0].Status)
}

func TestCreateContributorAccountDoesNotRenameNonDefaultProxy(t *testing.T) {
	proxyID := int64(8)
	proxyRepo := &contributorProxyRepoStub{
		proxies: []ProxyWithAccountCount{
			contributorProxy(proxyID, "named-proxy", "proxy-host.example.com", "JP", "203.0.113.8"),
		},
	}
	accountRepo := &contributorAccountRepoStub{}
	svc := &adminServiceImpl{accountRepo: accountRepo, proxyRepo: proxyRepo}

	_, err := svc.CreateContributorAccount(context.Background(), 42, &CreateAccountInput{
		Name:        "Claude OAuth",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeSetupToken,
		Credentials: map[string]any{"access_token": "token"},
		ProxyID:     &proxyID,
	})

	require.NoError(t, err)
	require.Empty(t, proxyRepo.updated)
}
