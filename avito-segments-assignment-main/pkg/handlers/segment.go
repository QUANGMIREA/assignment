package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"usersegmentator/config"
	"usersegmentator/pkg/errors"
	"usersegmentator/pkg/segment"
)

type SegmentsHandler struct {
	SegmentsRepo segment.Repository
	InfoLog      *log.Logger
	ErrLog       *log.Logger
}

func NewSegmentsHandler(db *sql.DB, cfg *config.Config) *SegmentsHandler {
	return &SegmentsHandler{
		SegmentsRepo: segment.NewSegmentsRepo(db, cfg),
		InfoLog:      log.New(os.Stdout, "INFO\tSEGMENTS HANDLER\t", log.Ldate|log.Ltime),
		ErrLog:       log.New(os.Stdout, "ERROR\tSEGMENTS HANDLER\t", log.Ldate|log.Ltime),
	}
}

// AddSegment godoc
//
//	@Summary		creates new segment
//	@Description	creates new segment
//	@Tags         	Segments
//	@Accept			json
//	@Param 			request		body 	segment.RequestSegmentSlug true "fraction — optional"
//	@Success		201	{string} string "created"
//	@Failure		400	{string} string "bad input"
//	@Failure		500	{string} string "something went wrong"
//	@Router			/api/create_segment [post]
func (sh *SegmentsHandler) AddSegment(w http.ResponseWriter, r *http.Request) {
	f := &segment.Template{}

	err := errors.ValidateAndParseJSON(r, f)
	if err != nil {
		sh.ErrLog.Printf("%s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = sh.SegmentsRepo.InsertSegment(r.Context(), f.SegmentSlug)
	if err != nil {
		sh.ErrLog.Printf("%s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if f.Fraction != 0 {
		err = sh.SegmentsRepo.AutoAssignSegment(r.Context(), f.Fraction, f.SegmentSlug, 0)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
}

// DeleteSegment godoc
//
//	@Summary		deletes existing segment
//	@Description	deletes existing segment
//	@Tags         	Segments
//	@Accept			json
//	@Param 			request		body 	segment.RequestSegmentSlug true "The input struct"
//	@Success		200	{string} string "deleted"
//	@Failure		400	{string} string "bad input"
//	@Failure		500	{string} string "something went wrong"
//	@Router			/api/delete_segment [delete]
func (sh *SegmentsHandler) DeleteSegment(w http.ResponseWriter, r *http.Request) {
	f := &segment.Template{}

	err := errors.ValidateAndParseJSON(r, f)
	if err != nil {
		sh.ErrLog.Printf("%s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = sh.SegmentsRepo.DeleteSegment(r.Context(), f.SegmentSlug)
	if err != nil {
		sh.ErrLog.Printf("%s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// UpdateUserSegments godoc
//
//	@Summary		assign and unassign segments from user
//	@Description	assign and unassign segments from user
//	@Tags         	Segments
//	@Accept			json
//	@Param 			request		body 	segment.RequestUpdateSegments true "The input struct"
//	@Success		200	{string} string "assigned and unassigned"
//	@Failure		400	{string} string "bad input"
//	@Failure		500	{string} string "something went wrong"
//	@Router			/api/update_user_segments [post]
func (sh *SegmentsHandler) UpdateUserSegments(w http.ResponseWriter, r *http.Request) {
	f := &segment.Template{}

	err := errors.ValidateAndParseJSON(r, f)
	if err != nil {
		sh.ErrLog.Printf("%s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = sh.SegmentsRepo.AssignSegments(r.Context(), []int{f.UserID}, f.AssignSegments, f.TTL)
	if err != nil {
		sh.ErrLog.Printf("%s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = sh.SegmentsRepo.UnassignSegments(r.Context(), []int{f.UserID}, f.UnassignSegments)
	if err != nil {
		sh.ErrLog.Printf("%s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetUserSegments godoc
//
//	@Summary		receive segments assigned to user
//	@Description	receive segments assigned to user
//	@Tags         	Segments
//	@Accept			json
//	@Produce		json
//	@Param 			request		body 	segment.RequestUserID true "The input struct"
//	@Success		200	{object} segment.UserSegments
//	@Failure		400	{string} string "bad input"
//	@Failure		500	{string} string "something went wrong"
//	@Router			/api/get_user_segments [get]
func (sh *SegmentsHandler) GetUserSegments(w http.ResponseWriter, r *http.Request) {
	receivedUserID := &segment.Template{}

	err := errors.ValidateAndParseJSON(r, receivedUserID)
	if err != nil {
		sh.ErrLog.Printf("%s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userSegments, err := sh.SegmentsRepo.GetUserSegments(r.Context(), receivedUserID.UserID)
	if err != nil {
		sh.ErrLog.Printf("%s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(userSegments)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = w.Write(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
