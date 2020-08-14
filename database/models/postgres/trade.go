// Code generated by SQLBoiler 3.5.1-gct (https://github.com/thrasher-corp/sqlboiler). DO NOT EDIT.
// This file is meant to be re-generated in place and/or deleted at any time.

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/thrasher-corp/sqlboiler/boil"
	"github.com/thrasher-corp/sqlboiler/queries"
	"github.com/thrasher-corp/sqlboiler/queries/qm"
	"github.com/thrasher-corp/sqlboiler/queries/qmhelper"
	"github.com/thrasher-corp/sqlboiler/strmangle"
	"github.com/volatiletech/null"
)

// Trade is an object representing the database table.
type Trade struct {
	ID         string      `boil:"id" json:"id" toml:"id" yaml:"id"`
	ExchangeID null.String `boil:"exchange_id" json:"exchange_id,omitempty" toml:"exchange_id" yaml:"exchange_id,omitempty"`
	Currency   string      `boil:"currency" json:"currency" toml:"currency" yaml:"currency"`
	Asset      string      `boil:"asset" json:"asset" toml:"asset" yaml:"asset"`
	Event      string      `boil:"event" json:"event" toml:"event" yaml:"event"`
	Price      float64     `boil:"price" json:"price" toml:"price" yaml:"price"`
	Amount     float64     `boil:"amount" json:"amount" toml:"amount" yaml:"amount"`
	Side       string      `boil:"side" json:"side" toml:"side" yaml:"side"`
	Timestamp  int64       `boil:"timestamp" json:"timestamp" toml:"timestamp" yaml:"timestamp"`

	R *tradeR `boil:"-" json:"-" toml:"-" yaml:"-"`
	L tradeL  `boil:"-" json:"-" toml:"-" yaml:"-"`
}

var TradeColumns = struct {
	ID         string
	ExchangeID string
	Currency   string
	Asset      string
	Event      string
	Price      string
	Amount     string
	Side       string
	Timestamp  string
}{
	ID:         "id",
	ExchangeID: "exchange_id",
	Currency:   "currency",
	Asset:      "asset",
	Event:      "event",
	Price:      "price",
	Amount:     "amount",
	Side:       "side",
	Timestamp:  "timestamp",
}

// Generated where

type whereHelperfloat64 struct{ field string }

func (w whereHelperfloat64) EQ(x float64) qm.QueryMod { return qmhelper.Where(w.field, qmhelper.EQ, x) }
func (w whereHelperfloat64) NEQ(x float64) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.NEQ, x)
}
func (w whereHelperfloat64) LT(x float64) qm.QueryMod { return qmhelper.Where(w.field, qmhelper.LT, x) }
func (w whereHelperfloat64) LTE(x float64) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.LTE, x)
}
func (w whereHelperfloat64) GT(x float64) qm.QueryMod { return qmhelper.Where(w.field, qmhelper.GT, x) }
func (w whereHelperfloat64) GTE(x float64) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.GTE, x)
}

var TradeWhere = struct {
	ID         whereHelperstring
	ExchangeID whereHelpernull_String
	Currency   whereHelperstring
	Asset      whereHelperstring
	Event      whereHelperstring
	Price      whereHelperfloat64
	Amount     whereHelperfloat64
	Side       whereHelperstring
	Timestamp  whereHelperint64
}{
	ID:         whereHelperstring{field: "\"trade\".\"id\""},
	ExchangeID: whereHelpernull_String{field: "\"trade\".\"exchange_id\""},
	Currency:   whereHelperstring{field: "\"trade\".\"currency\""},
	Asset:      whereHelperstring{field: "\"trade\".\"asset\""},
	Event:      whereHelperstring{field: "\"trade\".\"event\""},
	Price:      whereHelperfloat64{field: "\"trade\".\"price\""},
	Amount:     whereHelperfloat64{field: "\"trade\".\"amount\""},
	Side:       whereHelperstring{field: "\"trade\".\"side\""},
	Timestamp:  whereHelperint64{field: "\"trade\".\"timestamp\""},
}

