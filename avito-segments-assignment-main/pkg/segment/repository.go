package segment

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"os"
	"time"
	"usersegmentator/config"
	"usersegmentator/pkg/errors"
)

type Repository interface {
	InsertSegment(ctx context.Context, segmentSlug string) error
	DeleteSegment(ctx context.Context, segmentSlug string) error
	UnassignSegments(ctx context.Context, userID []int, segmentsToUnassign []string) error
	AssignSegments(ctx context.Context, userID []int, segmentsToAssign []string, ttl int) error
	GetUserSegments(ctx context.Context, userID int) (*UserSegments, error)
	GetNRandomUsersWithoutSegment(n int, slug string) ([]int, error)
	GetActiveUsersAmount(ctx context.Context) (int, error)
	GetSegmentsIDs(ctx context.Context, segmentSlugs []string) ([]int, error)
	AutoAssignSegment(ctx context.Context, fraction int, slug string, ttl int) error
	RunTTLChecker()
}

type segmentsRepository struct {
	db      *sql.DB
	cfg     *config.Config
	InfoLog *log.Logger
	ErrLog  *log.Logger
}

func NewSegmentsRepo(db *sql.DB, cfg *config.Config) Repository {
	sr := &segmentsRepository{
		db:      db,
		cfg:     cfg,
		InfoLog: log.New(os.Stdout, "INFO\tSEGMENTS REPO\t", log.Ldate|log.Ltime),
		ErrLog:  log.New(os.Stdout, "ERROR\tSEGMENTS REPO\t", log.Ldate|log.Ltime),
	}

	go func() {
		sr.RunTTLChecker()
	}()
	return sr
}

func (sr *segmentsRepository) RunTTLChecker() {
	sr.InfoLog.Printf("TTL checker is running")
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	ctx := context.Background()

	for {
		select {
		case <-ticker.C:
			tx, err := sr.db.BeginTx(ctx, nil)
			if err != nil {
				sr.ErrLog.Printf("%s: %s", errors.ErrorBeginTransaction, err)
				continue
			}

			rows, err := tx.QueryContext(ctx, "SELECT id FROM user_segment_relation "+
				"WHERE date_unassigned <= CURRENT_TIMESTAMP AND is_active = TRUE")

			if err != nil {
				sr.ErrLog.Printf("error checking table for ttl: %s", err)
				if rbErr := tx.Rollback(); rbErr != nil {
					sr.ErrLog.Printf("rollback error: %s", rbErr)
					continue
				}
			}

			var curID int
			ids := []int{}

			for rows.Next() {
				err = rows.Scan(&curID)
				if err != nil {
					sr.ErrLog.Printf("error reading row: %s", err)
					continue
				}
				ids = append(ids, curID)
			}

			err = rows.Close()
			if err != nil {
				sr.ErrLog.Printf("error closing rows: %s", err)
			}

			for _, id := range ids {
				_, err := tx.ExecContext(ctx, "UPDATE user_segment_relation SET is_active = FALSE WHERE id = ?", id)
				if err != nil {
					sr.ErrLog.Printf("error unassigning segments: %s", err)
					if rbErr := tx.Rollback(); rbErr != nil {
						sr.ErrLog.Printf("rollback error: %s", rbErr)
					}
				}
			}

			err = tx.Commit()
			if err != nil {
				sr.ErrLog.Printf("%s: %s", errors.ErrorCommittingTransaction, err)
			}
		}
	}
}

func (sr *segmentsRepository) AutoAssignSegment(ctx context.Context, fraction int, slug string, ttl int) error {
	if fraction < 1 || fraction > 100 {
		sr.ErrLog.Printf("invalid fraction value: %d", fraction)
		return fmt.Errorf("invalid fraction value: %d", fraction)
	}

	activeUsers, err := sr.GetActiveUsersAmount(ctx)
	if err != nil {
		sr.ErrLog.Printf("%s", err)
		return err
	}

	sampleSize := int(math.Ceil(float64(activeUsers) * (float64(fraction) / 100))) //nolint:gomnd // creating percents

	users, err := sr.GetNRandomUsersWithoutSegment(sampleSize, slug)
	if err != nil {
		sr.ErrLog.Printf("%s", err)
		return err
	}

	err = sr.AssignSegments(ctx, users, []string{slug}, ttl)
	if err != nil {
		sr.ErrLog.Printf("%s", err)
		return err
	}

	return nil
}

