package service

import "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/repository"

type Services struct {
	User        UserService
	Account     AccountService
	Category    CategoryService
	Provider    ProviderService
	Transaction TransactionService
	Analytics   AnalyticsService
}

func NewServices(repos *repository.Repositories) *Services {
	return &Services{
		User:        &userServiceImpl{repo: repos.User},
		Account:     &accountServiceImpl{repo: repos.Account},
		Category:    &categoryServiceImpl{repo: repos.Category},
		Provider:    &providerServiceImpl{repo: repos.Provider},
		Transaction: &transactionServiceImpl{txRepo: repos.Transaction, accRepo: repos.Account, catRepo: repos.Category},
		Analytics:   &analyticsServiceImpl{repo: repos.Analytics},
	}
}