// TradeRels is where relationship names are stored.
var TradeRels = struct {
	Exchange string
}{
	Exchange: "Exchange",
}

// tradeR is where relationships are stored.
type tradeR struct {
	Exchange *Script
}

// NewStruct creates a new relationship struct
func (*tradeR) NewStruct() *tradeR {
	return &tradeR{}
}

// tradeL is where Load methods for each relationship are stored.
type tradeL struct{}

var (
	tradeAllColumns            = []string{"id", "exchange_id", "currency", "asset", "event", "price", "amount", "side", "timestamp"}
	tradeColumnsWithoutDefault = []string{"exchange_id", "currency", "asset", "event", "price", "amount", "side", "timestamp"}
	tradeColumnsWithDefault    = []string{"id"}
	tradePrimaryKeyColumns     = []string{"id"}
)

type (
	// TradeSlice is an alias for a slice of pointers to Trade.
	// This should generally be used opposed to []Trade.
	TradeSlice []*Trade
	// TradeHook is the signature for custom Trade hook methods
	TradeHook func(context.Context, boil.ContextExecutor, *Trade) error

	tradeQuery struct {
		*queries.Query
	}
)

// Cache for insert, update and upsert
var (
	tradeType                 = reflect.TypeOf(&Trade{})
	tradeMapping              = queries.MakeStructMapping(tradeType)
	tradePrimaryKeyMapping, _ = queries.BindMapping(tradeType, tradeMapping, tradePrimaryKeyColumns)
	tradeInsertCacheMut       sync.RWMutex
	tradeInsertCache          = make(map[string]insertCache)
	tradeUpdateCacheMut       sync.RWMutex
	tradeUpdateCache          = make(map[string]updateCache)
	tradeUpsertCacheMut       sync.RWMutex
	tradeUpsertCache          = make(map[string]insertCache)
)

var (
	// Force time package dependency for automated UpdatedAt/CreatedAt.
	_ = time.Second
	// Force qmhelper dependency for where clause generation (which doesn't
	// always happen)
	_ = qmhelper.Where
)

var tradeBeforeInsertHooks []TradeHook
var tradeBeforeUpdateHooks []TradeHook
var tradeBeforeDeleteHooks []TradeHook
var tradeBeforeUpsertHooks []TradeHook

var tradeAfterInsertHooks []TradeHook
var tradeAfterSelectHooks []TradeHook
var tradeAfterUpdateHooks []TradeHook
var tradeAfterDeleteHooks []TradeHook
var tradeAfterUpsertHooks []TradeHook

// doBeforeInsertHooks executes all "before insert" hooks.
func (o *Trade) doBeforeInsertHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range tradeBeforeInsertHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// doBeforeUpdateHooks executes all "before Update" hooks.
func (o *Trade) doBeforeUpdateHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range tradeBeforeUpdateHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// doBeforeDeleteHooks executes all "before Delete" hooks.
func (o *Trade) doBeforeDeleteHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range tradeBeforeDeleteHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// doBeforeUpsertHooks executes all "before Upsert" hooks.
func (o *Trade) doBeforeUpsertHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range tradeBeforeUpsertHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// doAfterInsertHooks executes all "after Insert" hooks.
func (o *Trade) doAfterInsertHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range tradeAfterInsertHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// doAfterSelectHooks executes all "after Select" hooks.
func (o *Trade) doAfterSelectHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range tradeAfterSelectHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// doAfterUpdateHooks executes all "after Update" hooks.
func (o *Trade) doAfterUpdateHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range tradeAfterUpdateHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// doAfterDeleteHooks executes all "after Delete" hooks.
func (o *Trade) doAfterDeleteHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range tradeAfterDeleteHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// doAfterUpsertHooks executes all "after Upsert" hooks.
func (o *Trade) doAfterUpsertHooks(ctx context.Context, exec boil.ContextExecutor) (err error) {
	if boil.HooksAreSkipped(ctx) {
		return nil
	}

	for _, hook := range tradeAfterUpsertHooks {
		if err := hook(ctx, exec, o); err != nil {
			return err
		}
	}

	return nil
}

