package errors

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	ErrorBeginTransaction      = "error beginning transaction"
	ErrorGettingAffectedRows   = "error getting rows affected"
	ErrorGettingLastID         = "error getting last affected ID"
	ErrorGettingSegmentID      = "error getting segment id"
	ErrorCommittingTransaction = "error committing transaction"
)

func DBConnectLoop(dsn string, timeout time.Duration) (*sql.DB, error) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeoutExceeded := time.After(timeout)
	for {
		select {
		case <-timeoutExceeded:
			return nil, fmt.Errorf("db connection failed after %s timeout", timeout)

		case <-ticker.C:
			db, err := sql.Open("mysql", dsn)
			if err == nil {
				return db, nil
			}
			return nil, fmt.Errorf("%s: failed to connect to db %w", dsn, err)
		}
	}
}

func ValidateAndParseJSON(r *http.Request, parseInto interface{}) error {
	var body []byte
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}

	err = r.Body.Close()
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, parseInto)
	if err != nil {
		return err
	}

	return nil
}
