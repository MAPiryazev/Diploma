package validator

import "testing"

func TestValidateTransactionAmount(t *testing.T) {
	tests := []struct {
		name    string
		amount  string
		wantErr bool
	}{
		{name: "valid", amount: "123.45"},
		{name: "trimmed valid", amount: " 42.00 "},
		{name: "required", amount: "", wantErr: true},
		{name: "non numeric", amount: "abc", wantErr: true},
		{name: "non positive", amount: "0", wantErr: true},
		{name: "above max", amount: "1000000000.00", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTransactionAmount(tt.amount)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateTransactionAmount(%q) error = %v, wantErr %v", tt.amount, err, tt.wantErr)
			}
		})
	}
}

func TestValidateDateRange(t *testing.T) {
	validFrom := "2025-01-01T00:00:00Z"
	validTo := "2025-01-02T00:00:00Z"
	tooFarTo := "2036-01-02T00:00:01Z"

	tests := []struct {
		name    string
		from    string
		to      string
		wantErr bool
	}{
		{name: "valid", from: validFrom, to: validTo},
		{name: "missing bounds", from: "", to: validTo, wantErr: true},
		{name: "invalid from", from: "bad", to: validTo, wantErr: true},
		{name: "invalid to", from: validFrom, to: "bad", wantErr: true},
		{name: "from after to", from: validTo, to: validFrom, wantErr: true},
		{name: "range too large", from: validFrom, to: tooFarTo, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDateRange(tt.from, tt.to)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateDateRange(%q, %q) error = %v, wantErr %v", tt.from, tt.to, err, tt.wantErr)
			}
		})
	}
}

func TestValidateTransactionAccounts(t *testing.T) {
	tests := []struct {
		name    string
		txType  string
		fromID  string
		toID    string
		wantErr bool
	}{
		{name: "transfer valid", txType: "transfer", fromID: "a1", toID: "a2"},
		{name: "transfer requires from", txType: "transfer", fromID: "", toID: "a2", wantErr: true},
		{name: "transfer requires to", txType: "transfer", fromID: "a1", toID: "", wantErr: true},
		{name: "transfer accounts differ", txType: "transfer", fromID: "a1", toID: "a1", wantErr: true},
		{name: "income requires to", txType: "income", fromID: "", toID: "", wantErr: true},
		{name: "expense requires from", txType: "expense", fromID: "", toID: "", wantErr: true},
		{name: "income valid", txType: "income", toID: "a2"},
		{name: "expense valid", txType: "expense", fromID: "a1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTransactionAccounts(tt.txType, tt.fromID, tt.toID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateTransactionAccounts(%q, %q, %q) error = %v, wantErr %v", tt.txType, tt.fromID, tt.toID, err, tt.wantErr)
			}
		})
	}
}
