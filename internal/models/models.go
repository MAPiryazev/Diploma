package models

import (
	"time"
)

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type Account struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Number    string    `json:"number"`
	CreatedAt time.Time `json:"created_at"`
}

type Category struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Provider struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Transaction struct {
	ID            string     `json:"id"`
	UserID        string     `json:"user_id"`
	Amount        string     `json:"amount"`
	Currency      string     `json:"currency"`
	FromAccountID *string    `json:"from_account_id"`
	ToAccountID   *string    `json:"to_account_id"`
	ProviderID    *string    `json:"provider_id"`
	CategoryID    *string    `json:"category_id"`
	Type          string     `json:"type"`
	Status        string     `json:"status"`
	Description   *string    `json:"description"`
	ExternalID    *string    `json:"external_id"`
	OccurredAt    time.Time  `json:"occurred_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at"`
}
