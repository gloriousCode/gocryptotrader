package audit

import (
	"context"
	"time"

	"github.com/thrasher-corp/sqlboiler/boil"
	"github.com/thrasher-corp/sqlboiler/queries/qm"

	"github.com/thrasher-corp/gocryptotrader/database"
	modelPSQL "github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	modelSQLite "github.com/thrasher-corp/gocryptotrader/database/models/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Event inserts a new audit event to database
func Event(id, msgtype, message string) {
	if !database.CheckConnection() {
		return
	}

	ctx := context.Background()
	ctx = boil.SkipTimestamps(ctx)

	tx, err := database.BeginTransaction()
	if err != nil {
		log.Errorf(log.Global, "Event transaction begin failed: %v", err)
		return
	}

	if database.GetSQLDialect() == database.DBSQLite3 {
		var tempEvent = modelSQLite.AuditEvent{
			Type:       msgtype,
			Identifier: id,
			Message:    message,
		}
		err = tempEvent.Insert(ctx, tx, boil.Blacklist("created_at"))
	} else {
		var tempEvent = modelPSQL.AuditEvent{
			Type:       msgtype,
			Identifier: id,
			Message:    message,
		}
		err = tempEvent.Insert(ctx, tx, boil.Blacklist("created_at"))
	}

	if err != nil {
		log.Errorf(log.Global, "Event insert failed: %v", err)
		err = tx.Rollback()
		if err != nil {
			log.Errorf(log.Global, "Event Transaction rollback failed: %v", err)
		}
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Errorf(log.Global, "Event Transaction commit failed: %v", err)
		return
	}
}

// GetEvent () returns list of order events matching query
func GetEvent(startTime, endTime time.Time, order string, limit int) (interface{}, error) {
	if !database.CheckConnection() {
		return nil, database.ErrDatabaseSupportDisabled
	}

	query := qm.Where("created_at BETWEEN ? AND ?", startTime, endTime)

	orderByQueryString := "id"
	if order == "desc" {
		orderByQueryString += " desc"
	}

	orderByQuery := qm.OrderBy(orderByQueryString)
	limitQuery := qm.Limit(limit)

	ctx := context.Background()
	if database.GetSQLDialect() == database.DBSQLite3 {
		return modelSQLite.AuditEvents(query, orderByQuery, limitQuery).All(ctx, database.Executor())
	}

	return modelPSQL.AuditEvents(query, orderByQuery, limitQuery).All(ctx, database.Executor())
}
