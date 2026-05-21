package repository

import (
	"testing"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
)

func TestBuildBalanceAdjustments(t *testing.T) {
	fromID := "from-account"
	toID := "to-account"

	tests := []struct {
		name    string
		tx      *models.Transaction
		reverse bool
		want    []balanceAdjustment
	}{
		{
			name: "done transfer applies debit then credit",
			tx: &models.Transaction{
				UserID:        "user-1",
				Amount:        "500.00",
				Type:          "transfer",
				Status:        "done",
				FromAccountID: &fromID,
				ToAccountID:   &toID,
			},
			want: []balanceAdjustment{
				{AccountID: fromID, UserID: "user-1", Amount: "500.00", Field: "from_account_id", Kind: balanceAdjustmentDebit},
				{AccountID: toID, UserID: "user-1", Amount: "500.00", Field: "to_account_id", Kind: balanceAdjustmentCredit},
			},
		},
		{
			name: "reverse transfer restores destination then source",
			tx: &models.Transaction{
				UserID:        "user-1",
				Amount:        "500.00",
				Type:          "transfer",
				Status:        "done",
				FromAccountID: &fromID,
				ToAccountID:   &toID,
			},
			reverse: true,
			want: []balanceAdjustment{
				{AccountID: toID, UserID: "user-1", Amount: "500.00", Field: "to_account_id", Kind: balanceAdjustmentDebit},
				{AccountID: fromID, UserID: "user-1", Amount: "500.00", Field: "from_account_id", Kind: balanceAdjustmentCredit},
			},
		},
		{
			name: "done income credits destination account",
			tx: &models.Transaction{
				UserID:      "user-1",
				Amount:      "1200.00",
				Type:        "income",
				Status:      "done",
				ToAccountID: &toID,
			},
			want: []balanceAdjustment{
				{AccountID: toID, UserID: "user-1", Amount: "1200.00", Field: "to_account_id", Kind: balanceAdjustmentCredit},
			},
		},
		{
			name: "pending transaction has no balance effect",
			tx: &models.Transaction{
				UserID:        "user-1",
				Amount:        "10.00",
				Type:          "transfer",
				Status:        "pending",
				FromAccountID: &fromID,
				ToAccountID:   &toID,
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildBalanceAdjustments(tt.tx, tt.reverse)
			if len(got) != len(tt.want) {
				t.Fatalf("len(got)=%d len(want)=%d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("adjustment[%d]=%+v want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
