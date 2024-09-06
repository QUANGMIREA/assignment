package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"usersegmentator/config"
	"usersegmentator/pkg/errors"
	"usersegmentator/pkg/history"
)

type HistoryHandler struct {
	HistoryRepo history.Repository
	InfoLog     *log.Logger
	ErrLog      *log.Logger
}

func NewHistoryHandler(db *sql.DB, cfg *config.Config) *HistoryHandler {
	return &HistoryHandler{
		HistoryRepo: history.NewHistoryRepo(db, cfg),
		InfoLog:     log.New(os.Stdout, "INFO\tHistory HANDLER\t", log.Ldate|log.Ltime),
		ErrLog:      log.New(os.Stdout, "ERROR\tHistory HANDLER\t", log.Ldate|log.Ltime),
	}
}

// GetUserHistory godoc
//
//	@Summary		receive report on user segments assignments and unassignments
//	@Description	receive report on user segments assignments and unassignments within the given dates
//	@Tags         	History
//	@Accept			json
//	@Produce		json
//	@Param 			request		body 	history.Request true "The input struct"
//	@Success		200	{object} history.ReportResponse
//	@Failure		400	{string} string "bad input"
//	@Failure		500	{string} string "something went wrong"
//	@Router			/api/get_user_history [get]
func (rh *HistoryHandler) GetUserHistory(w http.ResponseWriter, r *http.Request) {
	receivedRequest := &history.Request{}

	err := errors.ValidateAndParseJSON(r, receivedRequest)
	if err != nil {
		rh.ErrLog.Printf("%s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	dates, err := rh.HistoryRepo.ParseAndValidateDates(receivedRequest.StartDate, receivedRequest.EndDate)
	if err != nil {
		rh.ErrLog.Printf("%s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userHistory, err := rh.HistoryRepo.GetUserHistory(r.Context(), receivedRequest.UserID, dates)
	if err != nil {
		rh.ErrLog.Printf("%s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	url, err := rh.HistoryRepo.CreateCSV(userHistory)
	if err != nil {
		rh.ErrLog.Printf("%s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(history.ReportResponse{CsvURL: url})

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = w.Write(resp)
	if err != nil {
		rh.ErrLog.Printf("%s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