// AddTradeHook registers your hook function for all future operations.
func AddTradeHook(hookPoint boil.HookPoint, tradeHook TradeHook) {
	switch hookPoint {
	case boil.BeforeInsertHook:
		tradeBeforeInsertHooks = append(tradeBeforeInsertHooks, tradeHook)
	case boil.BeforeUpdateHook:
		tradeBeforeUpdateHooks = append(tradeBeforeUpdateHooks, tradeHook)
	case boil.BeforeDeleteHook:
		tradeBeforeDeleteHooks = append(tradeBeforeDeleteHooks, tradeHook)
	case boil.BeforeUpsertHook:
		tradeBeforeUpsertHooks = append(tradeBeforeUpsertHooks, tradeHook)
	case boil.AfterInsertHook:
		tradeAfterInsertHooks = append(tradeAfterInsertHooks, tradeHook)
	case boil.AfterSelectHook:
		tradeAfterSelectHooks = append(tradeAfterSelectHooks, tradeHook)
	case boil.AfterUpdateHook:
		tradeAfterUpdateHooks = append(tradeAfterUpdateHooks, tradeHook)
	case boil.AfterDeleteHook:
		tradeAfterDeleteHooks = append(tradeAfterDeleteHooks, tradeHook)
	case boil.AfterUpsertHook:
		tradeAfterUpsertHooks = append(tradeAfterUpsertHooks, tradeHook)
	}
}

// One returns a single trade record from the query.
func (q tradeQuery) One(ctx context.Context, exec boil.ContextExecutor) (*Trade, error) {
	o := &Trade{}

	queries.SetLimit(q.Query, 1)

	err := q.Bind(ctx, exec, o)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "postgres: failed to execute a one query for trade")
	}

	if err := o.doAfterSelectHooks(ctx, exec); err != nil {
		return o, err
	}

	return o, nil
}

// All returns all Trade records from the query.
func (q tradeQuery) All(ctx context.Context, exec boil.ContextExecutor) (TradeSlice, error) {
	var o []*Trade

	err := q.Bind(ctx, exec, &o)
	if err != nil {
		return nil, errors.Wrap(err, "postgres: failed to assign all query results to Trade slice")
	}

	if len(tradeAfterSelectHooks) != 0 {
		for _, obj := range o {
			if err := obj.doAfterSelectHooks(ctx, exec); err != nil {
				return o, err
			}
		}
	}

	return o, nil
}

// Count returns the count of all Trade records in the query.
func (q tradeQuery) Count(ctx context.Context, exec boil.ContextExecutor) (int64, error) {
	var count int64

	queries.SetSelect(q.Query, nil)
	queries.SetCount(q.Query)

	err := q.Query.QueryRowContext(ctx, exec).Scan(&count)
	if err != nil {
		return 0, errors.Wrap(err, "postgres: failed to count trade rows")
	}

	return count, nil
}

// Exists checks if the row exists in the table.
func (q tradeQuery) Exists(ctx context.Context, exec boil.ContextExecutor) (bool, error) {
	var count int64

	queries.SetSelect(q.Query, nil)
	queries.SetCount(q.Query)
	queries.SetLimit(q.Query, 1)

	err := q.Query.QueryRowContext(ctx, exec).Scan(&count)
	if err != nil {
		return false, errors.Wrap(err, "postgres: failed to check if trade exists")
	}

	return count > 0, nil
}

// Exchange pointed to by the foreign key.
func (o *Trade) Exchange(mods ...qm.QueryMod) scriptQuery {
	queryMods := []qm.QueryMod{
		qm.Where("\"id\" = ?", o.ExchangeID),
	}

	queryMods = append(queryMods, mods...)

	query := Scripts(queryMods...)
	queries.SetFrom(query.Query, "\"script\"")

	return query
}

