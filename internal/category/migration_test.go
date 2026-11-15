package category

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ejsadiarin/corefinance/internal/testutil"
)

// Migration 00015 moves every record and rule onto the system categories
// (#23, ADR-0002): same name ignoring case and spaces, else `other` with a
// tag of the old name. The per-owner rows go.
func TestSystemCategoriesMigration_Integration(t *testing.T) {
	ctx := context.Background()
	pool := testutil.SetupTestDBThrough(t, "00014")
	a, b := uuid.New(), uuid.New()

	cat := func(table string, owner uuid.UUID, name string) uuid.UUID {
		t.Helper()
		var id uuid.UUID
		require.NoError(t, pool.QueryRow(ctx,
			`INSERT INTO `+table+` (user_id, name) VALUES ($1, $2) RETURNING id`, owner, name).Scan(&id))
		return id
	}
	expense := func(owner uuid.UUID, category *uuid.UUID, desc string) uuid.UUID {
		t.Helper()
		var id uuid.UUID
		require.NoError(t, pool.QueryRow(ctx,
			`INSERT INTO expenses (user_id, category_id, amount, description, expense_date)
			 VALUES ($1, $2, 100, $3, '2026-11-01') RETURNING id`, owner, category, desc).Scan(&id))
		return id
	}
	income := func(owner uuid.UUID, category *uuid.UUID, desc string) uuid.UUID {
		t.Helper()
		var id uuid.UUID
		require.NoError(t, pool.QueryRow(ctx,
			`INSERT INTO incomes (user_id, category_id, amount, description, date)
			 VALUES ($1, $2, 100, $3, '2026-11-01') RETURNING id`, owner, category, desc).Scan(&id))
		return id
	}

	aFood, bFood := cat("expense_categories", a, " Food "), cat("expense_categories", b, "food")
	aCoffee, bSnacks := cat("expense_categories", a, "Coffee"), cat("expense_categories", b, "Snacks")
	aSalary, bSideGig := cat("income_categories", a, "Salary"), cat("income_categories", b, "Side gig")
	var aCoffeeTag uuid.UUID
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO tags (user_id, name) VALUES ($1, 'COFFEE') RETURNING id`, a).Scan(&aCoffeeTag))

	aLunch, bLunch := expense(a, &aFood, "lunch"), expense(b, &bFood, "lunch")
	aLatte, bChips := expense(a, &aCoffee, "latte"), expense(b, &bSnacks, "chips")
	missing := uuid.New()
	dangling := expense(a, &missing, "points at nothing")
	uncategorized := expense(a, nil, "no category")
	aPay, bGig := income(a, &aSalary, "pay"), income(b, &bSideGig, "gig")

	var coffeeRule uuid.UUID
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO rules (kind, user_id, description, amount, category_id, recurring_type, start_date)
		 VALUES ('expense', $1, 'daily latte', 100, $2, 'daily', '2026-11-01') RETURNING id`, a, aCoffee).Scan(&coffeeRule))
	var salaryRule uuid.UUID
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO rules (kind, user_id, description, amount, category_id, recurring_type, start_date)
		 VALUES ('income', $1, 'pay', 100, $2, 'monthly', '2026-11-01') RETURNING id`, a, aSalary).Scan(&salaryRule))

	testutil.ApplyMigration(t, pool, "00015")

	categoryOf := func(table string, id uuid.UUID) *uuid.UUID {
		t.Helper()
		var c *uuid.UUID
		require.NoError(t, pool.QueryRow(ctx, `SELECT category_id FROM `+table+` WHERE id = $1`, id).Scan(&c))
		return c
	}
	food, _ := ByName(Expenses, "food")
	salary, _ := ByName(Incomes, "salary")

	assert.Equal(t, food.ID, *categoryOf("expenses", aLunch), "' Food ' maps to food")
	assert.Equal(t, food.ID, *categoryOf("expenses", bLunch), "every owner's food is the one food")
	assert.Equal(t, ExpenseOther, *categoryOf("expenses", aLatte))
	assert.Equal(t, ExpenseOther, *categoryOf("expenses", bChips))
	assert.Nil(t, categoryOf("expenses", dangling), "an id that pointed at nothing shows no category")
	assert.Nil(t, categoryOf("expenses", uncategorized))
	assert.Equal(t, salary.ID, *categoryOf("incomes", aPay))
	assert.Equal(t, IncomeOther, *categoryOf("incomes", bGig))
	assert.Equal(t, ExpenseOther, *categoryOf("rules", coffeeRule))
	assert.Equal(t, salary.ID, *categoryOf("rules", salaryRule))

	tagsOf := func(join, col string, id uuid.UUID) []string {
		t.Helper()
		rows, err := pool.Query(ctx, `SELECT t.name FROM tags t JOIN `+join+` j ON j.tag_id = t.id
		                              WHERE j.`+col+` = $1 ORDER BY t.name`, id)
		require.NoError(t, err)
		var names []string
		for rows.Next() {
			var n string
			require.NoError(t, rows.Scan(&n))
			names = append(names, n)
		}
		require.NoError(t, rows.Err())
		return names
	}
	assert.Equal(t, []string{"COFFEE"}, tagsOf("expense_tags", "expense_id", aLatte), "the owner's existing tag is reused")
	assert.Equal(t, []string{"Snacks"}, tagsOf("expense_tags", "expense_id", bChips), "a missing tag is created")
	assert.Equal(t, []string{"Side gig"}, tagsOf("income_tags", "income_id", bGig))
	assert.Empty(t, tagsOf("expense_tags", "expense_id", aLunch), "matched records get no tag")

	var coffeeTags int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM tags WHERE user_id = $1 AND lower(name) = 'coffee'`, a).Scan(&coffeeTags))
	assert.Equal(t, 1, coffeeTags, "no duplicate tag beside the owner's own")

	var owned int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT (SELECT count(*) FROM expense_categories WHERE user_id IS NOT NULL)
		      + (SELECT count(*) FROM income_categories WHERE user_id IS NOT NULL)`).Scan(&owned))
	assert.Zero(t, owned, "the per-owner rows are gone")

	_, err := pool.Exec(ctx, `INSERT INTO expense_categories (user_id, name) VALUES ($1, 'mine')`, a)
	assert.Error(t, err, "owners can't add categories back")
	_, err = pool.Exec(ctx, `UPDATE incomes SET category_id = $1 WHERE id = $2`, food.ID, aPay)
	assert.Error(t, err, "an income can't take an expense category")

	assertSeededListsMatchGo(t, pool)
}

// The rows the migration seeds are exactly the Go lists, in order.
func assertSeededListsMatchGo(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for table, want := range map[string][]System{"expense_categories": Expenses, "income_categories": Incomes} {
		rows, err := pool.Query(context.Background(),
			`SELECT id, name, color, icon FROM `+table+` WHERE is_active ORDER BY id`)
		require.NoError(t, err)
		var got []System
		for rows.Next() {
			var s System
			require.NoError(t, rows.Scan(&s.ID, &s.Name, &s.Color, &s.Icon))
			got = append(got, s)
		}
		require.NoError(t, rows.Err())
		assert.Equal(t, want, got, table)
	}
}
