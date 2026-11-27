package budget

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// writeAndRead sends a part through its JSON file, as a download and an
// upload would.
func writeAndRead(t *testing.T, part *Part) (*BudgetData, int) {
	t.Helper()
	file, err := json.Marshal(part)
	require.NoError(t, err)
	data, version, err := ReadPart(file)
	require.NoError(t, err)
	return data, version
}

// clearOwner deletes an owner's budget data, so their exported ids are free
// and another owner's import keeps them.
func clearOwner(t *testing.T, pool *pgxpool.Pool, owner uuid.UUID) {
	t.Helper()
	for _, stmt := range []string{
		`DELETE FROM expenses WHERE user_id = $1`,
		`DELETE FROM incomes WHERE user_id = $1`,
		`DELETE FROM rules WHERE user_id = $1`,
		`DELETE FROM tags WHERE user_id = $1`,
	} {
		_, err := pool.Exec(context.Background(), stmt, owner)
		require.NoError(t, err)
	}
}

// importOK imports and requires the part to be imported.
func importOK(t *testing.T, svc *Service, owner uuid.UUID, data *BudgetData, version int) *ImportResult {
	t.Helper()
	res, err := svc.Import(context.Background(), owner, data, version, false)
	require.NoError(t, err)
	require.Equal(t, StatusImported, res.Status, "errors: %+v", res.Errors)
	return res
}