// LoadExchange allows an eager lookup of values, cached into the
// loaded structs of the objects. This is for an N-1 relationship.
func (tradeL) LoadExchange(ctx context.Context, e boil.ContextExecutor, singular bool, maybeTrade interface{}, mods queries.Applicator) error {
	var slice []*Trade
	var object *Trade

	if singular {
		object = maybeTrade.(*Trade)
	} else {
		slice = *maybeTrade.(*[]*Trade)
	}

	args := make([]interface{}, 0, 1)
	if singular {
		if object.R == nil {
			object.R = &tradeR{}
		}
		if !queries.IsNil(object.ExchangeID) {
			args = append(args, object.ExchangeID)
		}

	} else {
	Outer:
		for _, obj := range slice {
			if obj.R == nil {
				obj.R = &tradeR{}
			}

			for _, a := range args {
				if queries.Equal(a, obj.ExchangeID) {
					continue Outer
				}
			}

			if !queries.IsNil(obj.ExchangeID) {
				args = append(args, obj.ExchangeID)
			}

		}
	}

	if len(args) == 0 {
		return nil
	}

	query := NewQuery(qm.From(`script`), qm.WhereIn(`script.id in ?`, args...))
	if mods != nil {
		mods.Apply(query)
	}

	results, err := query.QueryContext(ctx, e)
	if err != nil {
		return errors.Wrap(err, "failed to eager load Script")
	}

	var resultSlice []*Script
	if err = queries.Bind(results, &resultSlice); err != nil {
		return errors.Wrap(err, "failed to bind eager loaded slice Script")
	}

	if err = results.Close(); err != nil {
		return errors.Wrap(err, "failed to close results of eager load for script")
	}
	if err = results.Err(); err != nil {
		return errors.Wrap(err, "error occurred during iteration of eager loaded relations for script")
	}

	if len(tradeAfterSelectHooks) != 0 {
		for _, obj := range resultSlice {
			if err := obj.doAfterSelectHooks(ctx, e); err != nil {
				return err
			}
		}
	}

	if len(resultSlice) == 0 {
		return nil
	}

	if singular {
		foreign := resultSlice[0]
		object.R.Exchange = foreign
		if foreign.R == nil {
			foreign.R = &scriptR{}
		}
		foreign.R.ExchangeTrades = append(foreign.R.ExchangeTrades, object)
		return nil
	}

	for _, local := range slice {
		for _, foreign := range resultSlice {
			if queries.Equal(local.ExchangeID, foreign.ID) {
				local.R.Exchange = foreign
				if foreign.R == nil {
					foreign.R = &scriptR{}
				}
				foreign.R.ExchangeTrades = append(foreign.R.ExchangeTrades, local)
				break
			}
		}
	}

	return nil
}

// SetExchange of the trade to the related item.
// Sets o.R.Exchange to related.
// Adds o to related.R.ExchangeTrades.
func (o *Trade) SetExchange(ctx context.Context, exec boil.ContextExecutor, insert bool, related *Script) error {
	var err error
	if insert {
		if err = related.Insert(ctx, exec, boil.Infer()); err != nil {
			return errors.Wrap(err, "failed to insert into foreign table")
		}
	}

	updateQuery := fmt.Sprintf(
		"UPDATE \"trade\" SET %s WHERE %s",
		strmangle.SetParamNames("\"", "\"", 1, []string{"exchange_id"}),
		strmangle.WhereClause("\"", "\"", 2, tradePrimaryKeyColumns),
	)
	values := []interface{}{related.ID, o.ID}

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, updateQuery)
		fmt.Fprintln(boil.DebugWriter, values)
	}

	if _, err = exec.ExecContext(ctx, updateQuery, values...); err != nil {
		return errors.Wrap(err, "failed to update local table")
	}

	queries.Assign(&o.ExchangeID, related.ID)
	if o.R == nil {
		o.R = &tradeR{
			Exchange: related,
		}
	} else {
		o.R.Exchange = related
	}

	if related.R == nil {
		related.R = &scriptR{
			ExchangeTrades: TradeSlice{o},
		}
	} else {
		related.R.ExchangeTrades = append(related.R.ExchangeTrades, o)
	}

	return nil
}

