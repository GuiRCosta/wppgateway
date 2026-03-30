package testutil

import "github.com/guilhermecosta/wpp-gateway/internal/domain"

func MockTenantInput(name string) domain.CreateTenantInput {
	return domain.CreateTenantInput{Name: name}
}

func MockGroupInput(name string, strategy domain.Strategy) domain.CreateGroupInput {
	return domain.CreateGroupInput{
		Name:     name,
		Strategy: strategy,
	}
}
