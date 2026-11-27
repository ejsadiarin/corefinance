package budget

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ejsadiarin/corefinance/internal/category"
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/rates"
)

const (
	StatusImported = "imported"
	StatusFailed   = "failed"
)

// ImportResult is the shared part result (coregateway ADR-0008). A failed
// import wrote nothing; Errors says why.
type ImportResult struct {
	Part    string                  `json:"part"`
	Format  string                  `json:"format"`
	Version int                     `json:"version"`
	DryRun  bool                    `json:"dry_run"`
	Status  string                  `json:"status"`
	Counts  map[string]*ImportCount `json:"counts"`
	Errors  []ImportError           `json:"errors"`
}

type ImportCount struct {
	Created int `json:"created"`
	Skipped int `json:"skipped"`
}

type ImportError struct {
	Entity string     `json:"entity"`
	ID     *uuid.UUID `json:"id,omitempty"`
	Reason string     `json:"reason"`
}

// importFailure is an error in the file rather than in the service: it
// fails the part and is reported, not logged as a server error.
type importFailure struct{ errs []ImportError }

func (f *importFailure) Error() string { return fmt.Sprintf("%d import errors", len(f.errs)) }

func fail(entity string, id uuid.UUID, reason string) error {
	var p *uuid.UUID
	if id != uuid.Nil {
		p = &id
	}
	return &importFailure{errs: []ImportError{{Entity: entity, ID: p, Reason: reason}}}
}

// rowFailure turns a constraint or data error from one row into a failure
// naming that row; anything else is the service's problem.
func rowFailure(entity string, id uuid.UUID, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (strings.HasPrefix(pgErr.Code, "22") || strings.HasPrefix(pgErr.Code, "23")) {
		return fail(entity, id, pgErr.Message)
	}
	if errors.Is(err, ErrLegacyCurrency) || errors.Is(err, rates.ErrUnsupportedCurrency) {
		return fail(entity, id, err.Error())
	}
	return err
}

// Export is the owner's budget part.
func (s *Service) Export(ctx context.Context, userID uuid.UUID) (*Part, error) {
	tags, err := s.queries.ListTags(ctx, userID)
	if err != nil {
		return nil, err
	}
	rules, err := s.queries.ListAllRulesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	expenses, err := s.queries.ListAllExpensesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	incomes, err := s.queries.ListAllIncomesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	expenseTags, err := s.queries.ListAllExpenseTagsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	incomeTags, err := s.queries.ListAllIncomeTagsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	byExpense := map[uuid.UUID][]uuid.UUID{}
	for _, et := range expenseTags {
		byExpense[et.ExpenseID] = append(byExpense[et.ExpenseID], et.TagID)
	}
	byIncome := map[uuid.UUID][]uuid.UUID{}
	for _, it := range incomeTags {
		byIncome[it.IncomeID] = append(byIncome[it.IncomeID], it.TagID)
	}

	data := BudgetData{
		Tags:     make([]PartTag, 0, len(tags)),
		Rules:    make([]PartRule, 0, len(rules)),
		Expenses: make([]PartExpense, 0, len(expenses)),
		Incomes:  make([]PartIncome, 0, len(incomes)),
	}
	for _, t := range tags {
		data.Tags = append(data.Tags, PartTag{ID: t.ID, Name: t.Name, Color: text(t.Color), CreatedAt: stamp(t.CreatedAt)})
	}
	for _, r := range rules {
		list := category.Expenses
		if r.Kind == "income" {
			list = category.Incomes
		}
		data.Rules = append(data.Rules, partRule(r, systemName(r.CategoryID, list)))
	}
	for _, e := range expenses {
		data.Expenses = append(data.Expenses, partExpense(e, systemName(e.CategoryID, category.Expenses), byExpense[e.ID]))
	}
	for _, i := range incomes {
		data.Incomes = append(data.Incomes, partIncome(i, systemName(i.CategoryID, category.Incomes), byIncome[i.ID]))
	}

	slog.Info("budget.Service.Export: completed", "user_id", userID,
		"expenses", len(data.Expenses), "incomes", len(data.Incomes), "rules", len(data.Rules))
	return &Part{
		Schema:     PartSchema,
		Format:     PartFormat,
		Version:    PartVersion,
		ExportedAt: time.Now().UTC().Truncate(time.Second),
		Source:     PartSource{Service: "corefinance"},
		Data:       data,
	}, nil
}