// RemoveExchange relationship.
// Sets o.R.Exchange to nil.
// Removes o from all passed in related items' relationships struct (Optional).
func (o *Trade) RemoveExchange(ctx context.Context, exec boil.ContextExecutor, related *Script) error {
	var err error

	queries.SetScanner(&o.ExchangeID, nil)
	if _, err = o.Update(ctx, exec, boil.Whitelist("exchange_id")); err != nil {
		return errors.Wrap(err, "failed to update local table")
	}

	o.R.Exchange = nil
	if related == nil || related.R == nil {
		return nil
	}

	for i, ri := range related.R.ExchangeTrades {
		if queries.Equal(o.ExchangeID, ri.ExchangeID) {
			continue
		}

		ln := len(related.R.ExchangeTrades)
		if ln > 1 && i < ln-1 {
			related.R.ExchangeTrades[i] = related.R.ExchangeTrades[ln-1]
		}
		related.R.ExchangeTrades = related.R.ExchangeTrades[:ln-1]
		break
	}
	return nil
}

// Trades retrieves all the records using an executor.
func Trades(mods ...qm.QueryMod) tradeQuery {
	mods = append(mods, qm.From("\"trade\""))
	return tradeQuery{NewQuery(mods...)}
}

// FindTrade retrieves a single record by ID with an executor.
// If selectCols is empty Find will return all columns.
func FindTrade(ctx context.Context, exec boil.ContextExecutor, iD string, selectCols ...string) (*Trade, error) {
	tradeObj := &Trade{}

	sel := "*"
	if len(selectCols) > 0 {
		sel = strings.Join(strmangle.IdentQuoteSlice(dialect.LQ, dialect.RQ, selectCols), ",")
	}
	query := fmt.Sprintf(
		"select %s from \"trade\" where \"id\"=$1", sel,
	)

	q := queries.Raw(query, iD)

	err := q.Bind(ctx, exec, tradeObj)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "postgres: unable to select from trade")
	}

	return tradeObj, nil
}

// Insert a single record using an executor.
// See boil.Columns.InsertColumnSet documentation to understand column list inference for inserts.
func (o *Trade) Insert(ctx context.Context, exec boil.ContextExecutor, columns boil.Columns) error {
	if o == nil {
		return errors.New("postgres: no trade provided for insertion")
	}

	var err error

	if err := o.doBeforeInsertHooks(ctx, exec); err != nil {
		return err
	}

	nzDefaults := queries.NonZeroDefaultSet(tradeColumnsWithDefault, o)

	key := makeCacheKey(columns, nzDefaults)
	tradeInsertCacheMut.RLock()
	cache, cached := tradeInsertCache[key]
	tradeInsertCacheMut.RUnlock()

	if !cached {
		wl, returnColumns := columns.InsertColumnSet(
			tradeAllColumns,
			tradeColumnsWithDefault,
			tradeColumnsWithoutDefault,
			nzDefaults,
		)

		cache.valueMapping, err = queries.BindMapping(tradeType, tradeMapping, wl)
		if err != nil {
			return err
		}
		cache.retMapping, err = queries.BindMapping(tradeType, tradeMapping, returnColumns)
		if err != nil {
			return err
		}
		if len(wl) != 0 {
			cache.query = fmt.Sprintf("INSERT INTO \"trade\" (\"%s\") %%sVALUES (%s)%%s", strings.Join(wl, "\",\""), strmangle.Placeholders(dialect.UseIndexPlaceholders, len(wl), 1, 1))
		} else {
			cache.query = "INSERT INTO \"trade\" %sDEFAULT VALUES%s"
		}

		var queryOutput, queryReturning string

		if len(cache.retMapping) != 0 {
			queryReturning = fmt.Sprintf(" RETURNING \"%s\"", strings.Join(returnColumns, "\",\""))
		}

		cache.query = fmt.Sprintf(cache.query, queryOutput, queryReturning)
	}

	value := reflect.Indirect(reflect.ValueOf(o))
	vals := queries.ValuesFromMapping(value, cache.valueMapping)

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, cache.query)
		fmt.Fprintln(boil.DebugWriter, vals)
	}

	if len(cache.retMapping) != 0 {
		err = exec.QueryRowContext(ctx, cache.query, vals...).Scan(queries.PtrsFromMapping(value, cache.retMapping)...)
	} else {
		_, err = exec.ExecContext(ctx, cache.query, vals...)
	}

	if err != nil {
		return errors.Wrap(err, "postgres: unable to insert into trade")
	}

	if !cached {
		tradeInsertCacheMut.Lock()
		tradeInsertCache[key] = cache
		tradeInsertCacheMut.Unlock()
	}

	return o.doAfterInsertHooks(ctx, exec)
}

