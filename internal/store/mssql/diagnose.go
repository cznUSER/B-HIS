package mssql

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

const diagnoseCols = `diagnose_id, encounter_id, status, content, created_at, created_by, updated_at, updated_by`

func (s *Store) CreateDiagnose(ctx context.Context, d domain.Diagnose) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO dbo.diagnose (
			diagnose_id, encounter_id, status, content, created_at, created_by, updated_at, updated_by, is_deleted
		) VALUES (
			@diagnose_id, @encounter_id, @status, @content, @created_at, @created_by, @updated_at, @updated_by, 0
		)`,
		sql.Named("diagnose_id", d.DiagnoseID),
		sql.Named("encounter_id", d.RegisterID),
		sql.Named("status", d.Status),
		sql.Named("content", nullIfEmpty(d.Content)),
		sql.Named("created_at", d.CreatedAt.UTC()),
		sql.Named("created_by", nullIfEmpty(d.CreatedBy)),
		sql.Named("updated_at", d.UpdatedAt.UTC()),
		sql.Named("updated_by", nullIfEmpty(d.UpdatedBy)),
	)
	if err != nil {
		return classifyWriteError(err)
	}
	return nil
}

func (s *Store) GetDiagnose(ctx context.Context, diagnoseID string) (*domain.Diagnose, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+diagnoseCols+` FROM dbo.diagnose WHERE diagnose_id = @diagnose_id AND is_deleted = 0`, sql.Named("diagnose_id", diagnoseID))
	d, err := scanDiagnose(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Store) ListDiagnoses(ctx context.Context, q port.DiagnoseQuery) ([]domain.Diagnose, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT TOP (@limit) `+diagnoseCols+`
		FROM dbo.diagnose
		WHERE is_deleted = 0 AND (@encounter_id IS NULL OR encounter_id = @encounter_id)
		ORDER BY created_at, diagnose_id`,
		sql.Named("limit", limit), sql.Named("encounter_id", nullIfEmpty(strings.TrimSpace(q.RegisterID))))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Diagnose, 0)
	for rows.Next() {
		d, err := scanDiagnose(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) CommitDiagnose(ctx context.Context, diagnoseID, operatorID string, at time.Time) (*domain.Diagnose, error) {
	return s.transitionDiagnose(ctx, diagnoseID, operatorID, at, domain.DiagnoseStatusCommit, port.ErrDiagnoseNotCommittable, "SAVE")
}

func (s *Store) AbateDiagnose(ctx context.Context, diagnoseID, operatorID string, at time.Time) (*domain.Diagnose, error) {
	return s.transitionDiagnose(ctx, diagnoseID, operatorID, at, domain.DiagnoseStatusAbate, port.ErrDiagnoseNotAbatable, "SAVE", "COMMIT")
}

func (s *Store) transitionDiagnose(ctx context.Context, diagnoseID, operatorID string, at time.Time, next string, illegal error, allowed ...string) (*domain.Diagnose, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var status string
	err = tx.QueryRowContext(ctx, `SELECT status FROM dbo.diagnose WITH (UPDLOCK, ROWLOCK) WHERE diagnose_id = @diagnose_id AND is_deleted = 0`, sql.Named("diagnose_id", diagnoseID)).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	ok := false
	for _, candidate := range allowed {
		if status == candidate {
			ok = true
			break
		}
	}
	if !ok {
		return nil, illegal
	}
	at = at.UTC()
	res, err := tx.ExecContext(ctx, `UPDATE dbo.diagnose SET status = @status, updated_at = @updated_at, updated_by = @updated_by WHERE diagnose_id = @diagnose_id AND is_deleted = 0 AND status = @previous`,
		sql.Named("status", next), sql.Named("updated_at", at), sql.Named("updated_by", nullIfEmpty(operatorID)), sql.Named("diagnose_id", diagnoseID), sql.Named("previous", status))
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n != 1 {
		return nil, illegal
	}
	d, err := scanDiagnose(tx.QueryRowContext(ctx, `SELECT `+diagnoseCols+` FROM dbo.diagnose WHERE diagnose_id = @diagnose_id AND is_deleted = 0`, sql.Named("diagnose_id", diagnoseID)))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return d, nil
}

func scanDiagnose(sc scanner) (*domain.Diagnose, error) {
	var d domain.Diagnose
	var content, createdBy, updatedBy sql.NullString
	if err := sc.Scan(&d.DiagnoseID, &d.RegisterID, &d.Status, &content, &d.CreatedAt, &createdBy, &d.UpdatedAt, &updatedBy); err != nil {
		return nil, err
	}
	d.Content = content.String
	d.CreatedBy = createdBy.String
	d.UpdatedBy = updatedBy.String
	return &d, nil
}