func (sr *segmentsRepository) GetSegmentsIDs(ctx context.Context, segmentSlugs []string) ([]int, error) {
	ids := []int{}
	for _, f := range segmentSlugs {
		var curID int
		row, err := sr.db.QueryContext(ctx, "SELECT id FROM segments WHERE slug = ? LIMIT 1", f)
		if err != nil {
			return []int{}, err
		}
		row.Next()
		err = row.Scan(&curID)
		if err != nil {
			return []int{}, err
		}

		err = row.Close()
		if err != nil {
			return []int{}, err
		}

		ids = append(ids, curID)
	}
	return ids, nil
}

func (sr *segmentsRepository) GetNRandomUsersWithoutSegment(n int, slug string) ([]int, error) {
	userIDs := []int{}

	rows, err := sr.db.Query(
		`SELECT DISTINCT u.id FROM users u
				WHERE (SELECT user_id 
					   FROM user_segment_relation 
					   WHERE user_id = u.id 
					   AND segment_id = 
							(SELECT id 
							FROM segments 
							WHERE slug = ? 
							LIMIT 1) 
					   AND is_active = TRUE 
					   ORDER BY date_assigned 
					   LIMIT 1) IS NULL 
					   AND is_active = TRUE
				ORDER BY RAND() LIMIT ?`,
		slug,
		n,
	)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var id int
		err = rows.Scan(&id)
		if err != nil {
			return nil, err
		}
		userIDs = append(userIDs, id)
	}
	err = rows.Close()
	if err != nil {
		return nil, err
	}

	return userIDs, nil
}

func (sr *segmentsRepository) GetActiveUsersAmount(ctx context.Context) (int, error) {
	var amount int

	row, err := sr.db.QueryContext(ctx, "SELECT COUNT(id) FROM users WHERE is_active = TRUE")
	if err != nil {
		return -1, err
	}

	row.Next()
	err = row.Scan(&amount)
	if err != nil {
		return -1, err
	}

	err = row.Close()
	if err != nil {
		return -1, err
	}

	return amount, nil
}

func (sr *segmentsRepository) InsertSegment(ctx context.Context, segmentSlug string) error {
	if segmentSlug == "" {
		return fmt.Errorf("empty segment slug")
	}

	_, err := sr.db.ExecContext(
		ctx,
		"INSERT INTO segments (`slug`) VALUES (?) ON DUPLICATE KEY UPDATE is_active = TRUE",
		segmentSlug,
	)
	if err != nil {
		return err
	}

	sr.InfoLog.Printf("InsertSegment — %s\n", segmentSlug)
	return nil
}

func (sr *segmentsRepository) DeleteSegment(ctx context.Context, segmentSlug string) error {
	segmentID, err := sr.GetSegmentsIDs(ctx, []string{segmentSlug})
	if err != nil {
		sr.ErrLog.Printf("%s: %s", errors.ErrorGettingSegmentID, err)
		return err
	}

	tx, err := sr.db.BeginTx(ctx, nil)
	if err != nil {
		sr.ErrLog.Printf("%s: %s", errors.ErrorBeginTransaction, err)
		return err
	}

	_, err = tx.ExecContext(ctx, "UPDATE segments SET is_active = FALSE WHERE id = ?", segmentID[0])
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction error: %w, rollback error: %s", err, rbErr)
		}
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		"UPDATE user_segment_relation "+
			"SET is_active = FALSE, date_unassigned = CURRENT_TIMESTAMP "+
			"WHERE segment_id = ?",
		segmentID[0],
	)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction error: %w, rollback error: %s", err, rbErr)
		}
		return err
	}

	err = tx.Commit()
	if err != nil {
		sr.ErrLog.Printf("%s: %s", errors.ErrorCommittingTransaction, err)
		return err
	}

	sr.InfoLog.Printf("DeleteSegment — %s\n", segmentSlug)
	return nil
}