// Update uses an executor to update the Trade.
// See boil.Columns.UpdateColumnSet documentation to understand column list inference for updates.
// Update does not automatically update the record in case of default values. Use .Reload() to refresh the records.
func (o *Trade) Update(ctx context.Context, exec boil.ContextExecutor, columns boil.Columns) (int64, error) {
	var err error
	if err = o.doBeforeUpdateHooks(ctx, exec); err != nil {
		return 0, err
	}
	key := makeCacheKey(columns, nil)
	tradeUpdateCacheMut.RLock()
	cache, cached := tradeUpdateCache[key]
	tradeUpdateCacheMut.RUnlock()

	if !cached {
		wl := columns.UpdateColumnSet(
			tradeAllColumns,
			tradePrimaryKeyColumns,
		)

		if len(wl) == 0 {
			return 0, errors.New("postgres: unable to update trade, could not build whitelist")
		}

		cache.query = fmt.Sprintf("UPDATE \"trade\" SET %s WHERE %s",
			strmangle.SetParamNames("\"", "\"", 1, wl),
			strmangle.WhereClause("\"", "\"", len(wl)+1, tradePrimaryKeyColumns),
		)
		cache.valueMapping, err = queries.BindMapping(tradeType, tradeMapping, append(wl, tradePrimaryKeyColumns...))
		if err != nil {
			return 0, err
		}
	}

	values := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(o)), cache.valueMapping)

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, cache.query)
		fmt.Fprintln(boil.DebugWriter, values)
	}

	var result sql.Result
	result, err = exec.ExecContext(ctx, cache.query, values...)
	if err != nil {
		return 0, errors.Wrap(err, "postgres: unable to update trade row")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "postgres: failed to get rows affected by update for trade")
	}

	if !cached {
		tradeUpdateCacheMut.Lock()
		tradeUpdateCache[key] = cache
		tradeUpdateCacheMut.Unlock()
	}

	return rowsAff, o.doAfterUpdateHooks(ctx, exec)
}

// UpdateAll updates all rows with the specified column values.
func (q tradeQuery) UpdateAll(ctx context.Context, exec boil.ContextExecutor, cols M) (int64, error) {
	queries.SetUpdate(q.Query, cols)

	result, err := q.Query.ExecContext(ctx, exec)
	if err != nil {
		return 0, errors.Wrap(err, "postgres: unable to update all for trade")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "postgres: unable to retrieve rows affected for trade")
	}

	return rowsAff, nil
}

// UpdateAll updates all rows with the specified column values, using an executor.
func (o TradeSlice) UpdateAll(ctx context.Context, exec boil.ContextExecutor, cols M) (int64, error) {
	ln := int64(len(o))
	if ln == 0 {
		return 0, nil
	}

	if len(cols) == 0 {
		return 0, errors.New("postgres: update all requires at least one column argument")
	}

	colNames := make([]string, len(cols))
	args := make([]interface{}, len(cols))

	i := 0
	for name, value := range cols {
		colNames[i] = name
		args[i] = value
		i++
	}

	// Append all of the primary key values for each column
	for _, obj := range o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), tradePrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := fmt.Sprintf("UPDATE \"trade\" SET %s WHERE %s",
		strmangle.SetParamNames("\"", "\"", 1, colNames),
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), len(colNames)+1, tradePrimaryKeyColumns, len(o)))

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, args...)
	}

	result, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, errors.Wrap(err, "postgres: unable to update all in trade slice")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "postgres: unable to retrieve rows affected all in update all trade")
	}
	return rowsAff, nil
}

