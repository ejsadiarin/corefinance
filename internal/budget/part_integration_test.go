package budget

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ejsadiarin/corefinance/internal/category"
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/materialize"
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

// seedBudget gives an owner one of everything a part carries: a tag, a
// daily rule with its entries through 2026-11-24 (one edited), a tagged
// USD expense and an income.
func seedBudget(t *testing.T, pool *pgxpool.Pool, owner uuid.UUID) (ruleID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	q := db.New(pool)
	food, _ := category.ByName(category.Expenses, "food")
	day := func(s string) pgtype.Date {
		d := pgtype.Date{}
		require.NoError(t, d.Scan(s))
		return d
	}

	tag, err := q.CreateTag(ctx, db.CreateTagParams{UserID: owner, Name: "essential", Color: pgtype.Text{String: "#00ff00", Valid: true}})
	require.NoError(t, err)

	ruleID = uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO rules (id, kind, user_id, description, amount, currency, category_id, recurring_type,
		                   start_date, priority, is_active, materialized_through, next_due_at)
		VALUES ($1, 'expense', $2, 'coffee', 120, 'PHP', $3, 'daily', '2026-11-20', 'want', true, '2026-11-24',
		        '2026-11-25T00:00:00Z')`, ruleID, owner, food.ID)
	require.NoError(t, err)
	for _, d := range []string{"2026-11-20", "2026-11-21", "2026-11-22", "2026-11-23", "2026-11-24"} {
		_, err := q.CreateExpense(ctx, db.CreateExpenseParams{
			UserID: owner, CategoryID: pgtype.UUID{Bytes: food.ID, Valid: true}, Amount: decimal.NewFromInt(120),
			Currency: "PHP", ExchangeRate: decimal.NewFromInt(1), AmountPhp: decimal.NewFromInt(120), RateDate: day(d),
			Description: "coffee", ExpenseDate: day(d), Priority: "want", Status: "posted",
			SourceRuleID: pgtype.UUID{Bytes: ruleID, Valid: true},
		})
		require.NoError(t, err)
	}
	_, err = pool.Exec(ctx, `UPDATE expenses SET user_edited_at = now(), amount = 150, amount_php = 150
		WHERE source_rule_id = $1 AND expense_date = '2026-11-22'`, ruleID)
	require.NoError(t, err)

	usd, err := q.CreateExpense(ctx, db.CreateExpenseParams{
		UserID: owner, Amount: decimal.NewFromInt(10), Currency: "USD",
		ExchangeRate: decimal.RequireFromString("58.5"), AmountPhp: decimal.RequireFromString("585.00"),
		RateDate: day("2026-11-23"), Description: "app", ExpenseDate: day("2026-11-24"), Priority: "want",
		Status: "posted", IsDebt: true,
	})
	require.NoError(t, err)
	require.NoError(t, q.AddTagToExpense(ctx, db.AddTagToExpenseParams{ExpenseID: usd.ID, TagID: tag.ID}))

	_, err = q.CreateIncome(ctx, db.CreateIncomeParams{
		UserID: owner, Amount: decimal.NewFromInt(1000), Currency: "PHP", ExchangeRate: decimal.NewFromInt(1),
		AmountPhp: decimal.NewFromInt(1000), RateDate: day("2026-11-24"), Description: "pay", Date: day("2026-11-24"),
		Priority: "need", Status: "posted",
	})
	require.NoError(t, err)
	return ruleID
}

func TestBudgetPart_ValidatesAgainstItsSchema_Integration(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	svc := NewServiceWithRates(db.New(pool), pool, importRates{})
	owner := uuid.New()
	seedBudget(t, pool, owner)

	part, err := svc.Export(context.Background(), owner)
	require.NoError(t, err)
	file, err := json.Marshal(part)
	require.NoError(t, err)

	schemaFile, err := os.ReadFile("../../schemas/budget.v1.json")
	require.NoError(t, err)
	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaFile))
	require.NoError(t, err)
	c := jsonschema.NewCompiler()
	c.AssertFormat()
	require.NoError(t, c.AddResource(PartSchema, schemaDoc))
	schema, err := c.Compile(PartSchema)
	require.NoError(t, err)
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(file))
	require.NoError(t, err)
	require.NoError(t, schema.Validate(inst))

	// The schema is strict enough to notice a stray database column.
	var loose map[string]any
	require.NoError(t, json.Unmarshal(file, &loose))
	loose["data"].(map[string]any)["expenses"].([]any)[0].(map[string]any)["user_id"] = owner.String()
	require.Error(t, schema.Validate(loose))
}

// Importing your own export changes nothing: every row is already yours.
func TestBudgetPart_ReimportSkipsEverything_Integration(t *testing.T) {
	ctx := context.Background()
	pool := testutil.SetupTestDB(t)
	q := db.New(pool)
	svc := NewServiceWithRates(q, pool, importRates{})
	owner := uuid.New()
	seedBudget(t, pool, owner)
	before, err := q.ListAllExpensesByUser(ctx, owner)
	require.NoError(t, err)

	part, err := svc.Export(ctx, owner)
	require.NoError(t, err)
	data, version := writeAndRead(t, part)
	res := importOK(t, svc, owner, data, version)

	for entity, c := range res.Counts {
		assert.Zero(t, c.Created, entity)
	}
	assert.Equal(t, ImportCount{Skipped: 6}, *res.Counts["expenses"])
	assert.Equal(t, ImportCount{Skipped: 1}, *res.Counts["rules"])
	after, err := q.ListAllExpensesByUser(ctx, owner)
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

// Into an owner without the data, an import brings back the same rows with
// the same ids, and the rule carries on after its watermark: the worker
// fills no date the file already holds.
func TestBudgetPart_ImportIntoAnEmptyOwner_Integration(t *testing.T) {
	ctx := context.Background()
	pool := testutil.SetupTestDB(t)
	q := db.New(pool)
	svc := NewServiceWithRates(q, pool, importRates{})
	source, target := uuid.New(), uuid.New()
	ruleID := seedBudget(t, pool, source)

	part, err := svc.Export(ctx, source)
	require.NoError(t, err)
	data, version := writeAndRead(t, part)
	clearOwner(t, pool, source)
	res := importOK(t, svc, target, data, version)
	assert.Equal(t, ImportCount{Created: 6}, *res.Counts["expenses"])
	assert.Equal(t, ImportCount{Created: 1}, *res.Counts["incomes"])
	assert.Equal(t, ImportCount{Created: 1}, *res.Counts["rules"])
	assert.Equal(t, ImportCount{Created: 1}, *res.Counts["tags"])

	again, err := svc.Export(ctx, target)
	require.NoError(t, err)
	assert.Equal(t, part.Data, again.Data, "the same rows, ids, tags and rule state")

	var nextDue pgtype.Timestamptz
	require.NoError(t, pool.QueryRow(ctx, `SELECT next_due_at FROM rules WHERE id = $1`, ruleID).Scan(&nextDue))
	assert.False(t, nextDue.Valid, "next_due_at is recomputed, not imported")

	n, err := materialize.MaterializeUser(ctx, q, target, time.UTC, time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Zero(t, n, "no entry is filled for a date the file holds")
}

// A file from another account is copied: the source keeps its rows, and the
// target gets the same data under ids derived for it, with its links
// (rule entries, tags) pointing at the copies. Importing it again, even
// after the source has deleted everything, adds nothing.
func TestBudgetPart_CopiesAnotherAccountsFile_Integration(t *testing.T) {
	ctx := context.Background()
	pool := testutil.SetupTestDB(t)
	q := db.New(pool)
	svc := NewServiceWithRates(q, pool, importRates{})
	source, target := uuid.New(), uuid.New()
	ruleID := seedBudget(t, pool, source)
	before, err := q.ListAllExpensesByUser(ctx, source)
	require.NoError(t, err)

	part, err := svc.Export(ctx, source)
	require.NoError(t, err)
	data, version := writeAndRead(t, part)
	res := importOK(t, svc, target, data, version)
	assert.Equal(t, ImportCount{Created: 6}, *res.Counts["expenses"])
	assert.Equal(t, ImportCount{Created: 1}, *res.Counts["incomes"])
	assert.Equal(t, ImportCount{Created: 1}, *res.Counts["rules"])
	assert.Equal(t, ImportCount{Created: 1}, *res.Counts["tags"])

	after, err := q.ListAllExpensesByUser(ctx, source)
	require.NoError(t, err)
	assert.Equal(t, before, after, "the source's rows are untouched")

	copied, err := svc.Export(ctx, target)
	require.NoError(t, err)
	want, _ := writeAndRead(t, part)
	for i := range want.Tags {
		want.Tags[i].ID = DeriveID(target, want.Tags[i].ID)
	}
	for i := range want.Rules {
		want.Rules[i].ID = DeriveID(target, want.Rules[i].ID)
	}
	for i := range want.Expenses {
		e := &want.Expenses[i]
		e.ID = DeriveID(target, e.ID)
		if e.RuleID != nil {
			id := DeriveID(target, *e.RuleID)
			e.RuleID = &id
		}
		for j := range e.Tags {
			e.Tags[j] = DeriveID(target, e.Tags[j])
		}
	}
	for i := range want.Incomes {
		want.Incomes[i].ID = DeriveID(target, want.Incomes[i].ID)
	}
	got, _ := writeAndRead(t, copied)
	assert.Equal(t, want, got, "the same data under derived ids, links included")

	var entries int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM expenses WHERE user_id = $1 AND source_rule_id = $2`,
		target, DeriveID(target, ruleID)).Scan(&entries))
	assert.Equal(t, 5, entries, "the rule's entries follow the copied rule")

	for _, clear := range []bool{false, true} {
		if clear {
			clearOwner(t, pool, source)
		}
		data, version := writeAndRead(t, part)
		res := importOK(t, svc, target, data, version)
		for entity, c := range res.Counts {
			assert.Zero(t, c.Created, "%s (source cleared: %v)", entity, clear)
		}
	}
}