func (sr *segmentsRepository) UnassignSegments(ctx context.Context, userID []int, segmentsToUnassign []string) error {
	if len(segmentsToUnassign) == 0 {
		return nil
	}

	ids, err := sr.GetSegmentsIDs(ctx, segmentsToUnassign)
	if err != nil {
		return err
	}

	tx, err := sr.db.BeginTx(ctx, nil)
	if err != nil {
		sr.ErrLog.Printf("%s: %s", errors.ErrorBeginTransaction, err)
		return err
	}

	for _, usr := range userID {
		for _, id := range ids {
			_, err = tx.ExecContext(
				ctx,
				"UPDATE user_segment_relation "+
					"SET is_active = FALSE, date_unassigned = CURRENT_TIMESTAMP "+
					"WHERE user_id = ? AND segment_id = ? AND is_active = TRUE",
				usr,
				id,
			)

			if err != nil {
				if rbErr := tx.Rollback(); rbErr != nil {
					return fmt.Errorf("transaction error: %w, rollback error: %s", err, rbErr)
				}
				return err
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		sr.ErrLog.Printf("%s: %s", errors.ErrorCommittingTransaction, err)
		return err
	}

	sr.InfoLog.Printf("UnassignSegments — %d\n", userID)
	return nil
}

func (sr *segmentsRepository) AssignSegments(
	ctx context.Context,
	userID []int,
	segmentsToAssign []string,
	ttl int,
) error {
	if len(segmentsToAssign) == 0 {
		return nil
	}
	if len(segmentsToAssign) == 0 {
		return nil
	}

	ids, err := sr.GetSegmentsIDs(ctx, segmentsToAssign)
	if err != nil {
		return err
	}

	tx, err := sr.db.BeginTx(ctx, nil)
	if err != nil {
		sr.ErrLog.Printf("%s: %s", errors.ErrorBeginTransaction, err)
		return err
	}

	for _, usr := range userID {
		for _, segmentID := range ids {
			var rows *sql.Rows
			rows, err = tx.QueryContext(
				ctx,
				"SELECT id FROM user_segment_relation WHERE is_active = TRUE AND user_id = ? AND segment_id = ?",
				usr,
				segmentID,
			)

			if err != nil {
				if rbErr := tx.Rollback(); rbErr != nil {
					return fmt.Errorf("transaction error: %w, rollback error: %s", err, rbErr)
				}
				return err
			}

			ok := rows.Next()

			err = rows.Close()
			if err != nil {
				return err
			}

			if ok {
				return nil
			}

			result, err := tx.ExecContext(
				ctx,
				"INSERT INTO user_segment_relation (`user_id`, `segment_id`) VALUES (?, ?)",
				usr,
				segmentID,
			)
			if err != nil {
				if rbErr := tx.Rollback(); rbErr != nil {
					return fmt.Errorf("transaction error: %w, rollback error: %s", err, rbErr)
				}
				return err
			}

			if lastID, err := result.LastInsertId(); err == nil && ttl != 0 {
				unassignTime := time.Now().AddDate(0, 0, ttl)
				_, err = tx.ExecContext(
					ctx,
					"UPDATE user_segment_relation SET date_unassigned = ? WHERE id = ?",
					unassignTime,
					lastID,
				)
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		sr.ErrLog.Printf("%s: %s", errors.ErrorCommittingTransaction, err)
		return err
	}

	sr.InfoLog.Printf("AssignSegments — %d\n", userID)
	return nil
}

func (sr *segmentsRepository) GetUserSegments(ctx context.Context, userID int) (*UserSegments, error) {
	rows, err := sr.db.QueryContext(
		ctx,
		"SELECT slug FROM segments "+
			"WHERE id IN ("+
			"SELECT segment_id FROM user_segment_relation "+
			"WHERE user_id = ? AND is_active = TRUE"+
			") AND is_active = TRUE",
		userID,
	)
	if err != nil {
		return nil, err
	}

	userSegments := &UserSegments{
		UserID:   userID,
		Segments: []string{},
	}

	for rows.Next() {
		var segment string
		err = rows.Scan(&segment)
		if err != nil {
			return nil, err
		}
		userSegments.Segments = append(userSegments.Segments, segment)
	}
	err = rows.Close()
	if err != nil {
		return nil, err
	}

	sr.InfoLog.Printf("GetSegments — %d\n", userID)
	return userSegments, nil
}