// Upsert attempts an insert using an executor, and does an update or ignore on conflict.
// See boil.Columns documentation for how to properly use updateColumns and insertColumns.
func (o *Trade) Upsert(ctx context.Context, exec boil.ContextExecutor, updateOnConflict bool, conflictColumns []string, updateColumns, insertColumns boil.Columns) error {
	if o == nil {
		return errors.New("postgres: no trade provided for upsert")
	}

	if err := o.doBeforeUpsertHooks(ctx, exec); err != nil {
		return err
	}

	nzDefaults := queries.NonZeroDefaultSet(tradeColumnsWithDefault, o)

	// Build cache key in-line uglily - mysql vs psql problems
	buf := strmangle.GetBuffer()
	if updateOnConflict {
		buf.WriteByte('t')
	} else {
		buf.WriteByte('f')
	}
	buf.WriteByte('.')
	for _, c := range conflictColumns {
		buf.WriteString(c)
	}
	buf.WriteByte('.')
	buf.WriteString(strconv.Itoa(updateColumns.Kind))
	for _, c := range updateColumns.Cols {
		buf.WriteString(c)
	}
	buf.WriteByte('.')
	buf.WriteString(strconv.Itoa(insertColumns.Kind))
	for _, c := range insertColumns.Cols {
		buf.WriteString(c)
	}
	buf.WriteByte('.')
	for _, c := range nzDefaults {
		buf.WriteString(c)
	}
	key := buf.String()
	strmangle.PutBuffer(buf)

	tradeUpsertCacheMut.RLock()
	cache, cached := tradeUpsertCache[key]
	tradeUpsertCacheMut.RUnlock()

	var err error

	if !cached {
		insert, ret := insertColumns.InsertColumnSet(
			tradeAllColumns,
			tradeColumnsWithDefault,
			tradeColumnsWithoutDefault,
			nzDefaults,
		)
		update := updateColumns.UpdateColumnSet(
			tradeAllColumns,
			tradePrimaryKeyColumns,
		)

		if updateOnConflict && len(update) == 0 {
			return errors.New("postgres: unable to upsert trade, could not build update column list")
		}

		conflict := conflictColumns
		if len(conflict) == 0 {
			conflict = make([]string, len(tradePrimaryKeyColumns))
			copy(conflict, tradePrimaryKeyColumns)
		}
		cache.query = buildUpsertQueryPostgres(dialect, "\"trade\"", updateOnConflict, ret, update, conflict, insert)

		cache.valueMapping, err = queries.BindMapping(tradeType, tradeMapping, insert)
		if err != nil {
			return err
		}
		if len(ret) != 0 {
			cache.retMapping, err = queries.BindMapping(tradeType, tradeMapping, ret)
			if err != nil {
				return err
			}
		}
	}

	value := reflect.Indirect(reflect.ValueOf(o))
	vals := queries.ValuesFromMapping(value, cache.valueMapping)
	var returns []interface{}
	if len(cache.retMapping) != 0 {
		returns = queries.PtrsFromMapping(value, cache.retMapping)
	}

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, cache.query)
		fmt.Fprintln(boil.DebugWriter, vals)
	}

	if len(cache.retMapping) != 0 {
		err = exec.QueryRowContext(ctx, cache.query, vals...).Scan(returns...)
		if err == sql.ErrNoRows {
			err = nil // Postgres doesn't return anything when there's no update
		}
	} else {
		_, err = exec.ExecContext(ctx, cache.query, vals...)
	}
	if err != nil {
		return errors.Wrap(err, "postgres: unable to upsert trade")
	}

	if !cached {
		tradeUpsertCacheMut.Lock()
		tradeUpsertCache[key] = cache
		tradeUpsertCacheMut.Unlock()
	}

	return o.doAfterUpsertHooks(ctx, exec)
}

