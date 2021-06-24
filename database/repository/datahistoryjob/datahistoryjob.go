package datahistoryjob

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/database"
	modelPSQL "github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	"github.com/thrasher-corp/gocryptotrader/database/models/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	"github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/sqlboiler/boil"
)

// Setup returns a usable DataHistoryDB service
// so you don't need to interact with globals in any fashion
func Setup(db iDatabase) (*DataHistoryDB, error) {
	if db == nil {
		return nil, nil
	}
	if !db.IsConnected() {
		return nil, nil
	}
	return &DataHistoryDB{sql: db.GetSQL()}, nil
}

func (db *DataHistoryDB) Upsert(jobs ...DataHistoryJob) error {
	ctx := context.Background()

	tx, err := db.sql.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginTx %w", err)
	}
	defer func() {
		if err != nil {
			errRB := tx.Rollback()
			if errRB != nil {
				log.Errorf(log.DatabaseMgr, "Insert tx.Rollback %v", errRB)
			}
		}
	}()
	if repository.GetSQLDialect() == database.DBSQLite3 || repository.GetSQLDialect() == database.DBSQLite {
		err = upsertSqlite(ctx, tx, jobs...)
	} else {
		err = insertPostgres(ctx, tx, jobs...)
	}
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DataHistoryDB) GetByNickName(nickname string) (*DataHistoryJob, error) {
	return nil, nil
}

func upsertSqlite(ctx context.Context, tx *sql.Tx, jobs ...DataHistoryJob) error {
	for i := range jobs {
		if jobs[i].ID == "" {
			freshUUID, err := uuid.NewV4()
			if err != nil {
				return err
			}
			jobs[i].ID = freshUUID.String()
		}
		exchangeUUID, err := exchange.UUIDByName(jobs[i].Exchange)
		if err != nil {
			return err
		}
		var tempEvent = sqlite3.Datahistoryjob{
			ID:             jobs[i].ID,
			Nickname:       jobs[i].NickName,
			ExchangeNameID: exchangeUUID.String(),
			Asset:          strings.ToLower(jobs[i].Asset),
			Base:           strings.ToUpper(jobs[i].Base),
			Quote:          strings.ToUpper(jobs[i].Quote),
			StartTime:      jobs[i].StartDate.UTC().Format(time.RFC3339),
			EndTime:        jobs[i].EndDate.UTC().Format(time.RFC3339),
			Interval:       float64(jobs[i].Interval),
			DataType:       jobs[i].DataType,
			RequestSize:    float64(jobs[i].RequestSizeLimit),
			MaxRetries:     float64(jobs[i].MaxRetryAttempts),
			Status:         float64(jobs[i].Status),
		}
		err = tempEvent.Insert(ctx, tx, boil.Infer())
		if err != nil {
			return err
		}
	}

	return nil
}

func insertPostgres(ctx context.Context, tx *sql.Tx, jobs ...DataHistoryJob) error {
	var err error
	for i := range jobs {
		if jobs[i].ID == "" {
			var freshUUID uuid.UUID
			freshUUID, err = uuid.NewV4()
			if err != nil {
				return err
			}
			jobs[i].ID = freshUUID.String()
		}
		var tempEvent = modelPSQL.DataHistoryJob{
			ExchangeNameID: jobs[i].ExchangeNameID,
			Base:           strings.ToUpper(jobs[i].Base),
			Quote:          strings.ToUpper(jobs[i].Quote),
			Asset:          strings.ToLower(jobs[i].AssetType),
			Price:          jobs[i].Price,
			Amount:         jobs[i].Amount,
			Timestamp:      jobs[i].Timestamp.UTC(),
			ID:             jobs[i].ID,
		}
		if jobs[i].Side != "" {
			tempEvent.Side.SetValid(strings.ToUpper(jobs[i].Side))
		}
		if jobs[i].TID != "" {
			tempEvent.Tid.SetValid(jobs[i].TID)
		}

		err = tempEvent.Upsert(ctx, tx, false, nil, boil.Infer(), boil.Infer())
		if err != nil {
			return err
		}
	}

	return nil
}