// Import loads a budget part into the owner's data in one transaction, all
// or nothing (coregateway ADR-0008). Ids are kept, unless the file is
// another account's: then every row gets an id derived from the owner (see
// adoptIDs). A row the owner already has is skipped. A dry run does the
// same work and rolls it back. The error is the service's; a
// problem with the file is a failed result.
func (s *Service) Import(ctx context.Context, userID uuid.UUID, data *BudgetData, version int, dryRun bool) (*ImportResult, error) {
	res := &ImportResult{
		Part: PartName, Format: PartFormat, Version: version, DryRun: dryRun, Status: StatusImported,
		Counts: map[string]*ImportCount{"tags": {}, "rules": {}, "expenses": {}, "incomes": {}},
		Errors: []ImportError{},
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	err = s.importInto(ctx, db.New(tx), userID, data, res.Counts)
	var failure *importFailure
	switch {
	case errors.As(err, &failure):
		res.Status = StatusFailed
		res.Errors = failure.errs
		for _, c := range res.Counts {
			*c = ImportCount{}
		}
		slog.Info("budget.Service.Import: file refused", "user_id", userID, "errors", len(failure.errs))
		return res, nil
	case err != nil:
		return nil, err
	}

	if dryRun {
		return res, nil
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	slog.Info("budget.Service.Import: completed", "user_id", userID, "version", version)
	return res, nil
}

func (s *Service) importInto(ctx context.Context, q *db.Queries, userID uuid.UUID, data *BudgetData, counts map[string]*ImportCount) error {
	if err := checkIDs(data); err != nil {
		return err
	}
	if err := adoptIDs(ctx, q, userID, data); err != nil {
		return err
	}
	if err := checkForeignIDs(ctx, q, userID, data); err != nil {
		return err
	}

	tagIDs, err := importTags(ctx, q, userID, data.Tags, counts["tags"])
	if err != nil {
		return err
	}
	// A record's category with no system match becomes `other` plus a tag
	// of its name: the owner's tag of that name, or a new one.
	categoryTag := func(name string) (uuid.UUID, error) {
		key := strings.ToLower(name)
		if id, ok := tagIDs.byName[key]; ok {
			return id, nil
		}
		t, err := q.CreateTag(ctx, db.CreateTagParams{UserID: userID, Name: name})
		if err != nil {
			return uuid.Nil, err
		}
		counts["tags"].Created++
		tagIDs.byName[key] = t.ID
		return t.ID, nil
	}

	for _, r := range data.Rules {
		list, other := category.Expenses, category.ExpenseOther
		if r.Kind == "income" {
			list, other = category.Incomes, category.IncomeOther
		}
		if r.Kind != "expense" && r.Kind != "income" {
			return fail("rule", r.ID, fmt.Sprintf("kind %q is not expense or income", r.Kind))
		}
		if err := s.checkImportCurrency(ctx, r.Currency); err != nil {
			return rowFailure("rule", r.ID, err)
		}
		start, err := parseDate(r.StartDate)
		if err != nil || !start.Valid {
			return fail("rule", r.ID, "start_date is not a date")
		}
		end, err := parseOptDate(r.EndDate)
		if err != nil {
			return fail("rule", r.ID, "end_date is not a date")
		}
		through, err := parseOptDate(r.MaterializedThrough)
		if err != nil {
			return fail("rule", r.ID, "materialized_through is not a date")
		}
		n, err := q.ImportRule(ctx, db.ImportRuleParams{
			ID: r.ID, Kind: r.Kind, UserID: userID, Description: r.Description, Amount: r.Amount,
			Currency: rates.Normalize(r.Currency), CategoryID: pgUUID(categoryFor(list, other, r.Category).ID),
			Notes: pgText(r.Notes), RecurringType: r.Recurrence, StartDate: start, EndDate: end,
			Priority: r.Priority, IsActive: !r.Paused, MaterializedThrough: through, CreatedAt: pgStamp(r.CreatedAt),
		})
		if err != nil {
			return rowFailure("rule", r.ID, err)
		}
		count(counts["rules"], n)
	}

	for _, e := range data.Expenses {
		cat := categoryFor(category.Expenses, category.ExpenseOther, e.Category)
		p, err := s.recordParams(ctx, "expense", e.PartRecord)
		if err != nil {
			return err
		}
		n, err := q.ImportExpense(ctx, db.ImportExpenseParams{
			ID: e.ID, UserID: userID, CategoryID: pgUUID(cat.ID), Amount: e.Amount, Currency: p.conv.Currency,
			ExchangeRate: p.conv.Rate, AmountPhp: p.conv.AmountPHP, RateDate: pgtype.Date{Time: p.conv.RateDate, Valid: true},
			Description: e.Description, Notes: pgText(e.Notes), ExpenseDate: p.date, Priority: e.Priority,
			Status: e.Status, IsDebt: e.IsDebt, SourceRuleID: pgOptUUID(e.RuleID), UserEditedAt: pgStamp(e.EditedAt),
			OccurredAt: pgStamp(e.OccurredAt), CreatedAt: pgStamp(e.CreatedAt),
		})
		if err != nil {
			return rowFailure("expense", e.ID, err)
		}
		count(counts["expenses"], n)
		if n == 0 {
			continue // the owner's own copy stays as it is
		}
		for _, tag := range recordTags(e.Tags, tagIDs.byFileID) {
			if err := q.AddTagToExpense(ctx, db.AddTagToExpenseParams{ExpenseID: e.ID, TagID: tag}); err != nil {
				return err
			}
		}
		if cat.Tag != "" {
			tag, err := categoryTag(cat.Tag)
			if err != nil {
				return err
			}
			if err := q.AddTagToExpense(ctx, db.AddTagToExpenseParams{ExpenseID: e.ID, TagID: tag}); err != nil {
				return err
			}
		}
	}

	for _, i := range data.Incomes {
		cat := categoryFor(category.Incomes, category.IncomeOther, i.Category)
		p, err := s.recordParams(ctx, "income", i.PartRecord)
		if err != nil {
			return err
		}
		n, err := q.ImportIncome(ctx, db.ImportIncomeParams{
			ID: i.ID, UserID: userID, CategoryID: pgUUID(cat.ID), Amount: i.Amount, Currency: p.conv.Currency,
			ExchangeRate: p.conv.Rate, AmountPhp: p.conv.AmountPHP, RateDate: pgtype.Date{Time: p.conv.RateDate, Valid: true},
			Description: i.Description, Notes: pgText(i.Notes), Date: p.date, Priority: i.Priority,
			Status: i.Status, SourceRuleID: pgOptUUID(i.RuleID), UserEditedAt: pgStamp(i.EditedAt),
			OccurredAt: pgStamp(i.OccurredAt), CreatedAt: pgStamp(i.CreatedAt),
		})
		if err != nil {
			return rowFailure("income", i.ID, err)
		}
		count(counts["incomes"], n)
		if n == 0 {
			continue
		}
		for _, tag := range recordTags(i.Tags, tagIDs.byFileID) {
			if err := q.AddTagToIncome(ctx, db.AddTagToIncomeParams{IncomeID: i.ID, TagID: tag}); err != nil {
				return err
			}
		}
		if cat.Tag != "" {
			tag, err := categoryTag(cat.Tag)
			if err != nil {
				return err
			}
			if err := q.AddTagToIncome(ctx, db.AddTagToIncomeParams{IncomeID: i.ID, TagID: tag}); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkIDs refuses a file with a missing or repeated id.
func checkIDs(data *BudgetData) error {
	seen := map[uuid.UUID]bool{}
	check := func(entity string, id uuid.UUID) error {
		if id == uuid.Nil {
			return fail(entity, id, "id is missing")
		}
		if seen[id] {
			return fail(entity, id, "id appears more than once in the file")
		}
		seen[id] = true
		return nil
	}
	for _, t := range data.Tags {
		if err := check("tag", t.ID); err != nil {
			return err
		}
	}
	for _, r := range data.Rules {
		if err := check("rule", r.ID); err != nil {
			return err
		}
	}
	for _, e := range data.Expenses {
		if err := check("expense", e.ID); err != nil {
			return err
		}
	}
	for _, i := range data.Incomes {
		if err := check("income", i.ID); err != nil {
			return err
		}
	}
	return nil
}

// adoptIDs decides whether the file is a copy of another account's data
// and, if so, gives every row the id it derives for this owner, links and
// all (coregateway ADR-0008). It is a copy when any of its ids is held by
// another account, or when the owner already holds any derived id (an
// earlier copy of it). Otherwise the ids are kept, so restoring into the
// same account is unchanged.
func adoptIDs(ctx context.Context, q *db.Queries, userID uuid.UUID, data *BudgetData) error {
	var ids, derived []uuid.UUID
	add := func(id uuid.UUID) {
		ids = append(ids, id)
		derived = append(derived, DeriveID(userID, id))
	}
	for _, t := range data.Tags {
		add(t.ID)
	}
	for _, r := range data.Rules {
		add(r.ID)
	}
	for _, e := range data.Expenses {
		add(e.ID)
	}
	for _, i := range data.Incomes {
		add(i.ID)
	}
	if len(ids) == 0 {
		return nil
	}
	foreign, err := q.ListForeignBudgetIDs(ctx, db.ListForeignBudgetIDsParams{UserID: userID, TagIds: ids,
		RuleIds: ids, ExpenseIds: ids, IncomeIds: ids})
	if err != nil {
		return err
	}
	if len(foreign) == 0 {
		copied, err := q.ListOwnBudgetIDs(ctx, db.ListOwnBudgetIDsParams{UserID: userID, Ids: derived})
		if err != nil {
			return err
		}
		if len(copied) == 0 {
			return nil
		}
	}

	to := func(id uuid.UUID) uuid.UUID { return DeriveID(userID, id) }
	toAll := func(ids []uuid.UUID) []uuid.UUID {
		out := make([]uuid.UUID, len(ids))
		for i, id := range ids {
			out[i] = to(id)
		}
		return out
	}
	toOpt := func(id *uuid.UUID) *uuid.UUID {
		if id == nil {
			return nil
		}
		v := to(*id)
		return &v
	}
	for i := range data.Tags {
		data.Tags[i].ID = to(data.Tags[i].ID)
	}
	for i := range data.Rules {
		data.Rules[i].ID = to(data.Rules[i].ID)
	}
	for i := range data.Expenses {
		e := &data.Expenses[i]
		e.ID, e.RuleID, e.Tags = to(e.ID), toOpt(e.RuleID), toAll(e.Tags)
	}
	for i := range data.Incomes {
		r := &data.Incomes[i]
		r.ID, r.RuleID, r.Tags = to(r.ID), toOpt(r.RuleID), toAll(r.Tags)
	}
	return nil
}

// DeriveID is the id a row copied from another account gets in the owner's:
// a name-based UUID (version 5) of the original id in the owner's
// namespace. Every service derives the same way, so links between parts
// still meet (coregateway ADR-0008).
func DeriveID(owner, id uuid.UUID) uuid.UUID {
	return uuid.NewSHA1(owner, id[:])
}

// checkForeignIDs fails the import when any id in the file is another
// owner's: importing must never touch, or collide with, their data. After
// adoptIDs only a derived id could be, which no real file reaches.
func checkForeignIDs(ctx context.Context, q *db.Queries, userID uuid.UUID, data *BudgetData) error {
	p := db.ListForeignBudgetIDsParams{UserID: userID, TagIds: []uuid.UUID{}, RuleIds: []uuid.UUID{},
		ExpenseIds: []uuid.UUID{}, IncomeIds: []uuid.UUID{}}
	for _, t := range data.Tags {
		p.TagIds = append(p.TagIds, t.ID)
	}
	for _, r := range data.Rules {
		p.RuleIds = append(p.RuleIds, r.ID)
	}
	for _, e := range data.Expenses {
		p.ExpenseIds = append(p.ExpenseIds, e.ID)
	}
	for _, i := range data.Incomes {
		p.IncomeIds = append(p.IncomeIds, i.ID)
	}
	rows, err := q.ListForeignBudgetIDs(ctx, p)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	f := &importFailure{}
	for _, r := range rows {
		id := r.ID
		f.errs = append(f.errs, ImportError{Entity: r.Entity, ID: &id, Reason: "id belongs to another user"})
	}
	return f
}

type importedTags struct {
	byFileID map[uuid.UUID]uuid.UUID
	byName   map[string]uuid.UUID
}

// importTags keeps each tag's id, except that a tag whose name the owner
// already has maps onto the owner's tag, so a name never appears twice.
func importTags(ctx context.Context, q *db.Queries, userID uuid.UUID, tags []PartTag, c *ImportCount) (*importedTags, error) {
	own, err := q.ListTags(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := &importedTags{byFileID: map[uuid.UUID]uuid.UUID{}, byName: map[string]uuid.UUID{}}
	ownIDs := map[uuid.UUID]bool{}
	for _, t := range own {
		ownIDs[t.ID] = true
		key := strings.ToLower(strings.TrimSpace(t.Name))
		if _, seen := out.byName[key]; !seen {
			out.byName[key] = t.ID
		}
	}
	for _, t := range tags {
		key := strings.ToLower(strings.TrimSpace(t.Name))
		if key == "" {
			return nil, fail("tag", t.ID, "name is empty")
		}
		if ownIDs[t.ID] {
			out.byFileID[t.ID] = t.ID
			c.Skipped++
			continue
		}
		if id, ok := out.byName[key]; ok {
			out.byFileID[t.ID] = id
			c.Skipped++
			continue
		}
		n, err := q.ImportTag(ctx, db.ImportTagParams{ID: t.ID, UserID: userID, Name: strings.TrimSpace(t.Name),
			Color: pgText(t.Color), CreatedAt: pgStamp(t.CreatedAt)})
		if err != nil {
			return nil, rowFailure("tag", t.ID, err)
		}
		count(c, n)
		out.byFileID[t.ID] = t.ID
		out.byName[key] = t.ID
	}
	return out, nil
}

type recordParams struct {
	date pgtype.Date
	conv rates.Conversion
}

func (s *Service) recordParams(ctx context.Context, entity string, r PartRecord) (recordParams, error) {
	d, err := parseDate(r.Date)
	if err != nil || !d.Valid {
		return recordParams{}, fail(entity, r.ID, "date is not a date")
	}
	rateDate, err := parseOptDate(r.RateDate)
	if err != nil {
		return recordParams{}, fail(entity, r.ID, "rate_date is not a date")
	}
	if err := s.checkImportCurrency(ctx, r.Currency); err != nil {
		return recordParams{}, rowFailure(entity, r.ID, err)
	}
	conv, err := importedConversion(r.Amount, r.Currency, r.ExchangeRate, r.AmountPHP, rateDate, d)
	if err != nil {
		return recordParams{}, rowFailure(entity, r.ID, err)
	}
	return recordParams{date: d, conv: conv}, nil
}

// recordTags is a record's tags as the owner's tag ids; a tag the file
// doesn't list is dropped.
func recordTags(ids []uuid.UUID, byFileID map[uuid.UUID]uuid.UUID) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(ids))
	seen := map[uuid.UUID]bool{}
	for _, id := range ids {
		if t, ok := byFileID[id]; ok && !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}

func count(c *ImportCount, rows int64) {
	if rows > 0 {
		c.Created++
	} else {
		c.Skipped++
	}
}

func parseDate(s string) (pgtype.Date, error) {
	if s == "" {
		return pgtype.Date{}, nil
	}
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return pgtype.Date{}, err
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}

func parseOptDate(s *string) (pgtype.Date, error) {
	if s == nil {
		return pgtype.Date{}, nil
	}
	return parseDate(*s)
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: id != uuid.Nil}
}

func pgOptUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgUUID(*id)
}

func pgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func pgStamp(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
