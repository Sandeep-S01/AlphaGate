package execution

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type AccountRepository struct {
	db QueryRower
}

func NewAccountRepository(db QueryRower) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) Get(ctx context.Context) (Account, error) {
	query := `
SELECT base_asset, quote_asset, base_balance, quote_balance, updated_at
FROM paper_accounts
ORDER BY created_at ASC
LIMIT 1`
	var account Account
	if err := r.db.QueryRow(ctx, query).Scan(
		&account.BaseAsset,
		&account.QuoteAsset,
		&account.BaseBalance,
		&account.QuoteBalance,
		&account.UpdatedAt,
	); err != nil {
		return Account{}, fmt.Errorf("query paper account: %w", err)
	}
	return account, nil
}

func (r *AccountRepository) Save(ctx context.Context, account Account) error {
	query, args := BuildResetAccountSQL(account)
	var id string
	err := r.db.QueryRow(ctx, query, args...).Scan(&id)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("save paper account: %w", err)
	}
	return nil
}

func BuildResetAccountSQL(account Account) (string, []any) {
	return `
UPDATE paper_accounts
SET base_asset = $1,
    quote_asset = $2,
    base_balance = $3,
    quote_balance = $4,
    updated_at = NOW()
WHERE id = (
    SELECT id FROM paper_accounts ORDER BY created_at ASC LIMIT 1
) RETURNING id`, []any{
			account.BaseAsset,
			account.QuoteAsset,
			account.BaseBalance,
			account.QuoteBalance,
		}
}
