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

// SQL follows migrations/sqlserver/core/002_patient. Do not add columns here.
// Empty id_number / card_no are stored as NULL so the filtered unique indexes allow repeats.
// Birthday ≤ today is enforced in the app; the table CHECK only requires birthday > 1900-01-01.

const patientCols = `patient_id, hosp_pk_no, patient_no, card_no, name, sex, birthday,
	id_type, id_number, phone, address, fare_identity, card_status, status, created_at`

func (s *Store) Create(ctx context.Context, p domain.Patient) error {
	idNumber := nullIfEmpty(p.IDNumber)
	cardNo := nullIfEmpty(p.CardNo)
	var kind string
	err := s.db.QueryRowContext(ctx, `
		SELECT CASE
			WHEN @id_number IS NOT NULL AND EXISTS (
				SELECT 1 FROM dbo.patient
				WHERE is_deleted = 0 AND id_number = @id_number AND id_number <> ''
			) THEN 'id'
			WHEN @card_no IS NOT NULL AND EXISTS (
				SELECT 1 FROM dbo.patient
				WHERE is_deleted = 0 AND card_no = @card_no AND card_no <> ''
			) THEN 'card'
			WHEN EXISTS (
				SELECT 1 FROM dbo.patient
				WHERE is_deleted = 0 AND hosp_pk_no = @hosp_pk_no
			) THEN 'hosp'
			ELSE ''
		END`,
		sql.Named("id_number", idNumber),
		sql.Named("card_no", cardNo),
		sql.Named("hosp_pk_no", p.HospPkNo),
	).Scan(&kind)
	if err != nil {
		return err
	}
	switch kind {
	case "id":
		return port.ErrDuplicateIDNumber
	case "card":
		return port.ErrDuplicateCardNo
	case "hosp":
		return port.ErrDuplicateBusinessNo
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO dbo.patient (
			patient_id, hosp_pk_no, patient_no, card_no, name, sex, birthday,
			id_type, id_number, phone, address, fare_identity, card_status, status,
			created_at, created_by, updated_at, updated_by, is_deleted
		) VALUES (
			@patient_id, @hosp_pk_no, @patient_no, @card_no, @name, @sex, @birthday,
			@id_type, @id_number, @phone, @address, @fare_identity, @card_status, @status,
			@created_at, @created_by, @updated_at, @updated_by, 0
		)`,
		sql.Named("patient_id", p.PatientID),
		sql.Named("hosp_pk_no", p.HospPkNo),
		sql.Named("patient_no", nullIfEmpty(p.PatientNo)),
		sql.Named("card_no", cardNo),
		sql.Named("name", p.Name),
		sql.Named("sex", nullIfEmpty(p.Sex)),
		sql.Named("birthday", p.Birthday),
		sql.Named("id_type", nullIfEmpty(p.IDType)),
		sql.Named("id_number", idNumber),
		sql.Named("phone", nullIfEmpty(p.Phone)),
		sql.Named("address", nullIfEmpty(p.Address)),
		sql.Named("fare_identity", nullIfEmpty(p.FareIdentity)),
		sql.Named("card_status", p.CardStatus),
		sql.Named("status", p.Status),
		sql.Named("created_at", p.CreatedAt.UTC()),
		sql.Named("created_by", nullIfEmpty(p.CreatedBy)),
		sql.Named("updated_at", p.UpdatedAt.UTC()),
		sql.Named("updated_by", nullIfEmpty(p.UpdatedBy)),
	)
	if err != nil {
		return classifyWriteError(err)
	}
	return nil
}

func (s *Store) Get(ctx context.Context, patientID string) (*domain.Patient, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+patientCols+`
		FROM dbo.patient WHERE patient_id = @patient_id AND is_deleted = 0`,
		sql.Named("patient_id", patientID))
	p, err := scanPatient(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, port.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Store) Search(ctx context.Context, q port.PatientQuery) ([]domain.Patient, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT TOP (@limit) `+patientCols+`
		FROM dbo.patient
		WHERE is_deleted = 0
		  AND (@card_no IS NULL OR card_no = @card_no)
		  AND (@id_number IS NULL OR id_number = @id_number)
		  AND (@hosp_pk_no IS NULL OR hosp_pk_no = @hosp_pk_no)
		  AND (@name IS NULL OR name LIKE @name ESCAPE '\')
		  AND (@address IS NULL OR address LIKE @address ESCAPE '\')
		ORDER BY created_at, patient_id`,
		sql.Named("limit", limit),
		sql.Named("card_no", nullIfEmpty(q.CardNo)),
		sql.Named("id_number", nullIfEmpty(q.IDNumber)),
		sql.Named("hosp_pk_no", nullIfEmpty(q.HospPkNo)),
		sql.Named("name", likeOrNil(q.Name)),
		sql.Named("address", likeOrNil(q.Address)),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Patient, 0)
	for rows.Next() {
		p, err := scanPatient(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanPatient(sc scanner) (*domain.Patient, error) {
	var p domain.Patient
	var patientNo, cardNo, sex, idType, idNumber, phone, address, fare sql.NullString
	var birthday time.Time
	if err := sc.Scan(
		&p.PatientID, &p.HospPkNo, &patientNo, &cardNo, &p.Name, &sex, &birthday,
		&idType, &idNumber, &phone, &address, &fare, &p.CardStatus, &p.Status, &p.CreatedAt,
	); err != nil {
		return nil, err
	}
	p.PatientNo = patientNo.String
	p.CardNo = cardNo.String
	p.Sex = sex.String
	p.Birthday = birthday
	p.IDType = idType.String
	p.IDNumber = idNumber.String
	p.Phone = phone.String
	p.Address = address.String
	p.FareIdentity = fare.String
	return &p, nil
}

func likeOrNil(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return likeContains(s)
}

// likeContains builds a substring pattern. % _ [ \ are escaped for ESCAPE '\'.
func likeContains(s string) string {
	var b strings.Builder
	b.WriteByte('%')
	for _, r := range s {
		switch r {
		case '%', '_', '[', '\\':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	b.WriteByte('%')
	return b.String()
}
