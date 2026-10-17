package mssql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

// CreateWithIdempotency keeps the patient row and its idempotency record in a
// single serializable SQL transaction. It is intentionally limited to the
// patient.create scope until the remaining business writes have matching
// transaction boundaries.
func (s *Store) CreateWithIdempotency(ctx context.Context, p domain.Patient, scope, key, requestHash string, expiresAt time.Time) (domain.Patient, bool, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return domain.Patient{}, false, err
	}
	defer func() { _ = tx.Rollback() }()

	var savedHash, state, resourceID string
	var savedExpires time.Time
	err = tx.QueryRowContext(ctx, `
		SELECT request_hash, state, resource_id, expires_at
		FROM dbo.idempotency_record WITH (UPDLOCK, HOLDLOCK)
		WHERE scope = @scope AND idem_key = @idem_key`,
		sql.Named("scope", scope), sql.Named("idem_key", key)).Scan(&savedHash, &state, &resourceID, &savedExpires)
	if err == nil {
		if !savedExpires.After(time.Now().UTC()) {
			if _, err := tx.ExecContext(ctx, `DELETE FROM dbo.idempotency_record WHERE scope=@scope AND idem_key=@idem_key`, sql.Named("scope", scope), sql.Named("idem_key", key)); err != nil {
				return domain.Patient{}, false, err
			}
		} else {
			if savedHash != requestHash {
				return domain.Patient{}, false, port.ErrIdempotencyConflict
			}
			switch state {
			case "PENDING":
				return domain.Patient{}, false, port.ErrIdempotencyInProgress
			case "COMPLETED":
				row, getErr := scanPatient(tx.QueryRowContext(ctx, `SELECT `+patientCols+` FROM dbo.patient WHERE patient_id=@patient_id AND is_deleted=0`, sql.Named("patient_id", resourceID)))
				if errors.Is(getErr, sql.ErrNoRows) {
					return domain.Patient{}, false, port.ErrNotFound
				}
				if getErr != nil {
					return domain.Patient{}, false, getErr
				}
				if err := tx.Commit(); err != nil {
					return domain.Patient{}, false, err
				}
				return *row, true, nil
			case "FAILED":
				if _, err := tx.ExecContext(ctx, `DELETE FROM dbo.idempotency_record WHERE scope=@scope AND idem_key=@idem_key`, sql.Named("scope", scope), sql.Named("idem_key", key)); err != nil {
					return domain.Patient{}, false, err
				}
			default:
				return domain.Patient{}, false, port.ErrIdempotencyInProgress
			}
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return domain.Patient{}, false, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO dbo.idempotency_record(scope, idem_key, request_hash, state, resource_type, resource_id, created_at, expires_at)
		VALUES(@scope, @idem_key, @request_hash, 'PENDING', 'Patient', NULL, SYSUTCDATETIME(), @expires_at)`,
		sql.Named("scope", scope), sql.Named("idem_key", key), sql.Named("request_hash", requestHash), sql.Named("expires_at", expiresAt.UTC())); err != nil {
		return domain.Patient{}, false, err
	}
	if err := insertPatient(ctx, tx, p); err != nil {
		return domain.Patient{}, false, classifyWriteError(err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE dbo.idempotency_record
		SET state='COMPLETED', resource_type='Patient', resource_id=@resource_id, completed_at=SYSUTCDATETIME()
		WHERE scope=@scope AND idem_key=@idem_key`,
		sql.Named("resource_id", p.PatientID), sql.Named("scope", scope), sql.Named("idem_key", key)); err != nil {
		return domain.Patient{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Patient{}, false, err
	}
	return p, false, nil
}

type sqlExecer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func insertPatient(ctx context.Context, execer sqlExecer, p domain.Patient) error {
	_, err := execer.ExecContext(ctx, `
		INSERT INTO dbo.patient (
			patient_id, hosp_pk_no, patient_no, card_no, name, sex, birthday,
			id_type, id_number, phone, address, fare_identity, member_card_no, member_level, contract_unit,
			patient_certificate_no, birth_place, ethnicity, nationality, marital_status, occupation_category,
			occupation, education, work_unit, work_unit_phone, work_unit_postcode, household_address, household_phone, household_postcode,
			contact_name, contact_address, contact_phone, contact_relation, card_status, status,
			created_at, created_by, updated_at, updated_by, is_deleted
		) VALUES (
			@patient_id, @hosp_pk_no, @patient_no, @card_no, @name, @sex, @birthday,
			@id_type, @id_number, @phone, @address, @fare_identity, @member_card_no, @member_level, @contract_unit,
			@patient_certificate_no, @birth_place, @ethnicity, @nationality, @marital_status, @occupation_category,
			@occupation, @education, @work_unit, @work_unit_phone, @work_unit_postcode, @household_address, @household_phone, @household_postcode,
			@contact_name, @contact_address, @contact_phone, @contact_relation, @card_status, @status,
			@created_at, @created_by, @updated_at, @updated_by, 0
		)`,
		sql.Named("patient_id", p.PatientID),
		sql.Named("hosp_pk_no", p.HospPkNo),
		sql.Named("patient_no", nullIfEmpty(p.PatientNo)),
		sql.Named("card_no", nullIfEmpty(p.CardNo)),
		sql.Named("name", p.Name),
		sql.Named("sex", nullIfEmpty(p.Sex)),
		sql.Named("birthday", p.Birthday),
		sql.Named("id_type", nullIfEmpty(p.IDType)),
		sql.Named("id_number", nullIfEmpty(p.IDNumber)),
		sql.Named("phone", nullIfEmpty(p.Phone)),
		sql.Named("address", nullIfEmpty(p.Address)),
		sql.Named("fare_identity", nullIfEmpty(p.FareIdentity)),
		sql.Named("member_card_no", nullIfEmpty(p.MemberCardNo)),
		sql.Named("member_level", nullIfEmpty(p.MemberLevel)),
		sql.Named("contract_unit", nullIfEmpty(p.ContractUnit)),
		sql.Named("patient_certificate_no", nullIfEmpty(p.PatientCertificateNo)),
		sql.Named("birth_place", nullIfEmpty(p.BirthPlace)),
		sql.Named("ethnicity", nullIfEmpty(p.Ethnicity)),
		sql.Named("nationality", nullIfEmpty(p.Nationality)),
		sql.Named("marital_status", nullIfEmpty(p.MaritalStatus)),
		sql.Named("occupation_category", nullIfEmpty(p.OccupationCategory)),
		sql.Named("occupation", nullIfEmpty(p.Occupation)),
		sql.Named("education", nullIfEmpty(p.Education)),
		sql.Named("work_unit", nullIfEmpty(p.WorkUnit)),
		sql.Named("work_unit_phone", nullIfEmpty(p.WorkUnitPhone)),
		sql.Named("work_unit_postcode", nullIfEmpty(p.WorkUnitPostcode)),
		sql.Named("household_address", nullIfEmpty(p.HouseholdAddress)),
		sql.Named("household_phone", nullIfEmpty(p.HouseholdPhone)),
		sql.Named("household_postcode", nullIfEmpty(p.HouseholdPostcode)),
		sql.Named("contact_name", nullIfEmpty(p.ContactName)),
		sql.Named("contact_address", nullIfEmpty(p.ContactAddress)),
		sql.Named("contact_phone", nullIfEmpty(p.ContactPhone)),
		sql.Named("contact_relation", nullIfEmpty(p.ContactRelation)),
		sql.Named("card_status", p.CardStatus),
		sql.Named("status", p.Status),
		sql.Named("created_at", p.CreatedAt.UTC()),
		sql.Named("created_by", nullIfEmpty(p.CreatedBy)),
		sql.Named("updated_at", p.UpdatedAt.UTC()),
		sql.Named("updated_by", nullIfEmpty(p.UpdatedBy)),
	)
	return err
}