// Delete deletes a single Trade record with an executor.
// Delete will match against the primary key column to find the record to delete.
func (o *Trade) Delete(ctx context.Context, exec boil.ContextExecutor) (int64, error) {
	if o == nil {
		return 0, errors.New("postgres: no Trade provided for delete")
	}

	if err := o.doBeforeDeleteHooks(ctx, exec); err != nil {
		return 0, err
	}

	args := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(o)), tradePrimaryKeyMapping)
	sql := "DELETE FROM \"trade\" WHERE \"id\"=$1"

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, args...)
	}

	result, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, errors.Wrap(err, "postgres: unable to delete from trade")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "postgres: failed to get rows affected by delete for trade")
	}

	if err := o.doAfterDeleteHooks(ctx, exec); err != nil {
		return 0, err
	}

	return rowsAff, nil
}

// DeleteAll deletes all matching rows.
func (q tradeQuery) DeleteAll(ctx context.Context, exec boil.ContextExecutor) (int64, error) {
	if q.Query == nil {
		return 0, errors.New("postgres: no tradeQuery provided for delete all")
	}

	queries.SetDelete(q.Query)

	result, err := q.Query.ExecContext(ctx, exec)
	if err != nil {
		return 0, errors.Wrap(err, "postgres: unable to delete all from trade")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "postgres: failed to get rows affected by deleteall for trade")
	}

	return rowsAff, nil
}

// DeleteAll deletes all rows in the slice, using an executor.
func (o TradeSlice) DeleteAll(ctx context.Context, exec boil.ContextExecutor) (int64, error) {
	if len(o) == 0 {
		return 0, nil
	}

	if len(tradeBeforeDeleteHooks) != 0 {
		for _, obj := range o {
			if err := obj.doBeforeDeleteHooks(ctx, exec); err != nil {
				return 0, err
			}
		}
	}

	var args []interface{}
	for _, obj := range o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), tradePrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := "DELETE FROM \"trade\" WHERE " +
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), 1, tradePrimaryKeyColumns, len(o))

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, args)
	}

	result, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, errors.Wrap(err, "postgres: unable to delete all from trade slice")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "postgres: failed to get rows affected by deleteall for trade")
	}

	if len(tradeAfterDeleteHooks) != 0 {
		for _, obj := range o {
			if err := obj.doAfterDeleteHooks(ctx, exec); err != nil {
				return 0, err
			}
		}
	}

	return rowsAff, nil
}

// Reload refetches the object from the database
// using the primary keys with an executor.
func (o *Trade) Reload(ctx context.Context, exec boil.ContextExecutor) error {
	ret, err := FindTrade(ctx, exec, o.ID)
	if err != nil {
		return err
	}

	*o = *ret
	return nil
}

// ReloadAll refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
func (o *TradeSlice) ReloadAll(ctx context.Context, exec boil.ContextExecutor) error {
	if o == nil || len(*o) == 0 {
		return nil
	}

	slice := TradeSlice{}
	var args []interface{}
	for _, obj := range *o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), tradePrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := "SELECT \"trade\".* FROM \"trade\" WHERE " +
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), 1, tradePrimaryKeyColumns, len(*o))

	q := queries.Raw(sql, args...)

	err := q.Bind(ctx, exec, &slice)
	if err != nil {
		return errors.Wrap(err, "postgres: unable to reload all in TradeSlice")
	}

	*o = slice

	return nil
}

// TradeExists checks if the Trade row exists.
func TradeExists(ctx context.Context, exec boil.ContextExecutor, iD string) (bool, error) {
	var exists bool
	sql := "select exists(select 1 from \"trade\" where \"id\"=$1 limit 1)"

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, iD)
	}

	row := exec.QueryRowContext(ctx, sql, iD)

	err := row.Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, "postgres: unable to check if trade exists")
	}

	return exists, nil
}
