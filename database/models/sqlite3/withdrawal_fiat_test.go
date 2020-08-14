// Code generated by SQLBoiler 3.5.1-gct (https://github.com/thrasher-corp/sqlboiler). DO NOT EDIT.
// This file is meant to be re-generated in place and/or deleted at any time.

package sqlite3

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/thrasher-corp/sqlboiler/boil"
	"github.com/thrasher-corp/sqlboiler/queries"
	"github.com/thrasher-corp/sqlboiler/randomize"
	"github.com/thrasher-corp/sqlboiler/strmangle"
)

var (
	// Relationships sometimes use the reflection helper queries.Equal/queries.Assign
	// so force a package dependency in case they don't.
	_ = queries.Equal
)

func testWithdrawalFiats(t *testing.T) {
	t.Parallel()

	query := WithdrawalFiats()

	if query.Query == nil {
		t.Error("expected a query, got nothing")
	}
}

func testWithdrawalFiatsDelete(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &WithdrawalFiat{}
	if err = randomize.Struct(seed, o, withdrawalFiatDBTypes, true, withdrawalFiatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize WithdrawalFiat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	if rowsAff, err := o.Delete(ctx, tx); err != nil {
		t.Error(err)
	} else if rowsAff != 1 {
		t.Error("should only have deleted one row, but affected:", rowsAff)
	}

	count, err := WithdrawalFiats().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 0 {
		t.Error("want zero records, got:", count)
	}
}

func testWithdrawalFiatsQueryDeleteAll(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &WithdrawalFiat{}
	if err = randomize.Struct(seed, o, withdrawalFiatDBTypes, true, withdrawalFiatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize WithdrawalFiat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	if rowsAff, err := WithdrawalFiats().DeleteAll(ctx, tx); err != nil {
		t.Error(err)
	} else if rowsAff != 1 {
		t.Error("should only have deleted one row, but affected:", rowsAff)
	}

	count, err := WithdrawalFiats().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 0 {
		t.Error("want zero records, got:", count)
	}
}

func testWithdrawalFiatsSliceDeleteAll(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &WithdrawalFiat{}
	if err = randomize.Struct(seed, o, withdrawalFiatDBTypes, true, withdrawalFiatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize WithdrawalFiat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	slice := WithdrawalFiatSlice{o}

	if rowsAff, err := slice.DeleteAll(ctx, tx); err != nil {
		t.Error(err)
	} else if rowsAff != 1 {
		t.Error("should only have deleted one row, but affected:", rowsAff)
	}

	count, err := WithdrawalFiats().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 0 {
		t.Error("want zero records, got:", count)
	}
}

func testWithdrawalFiatsExists(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &WithdrawalFiat{}
	if err = randomize.Struct(seed, o, withdrawalFiatDBTypes, true, withdrawalFiatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize WithdrawalFiat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	e, err := WithdrawalFiatExists(ctx, tx, o.ID)
	if err != nil {
		t.Errorf("Unable to check if WithdrawalFiat exists: %s", err)
	}
	if !e {
		t.Errorf("Expected WithdrawalFiatExists to return true, but got false.")
	}
}

func testWithdrawalFiatsFind(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &WithdrawalFiat{}
	if err = randomize.Struct(seed, o, withdrawalFiatDBTypes, true, withdrawalFiatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize WithdrawalFiat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	withdrawalFiatFound, err := FindWithdrawalFiat(ctx, tx, o.ID)
	if err != nil {
		t.Error(err)
	}

	if withdrawalFiatFound == nil {
		t.Error("want a record, got nil")
	}
}

func testWithdrawalFiatsBind(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &WithdrawalFiat{}
	if err = randomize.Struct(seed, o, withdrawalFiatDBTypes, true, withdrawalFiatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize WithdrawalFiat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	if err = WithdrawalFiats().Bind(ctx, tx, o); err != nil {
		t.Error(err)
	}
}

func testWithdrawalFiatsOne(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &WithdrawalFiat{}
	if err = randomize.Struct(seed, o, withdrawalFiatDBTypes, true, withdrawalFiatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize WithdrawalFiat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	if x, err := WithdrawalFiats().One(ctx, tx); err != nil {
		t.Error(err)
	} else if x == nil {
		t.Error("expected to get a non nil record")
	}
}

func testWithdrawalFiatsAll(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	withdrawalFiatOne := &WithdrawalFiat{}
	withdrawalFiatTwo := &WithdrawalFiat{}
	if err = randomize.Struct(seed, withdrawalFiatOne, withdrawalFiatDBTypes, false, withdrawalFiatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize WithdrawalFiat struct: %s", err)
	}
	if err = randomize.Struct(seed, withdrawalFiatTwo, withdrawalFiatDBTypes, false, withdrawalFiatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize WithdrawalFiat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = withdrawalFiatOne.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}
	if err = withdrawalFiatTwo.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	slice, err := WithdrawalFiats().All(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if len(slice) != 2 {
		t.Error("want 2 records, got:", len(slice))
	}
}

func testWithdrawalFiatsCount(t *testing.T) {
	t.Parallel()

	var err error
	seed := randomize.NewSeed()
	withdrawalFiatOne := &WithdrawalFiat{}
	withdrawalFiatTwo := &WithdrawalFiat{}
	if err = randomize.Struct(seed, withdrawalFiatOne, withdrawalFiatDBTypes, false, withdrawalFiatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize WithdrawalFiat struct: %s", err)
	}
	if err = randomize.Struct(seed, withdrawalFiatTwo, withdrawalFiatDBTypes, false, withdrawalFiatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize WithdrawalFiat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = withdrawalFiatOne.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}
	if err = withdrawalFiatTwo.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	count, err := WithdrawalFiats().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 2 {
		t.Error("want 2 records, got:", count)
	}
}

func withdrawalFiatBeforeInsertHook(ctx context.Context, e boil.ContextExecutor, o *WithdrawalFiat) error {
	*o = WithdrawalFiat{}
	return nil
}

func withdrawalFiatAfterInsertHook(ctx context.Context, e boil.ContextExecutor, o *WithdrawalFiat) error {
	*o = WithdrawalFiat{}
	return nil
}

func withdrawalFiatAfterSelectHook(ctx context.Context, e boil.ContextExecutor, o *WithdrawalFiat) error {
	*o = WithdrawalFiat{}
	return nil
}

func withdrawalFiatBeforeUpdateHook(ctx context.Context, e boil.ContextExecutor, o *WithdrawalFiat) error {
	*o = WithdrawalFiat{}
	return nil
}

func withdrawalFiatAfterUpdateHook(ctx context.Context, e boil.ContextExecutor, o *WithdrawalFiat) error {
	*o = WithdrawalFiat{}
	return nil
}

func withdrawalFiatBeforeDeleteHook(ctx context.Context, e boil.ContextExecutor, o *WithdrawalFiat) error {
	*o = WithdrawalFiat{}
	return nil
}

func withdrawalFiatAfterDeleteHook(ctx context.Context, e boil.ContextExecutor, o *WithdrawalFiat) error {
	*o = WithdrawalFiat{}
	return nil
}

func withdrawalFiatBeforeUpsertHook(ctx context.Context, e boil.ContextExecutor, o *WithdrawalFiat) error {
	*o = WithdrawalFiat{}
	return nil
}

func withdrawalFiatAfterUpsertHook(ctx context.Context, e boil.ContextExecutor, o *WithdrawalFiat) error {
	*o = WithdrawalFiat{}
	return nil
}

func testWithdrawalFiatsHooks(t *testing.T) {
	t.Parallel()

	var err error

	ctx := context.Background()
	empty := &WithdrawalFiat{}
	o := &WithdrawalFiat{}

	seed := randomize.NewSeed()
	if err = randomize.Struct(seed, o, withdrawalFiatDBTypes, false); err != nil {
		t.Errorf("Unable to randomize WithdrawalFiat object: %s", err)
	}

	AddWithdrawalFiatHook(boil.BeforeInsertHook, withdrawalFiatBeforeInsertHook)
	if err = o.doBeforeInsertHooks(ctx, nil); err != nil {
		t.Errorf("Unable to execute doBeforeInsertHooks: %s", err)
	}
	if !reflect.DeepEqual(o, empty) {
		t.Errorf("Expected BeforeInsertHook function to empty object, but got: %#v", o)
	}
	withdrawalFiatBeforeInsertHooks = []WithdrawalFiatHook{}

	AddWithdrawalFiatHook(boil.AfterInsertHook, withdrawalFiatAfterInsertHook)
	if err = o.doAfterInsertHooks(ctx, nil); err != nil {
		t.Errorf("Unable to execute doAfterInsertHooks: %s", err)
	}
	if !reflect.DeepEqual(o, empty) {
		t.Errorf("Expected AfterInsertHook function to empty object, but got: %#v", o)
	}
	withdrawalFiatAfterInsertHooks = []WithdrawalFiatHook{}

	AddWithdrawalFiatHook(boil.AfterSelectHook, withdrawalFiatAfterSelectHook)
	if err = o.doAfterSelectHooks(ctx, nil); err != nil {
		t.Errorf("Unable to execute doAfterSelectHooks: %s", err)
	}
	if !reflect.DeepEqual(o, empty) {
		t.Errorf("Expected AfterSelectHook function to empty object, but got: %#v", o)
	}
	withdrawalFiatAfterSelectHooks = []WithdrawalFiatHook{}

	AddWithdrawalFiatHook(boil.BeforeUpdateHook, withdrawalFiatBeforeUpdateHook)
	if err = o.doBeforeUpdateHooks(ctx, nil); err != nil {
		t.Errorf("Unable to execute doBeforeUpdateHooks: %s", err)
	}
	if !reflect.DeepEqual(o, empty) {
		t.Errorf("Expected BeforeUpdateHook function to empty object, but got: %#v", o)
	}
	withdrawalFiatBeforeUpdateHooks = []WithdrawalFiatHook{}

	AddWithdrawalFiatHook(boil.AfterUpdateHook, withdrawalFiatAfterUpdateHook)
	if err = o.doAfterUpdateHooks(ctx, nil); err != nil {
		t.Errorf("Unable to execute doAfterUpdateHooks: %s", err)
	}
	if !reflect.DeepEqual(o, empty) {
		t.Errorf("Expected AfterUpdateHook function to empty object, but got: %#v", o)
	}
	withdrawalFiatAfterUpdateHooks = []WithdrawalFiatHook{}

	AddWithdrawalFiatHook(boil.BeforeDeleteHook, withdrawalFiatBeforeDeleteHook)
	if err = o.doBeforeDeleteHooks(ctx, nil); err != nil {
		t.Errorf("Unable to execute doBeforeDeleteHooks: %s", err)
	}
	if !reflect.DeepEqual(o, empty) {
		t.Errorf("Expected BeforeDeleteHook function to empty object, but got: %#v", o)
	}
	withdrawalFiatBeforeDeleteHooks = []WithdrawalFiatHook{}

	AddWithdrawalFiatHook(boil.AfterDeleteHook, withdrawalFiatAfterDeleteHook)
	if err = o.doAfterDeleteHooks(ctx, nil); err != nil {
		t.Errorf("Unable to execute doAfterDeleteHooks: %s", err)
	}
	if !reflect.DeepEqual(o, empty) {
		t.Errorf("Expected AfterDeleteHook function to empty object, but got: %#v", o)
	}
	withdrawalFiatAfterDeleteHooks = []WithdrawalFiatHook{}

	AddWithdrawalFiatHook(boil.BeforeUpsertHook, withdrawalFiatBeforeUpsertHook)
	if err = o.doBeforeUpsertHooks(ctx, nil); err != nil {
		t.Errorf("Unable to execute doBeforeUpsertHooks: %s", err)
	}
	if !reflect.DeepEqual(o, empty) {
		t.Errorf("Expected BeforeUpsertHook function to empty object, but got: %#v", o)
	}
	withdrawalFiatBeforeUpsertHooks = []WithdrawalFiatHook{}

	AddWithdrawalFiatHook(boil.AfterUpsertHook, withdrawalFiatAfterUpsertHook)
	if err = o.doAfterUpsertHooks(ctx, nil); err != nil {
		t.Errorf("Unable to execute doAfterUpsertHooks: %s", err)
	}
	if !reflect.DeepEqual(o, empty) {
		t.Errorf("Expected AfterUpsertHook function to empty object, but got: %#v", o)
	}
	withdrawalFiatAfterUpsertHooks = []WithdrawalFiatHook{}
}

func testWithdrawalFiatsInsert(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &WithdrawalFiat{}
	if err = randomize.Struct(seed, o, withdrawalFiatDBTypes, true, withdrawalFiatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize WithdrawalFiat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	count, err := WithdrawalFiats().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 1 {
		t.Error("want one record, got:", count)
	}
}

func testWithdrawalFiatsInsertWhitelist(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &WithdrawalFiat{}
	if err = randomize.Struct(seed, o, withdrawalFiatDBTypes, true); err != nil {
		t.Errorf("Unable to randomize WithdrawalFiat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Whitelist(withdrawalFiatColumnsWithoutDefault...)); err != nil {
		t.Error(err)
	}

	count, err := WithdrawalFiats().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 1 {
		t.Error("want one record, got:", count)
	}
}

func testWithdrawalFiatToOneWithdrawalHistoryUsingWithdrawalHistory(t *testing.T) {
	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()

	var local WithdrawalFiat
	var foreign WithdrawalHistory

	seed := randomize.NewSeed()
	if err := randomize.Struct(seed, &local, withdrawalFiatDBTypes, false, withdrawalFiatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize WithdrawalFiat struct: %s", err)
	}
	if err := randomize.Struct(seed, &foreign, withdrawalHistoryDBTypes, false, withdrawalHistoryColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize WithdrawalHistory struct: %s", err)
	}

	if err := foreign.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Fatal(err)
	}

	local.WithdrawalHistoryID = foreign.ID
	if err := local.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Fatal(err)
	}

	check, err := local.WithdrawalHistory().One(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}

	if check.ID != foreign.ID {
		t.Errorf("want: %v, got %v", foreign.ID, check.ID)
	}

	slice := WithdrawalFiatSlice{&local}
	if err = local.L.LoadWithdrawalHistory(ctx, tx, false, (*[]*WithdrawalFiat)(&slice), nil); err != nil {
		t.Fatal(err)
	}
	if local.R.WithdrawalHistory == nil {
		t.Error("struct should have been eager loaded")
	}

	local.R.WithdrawalHistory = nil
	if err = local.L.LoadWithdrawalHistory(ctx, tx, true, &local, nil); err != nil {
		t.Fatal(err)
	}
	if local.R.WithdrawalHistory == nil {
		t.Error("struct should have been eager loaded")
	}
}

func testWithdrawalFiatToOneSetOpWithdrawalHistoryUsingWithdrawalHistory(t *testing.T) {
	var err error

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()

	var a WithdrawalFiat
	var b, c WithdrawalHistory

	seed := randomize.NewSeed()
	if err = randomize.Struct(seed, &a, withdrawalFiatDBTypes, false, strmangle.SetComplement(withdrawalFiatPrimaryKeyColumns, withdrawalFiatColumnsWithoutDefault)...); err != nil {
		t.Fatal(err)
	}
	if err = randomize.Struct(seed, &b, withdrawalHistoryDBTypes, false, strmangle.SetComplement(withdrawalHistoryPrimaryKeyColumns, withdrawalHistoryColumnsWithoutDefault)...); err != nil {
		t.Fatal(err)
	}
	if err = randomize.Struct(seed, &c, withdrawalHistoryDBTypes, false, strmangle.SetComplement(withdrawalHistoryPrimaryKeyColumns, withdrawalHistoryColumnsWithoutDefault)...); err != nil {
		t.Fatal(err)
	}

	if err := a.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Fatal(err)
	}
	if err = b.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Fatal(err)
	}

	for i, x := range []*WithdrawalHistory{&b, &c} {
		err = a.SetWithdrawalHistory(ctx, tx, i != 0, x)
		if err != nil {
			t.Fatal(err)
		}

		if a.R.WithdrawalHistory != x {
			t.Error("relationship struct not set to correct value")
		}

		if x.R.WithdrawalFiats[0] != &a {
			t.Error("failed to append to foreign relationship struct")
		}
		if a.WithdrawalHistoryID != x.ID {
			t.Error("foreign key was wrong value", a.WithdrawalHistoryID)
		}

		zero := reflect.Zero(reflect.TypeOf(a.WithdrawalHistoryID))
		reflect.Indirect(reflect.ValueOf(&a.WithdrawalHistoryID)).Set(zero)

		if err = a.Reload(ctx, tx); err != nil {
			t.Fatal("failed to reload", err)
		}

		if a.WithdrawalHistoryID != x.ID {
			t.Error("foreign key was wrong value", a.WithdrawalHistoryID, x.ID)
		}
	}
}

func testWithdrawalFiatsReload(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &WithdrawalFiat{}
	if err = randomize.Struct(seed, o, withdrawalFiatDBTypes, true, withdrawalFiatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize WithdrawalFiat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	if err = o.Reload(ctx, tx); err != nil {
		t.Error(err)
	}
}

func testWithdrawalFiatsReloadAll(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &WithdrawalFiat{}
	if err = randomize.Struct(seed, o, withdrawalFiatDBTypes, true, withdrawalFiatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize WithdrawalFiat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	slice := WithdrawalFiatSlice{o}

	if err = slice.ReloadAll(ctx, tx); err != nil {
		t.Error(err)
	}
}

func testWithdrawalFiatsSelect(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &WithdrawalFiat{}
	if err = randomize.Struct(seed, o, withdrawalFiatDBTypes, true, withdrawalFiatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize WithdrawalFiat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	slice, err := WithdrawalFiats().All(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if len(slice) != 1 {
		t.Error("want one record, got:", len(slice))
	}
}

var (
	withdrawalFiatDBTypes = map[string]string{`ID`: `INTEGER`, `BankName`: `TEXT`, `BankAddress`: `TEXT`, `BankAccountName`: `TEXT`, `BankAccountNumber`: `TEXT`, `BSB`: `TEXT`, `SwiftCode`: `TEXT`, `Iban`: `TEXT`, `BankCode`: `REAL`, `WithdrawalHistoryID`: `TEXT`}
	_                     = bytes.MinRead
)

func testWithdrawalFiatsUpdate(t *testing.T) {
	t.Parallel()

	if 0 == len(withdrawalFiatPrimaryKeyColumns) {
		t.Skip("Skipping table with no primary key columns")
	}
	if len(withdrawalFiatAllColumns) == len(withdrawalFiatPrimaryKeyColumns) {
		t.Skip("Skipping table with only primary key columns")
	}

	seed := randomize.NewSeed()
	var err error
	o := &WithdrawalFiat{}
	if err = randomize.Struct(seed, o, withdrawalFiatDBTypes, true, withdrawalFiatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize WithdrawalFiat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	count, err := WithdrawalFiats().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 1 {
		t.Error("want one record, got:", count)
	}

	if err = randomize.Struct(seed, o, withdrawalFiatDBTypes, true, withdrawalFiatPrimaryKeyColumns...); err != nil {
		t.Errorf("Unable to randomize WithdrawalFiat struct: %s", err)
	}

	if rowsAff, err := o.Update(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	} else if rowsAff != 1 {
		t.Error("should only affect one row but affected", rowsAff)
	}
}

func testWithdrawalFiatsSliceUpdateAll(t *testing.T) {
	t.Parallel()

	if len(withdrawalFiatAllColumns) == len(withdrawalFiatPrimaryKeyColumns) {
		t.Skip("Skipping table with only primary key columns")
	}

	seed := randomize.NewSeed()
	var err error
	o := &WithdrawalFiat{}
	if err = randomize.Struct(seed, o, withdrawalFiatDBTypes, true, withdrawalFiatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize WithdrawalFiat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	count, err := WithdrawalFiats().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 1 {
		t.Error("want one record, got:", count)
	}

	if err = randomize.Struct(seed, o, withdrawalFiatDBTypes, true, withdrawalFiatPrimaryKeyColumns...); err != nil {
		t.Errorf("Unable to randomize WithdrawalFiat struct: %s", err)
	}

	// Remove Primary keys and unique columns from what we plan to update
	var fields []string
	if strmangle.StringSliceMatch(withdrawalFiatAllColumns, withdrawalFiatPrimaryKeyColumns) {
		fields = withdrawalFiatAllColumns
	} else {
		fields = strmangle.SetComplement(
			withdrawalFiatAllColumns,
			withdrawalFiatPrimaryKeyColumns,
		)
	}

	value := reflect.Indirect(reflect.ValueOf(o))
	typ := reflect.TypeOf(o).Elem()
	n := typ.NumField()

	updateMap := M{}
	for _, col := range fields {
		for i := 0; i < n; i++ {
			f := typ.Field(i)
			if f.Tag.Get("boil") == col {
				updateMap[col] = value.Field(i).Interface()
			}
		}
	}

	slice := WithdrawalFiatSlice{o}
	if rowsAff, err := slice.UpdateAll(ctx, tx, updateMap); err != nil {
		t.Error(err)
	} else if rowsAff != 1 {
		t.Error("wanted one record updated but got", rowsAff)
	}
}