// A dry run reports what an import would do and writes nothing.
func TestBudgetPart_DryRunWritesNothing_Integration(t *testing.T) {
	ctx := context.Background()
	pool := testutil.SetupTestDB(t)
	q := db.New(pool)
	svc := NewServiceWithRates(q, pool, importRates{})
	source, target := uuid.New(), uuid.New()
	seedBudget(t, pool, source)

	part, err := svc.Export(ctx, source)
	require.NoError(t, err)
	data, version := writeAndRead(t, part)
	clearOwner(t, pool, source)

	dry, err := svc.Import(ctx, target, data, version, true)
	require.NoError(t, err)
	assert.True(t, dry.DryRun)
	assert.Equal(t, StatusImported, dry.Status)
	assert.Equal(t, ImportCount{Created: 6}, *dry.Counts["expenses"])
	got, err := q.ListAllExpensesByUser(ctx, target)
	require.NoError(t, err)
	assert.Empty(t, got)

	real := importOK(t, svc, target, data, version)
	assert.Equal(t, dry.Counts, real.Counts)
}

// The unversioned export that came before version 1 still imports, ids and
// all; a version from the future does not.
func TestBudgetPart_ReadsVersionZeroAndRefusesNewer_Integration(t *testing.T) {
	ctx := context.Background()
	pool := testutil.SetupTestDB(t)
	q := db.New(pool)
	svc := NewServiceWithRates(q, pool, importRates{})
	owner := uuid.New()
	expID, tagID, ruleID, catID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	v0 := []byte(`{
		"expense_categories": [{"id": "` + catID.String() + `", "name": "Food"}],
		"income_categories": [],
		"tags": [{"id": "` + tagID.String() + `", "user_id": "` + uuid.NewString() + `", "name": "essential", "color": null}],
		"expenses": [{"id": "` + expID.String() + `", "user_id": "` + uuid.NewString() + `",
			"category_id": "` + catID.String() + `", "amount": "120", "currency": "PHP", "description": "coffee",
			"notes": null, "expense_date": "2026-11-20", "priority": "want", "status": "posted", "is_debt": false,
			"source_rule_id": "` + ruleID.String() + `", "exchange_rate": "1", "amount_php": "120", "rate_date": "2026-11-20"}],
		"incomes": [],
		"recurring_expenses": [{"id": "` + ruleID.String() + `", "description": "coffee", "amount": "120",
			"currency": "PHP", "category_id": "` + catID.String() + `", "recurring_type": "daily",
			"start_date": "2026-11-20", "end_date": null, "priority": "want", "is_active": true,
			"materialized_through": "2026-11-20"}],
		"recurring_incomes": [],
		"expense_tags": [{"expense_id": "` + expID.String() + `", "tag_id": "` + tagID.String() + `"}],
		"income_tags": []
	}`)
	data, version, err := ReadPart(v0)
	require.NoError(t, err)
	assert.Equal(t, 0, version)
	res := importOK(t, svc, owner, data, version)
	assert.Equal(t, 0, res.Version)

	got, err := q.ListAllExpensesByUser(ctx, owner)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, expID, got[0].ID)
	food, _ := category.ByName(category.Expenses, "food")
	assert.Equal(t, food.ID, uuid.UUID(got[0].CategoryID.Bytes))
	assert.Equal(t, ruleID, uuid.UUID(got[0].SourceRuleID.Bytes), "the entry still belongs to its rule")
	tags, err := q.ListAllExpenseTagsByUser(ctx, owner)
	require.NoError(t, err)
	assert.Equal(t, []db.ExpenseTag{{ExpenseID: expID, TagID: tagID}}, tags)

	_, _, err = ReadPart([]byte(`{"format": "coregateway.budget", "version": 2, "data": {}}`))
	require.ErrorIs(t, err, ErrUnreadable)
	_, _, err = ReadPart([]byte(`{"format": "coregateway.kanban", "version": 1, "data": {}}`))
	require.ErrorIs(t, err, ErrUnreadable)
	_, _, err = ReadPart([]byte(`{"hello": "world"}`))
	require.ErrorIs(t, err, ErrUnreadable)
}
