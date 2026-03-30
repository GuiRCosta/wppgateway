package testutil

import (
	"context"

	"github.com/google/uuid"

	"github.com/guilhermecosta/wpp-gateway/internal/domain"
)

// MockTenantRepo implements domain.TenantRepository for testing.
type MockTenantRepo struct {
	Tenants map[uuid.UUID]*domain.Tenant
	ByKey   map[string]*domain.Tenant
}

func NewMockTenantRepo() *MockTenantRepo {
	return &MockTenantRepo{
		Tenants: make(map[uuid.UUID]*domain.Tenant),
		ByKey:   make(map[string]*domain.Tenant),
	}
}

func (m *MockTenantRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.Tenant, error) {
	return m.Tenants[id], nil
}

func (m *MockTenantRepo) FindByAPIKey(_ context.Context, apiKey string) (*domain.Tenant, error) {
	return m.ByKey[apiKey], nil
}

func (m *MockTenantRepo) Create(_ context.Context, input domain.CreateTenantInput, apiKey string) (*domain.Tenant, error) {
	t := &domain.Tenant{
		ID:           uuid.New(),
		Name:         input.Name,
		APIKey:       apiKey,
		Plan:         "basic",
		MaxGroups:    5,
		MaxInstances: 20,
		IsActive:     true,
	}
	m.Tenants[t.ID] = t
	m.ByKey[apiKey] = t
	return t, nil
}

func (m *MockTenantRepo) Update(_ context.Context, id uuid.UUID, input domain.UpdateTenantInput) (*domain.Tenant, error) {
	t := m.Tenants[id]
	if t == nil {
		return nil, nil
	}
	if input.Name != nil {
		updated := *t
		updated.Name = *input.Name
		m.Tenants[id] = &updated
		return &updated, nil
	}
	return t, nil
}

func (m *MockTenantRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.Tenants, id)
	return nil
}

// MockGroupRepo implements domain.GroupRepository for testing.
type MockGroupRepo struct {
	Groups map[uuid.UUID]*domain.Group
}

func NewMockGroupRepo() *MockGroupRepo {
	return &MockGroupRepo{Groups: make(map[uuid.UUID]*domain.Group)}
}

func (m *MockGroupRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.Group, error) {
	return m.Groups[id], nil
}

func (m *MockGroupRepo) FindByTenantID(_ context.Context, tenantID uuid.UUID) ([]domain.Group, error) {
	var groups []domain.Group
	for _, g := range m.Groups {
		if g.TenantID == tenantID {
			groups = append(groups, *g)
		}
	}
	return groups, nil
}

func (m *MockGroupRepo) Create(_ context.Context, tenantID uuid.UUID, input domain.CreateGroupInput) (*domain.Group, error) {
	g := &domain.Group{
		ID:       uuid.New(),
		TenantID: tenantID,
		Name:     input.Name,
		Strategy: input.Strategy,
		Config:   []byte("{}"),
		IsActive: true,
	}
	m.Groups[g.ID] = g
	return g, nil
}

func (m *MockGroupRepo) Update(_ context.Context, id uuid.UUID, input domain.UpdateGroupInput) (*domain.Group, error) {
	g := m.Groups[id]
	if g == nil {
		return nil, nil
	}
	if input.Name != nil {
		updated := *g
		updated.Name = *input.Name
		m.Groups[id] = &updated
		return &updated, nil
	}
	return g, nil
}

func (m *MockGroupRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.Groups, id)
	return nil
}

func (m *MockGroupRepo) CountByTenantID(_ context.Context, tenantID uuid.UUID) (int64, error) {
	var count int64
	for _, g := range m.Groups {
		if g.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

// MockInstanceRepo implements domain.InstanceRepository for testing.
type MockInstanceRepo struct {
	Instances map[uuid.UUID]*domain.Instance
}

func NewMockInstanceRepo() *MockInstanceRepo {
	return &MockInstanceRepo{Instances: make(map[uuid.UUID]*domain.Instance)}
}

func (m *MockInstanceRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.Instance, error) {
	return m.Instances[id], nil
}

func (m *MockInstanceRepo) FindByGroupID(_ context.Context, groupID uuid.UUID) ([]domain.Instance, error) {
	var instances []domain.Instance
	for _, i := range m.Instances {
		if i.GroupID == groupID {
			instances = append(instances, *i)
		}
	}
	return instances, nil
}

func (m *MockInstanceRepo) FindAvailableByGroupID(_ context.Context, groupID uuid.UUID) ([]domain.Instance, error) {
	var instances []domain.Instance
	for _, i := range m.Instances {
		if i.GroupID == groupID && i.Status == domain.StatusAvailable {
			instances = append(instances, *i)
		}
	}
	return instances, nil
}

func (m *MockInstanceRepo) Create(_ context.Context, groupID uuid.UUID, input domain.CreateInstanceInput) (*domain.Instance, error) {
	i := &domain.Instance{
		ID:           uuid.New(),
		GroupID:      groupID,
		DisplayName:  input.DisplayName,
		Status:       domain.StatusDisconnected,
		Priority:     input.Priority,
		DailyBudget:  input.DailyBudget,
		HourlyBudget: input.HourlyBudget,
		DeliveryRate: 1.0,
	}
	m.Instances[i.ID] = i
	return i, nil
}

func (m *MockInstanceRepo) Update(_ context.Context, id uuid.UUID, _ domain.UpdateInstanceInput) (*domain.Instance, error) {
	return m.Instances[id], nil
}

func (m *MockInstanceRepo) UpdateStatus(_ context.Context, id uuid.UUID, status domain.InstanceStatus) error {
	if i := m.Instances[id]; i != nil {
		updated := *i
		updated.Status = status
		m.Instances[id] = &updated
	}
	return nil
}

func (m *MockInstanceRepo) UpdatePhone(_ context.Context, id uuid.UUID, phone string) error {
	if i := m.Instances[id]; i != nil {
		updated := *i
		updated.PhoneNumber = &phone
		m.Instances[id] = &updated
	}
	return nil
}

func (m *MockInstanceRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.Instances, id)
	return nil
}

func (m *MockInstanceRepo) CountByGroupID(_ context.Context, groupID uuid.UUID) (int64, error) {
	var count int64
	for _, i := range m.Instances {
		if i.GroupID == groupID {
			count++
		}
	}
	return count, nil
}

func (m *MockInstanceRepo) CountByTenantID(_ context.Context, _ uuid.UUID) (int64, error) {
	return int64(len(m.Instances)), nil
}
