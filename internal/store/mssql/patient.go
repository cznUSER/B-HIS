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

// SQL follows migrations/sqlserver/core/002_patient and 016_patient_registration_detail.
// Empty id_number / card_no are stored as NULL so the filtered unique indexes allow repeats.
// Birthday ≤ today is enforced in the app; the table CHECK only requires birthday > 1900-01-01.

const patientCols = `patient_id, hosp_pk_no, patient_no, card_no, name, sex, birthday,
	id_type, id_number, phone, address, fare_identity, member_card_no, member_level, contract_unit,
	patient_certificate_no, birth_place, ethnicity, nationality, marital_status, occupation_category,
	occupation, education, work_unit, work_unit_phone, work_unit_postcode, household_address, household_phone, household_postcode,
	contact_name, contact_address, contact_phone, contact_relation, card_status, status, created_at`

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
	err = insertPatient(ctx, s.db, p)
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
	rows, err := s.db.QueryContext(ctx, `WITH patient_page AS (
		SELECT `+patientCols+`, ROW_NUMBER() OVER (ORDER BY created_at, patient_id) AS row_num
		FROM dbo.patient
		WHERE is_deleted = 0
		  AND (@patient_no IS NULL OR patient_no = @patient_no)
		  AND (@card_no IS NULL OR card_no = @card_no)
		  AND (@id_number IS NULL OR id_number = @id_number)
		  AND (@hosp_pk_no IS NULL OR hosp_pk_no = @hosp_pk_no)
		  AND (@name IS NULL OR name LIKE @name ESCAPE '\')
		  AND (@address IS NULL OR address LIKE @address ESCAPE '\')
	)
	SELECT `+patientCols+`
	FROM patient_page
	WHERE row_num > @offset AND row_num <= @offset + @limit
	ORDER BY row_num`,
		sql.Named("limit", limit),
		sql.Named("offset", q.Offset),
		sql.Named("patient_no", nullIfEmpty(q.PatientNo)),
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
	var memberCardNo, memberLevel, contractUnit, patientCertificateNo, birthPlace sql.NullString
	var ethnicity, nationality, maritalStatus, occupationCategory, occupation, education sql.NullString
	var workUnit, workUnitPhone, workUnitPostcode, householdAddress sql.NullString
	var householdPhone, householdPostcode sql.NullString
	var contactName, contactAddress, contactPhone, contactRelation sql.NullString
	var birthday time.Time
	if err := sc.Scan(
		&p.PatientID, &p.HospPkNo, &patientNo, &cardNo, &p.Name, &sex, &birthday,
		&idType, &idNumber, &phone, &address, &fare, &memberCardNo, &memberLevel, &contractUnit,
		&patientCertificateNo, &birthPlace, &ethnicity, &nationality, &maritalStatus, &occupationCategory,
		&occupation, &education, &workUnit, &workUnitPhone, &workUnitPostcode, &householdAddress, &householdPhone, &householdPostcode,
		&contactName, &contactAddress, &contactPhone, &contactRelation, &p.CardStatus, &p.Status, &p.CreatedAt,
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
	p.MemberCardNo = memberCardNo.String
	p.MemberLevel = memberLevel.String
	p.ContractUnit = contractUnit.String
	p.PatientCertificateNo = patientCertificateNo.String
	p.BirthPlace = birthPlace.String
	p.Ethnicity = ethnicity.String
	p.Nationality = nationality.String
	p.MaritalStatus = maritalStatus.String
	p.OccupationCategory = occupationCategory.String
	p.Occupation = occupation.String
	p.Education = education.String
	p.WorkUnit = workUnit.String
	p.WorkUnitPhone = workUnitPhone.String
	p.WorkUnitPostcode = workUnitPostcode.String
	p.HouseholdAddress = householdAddress.String
	p.HouseholdPhone = householdPhone.String
	p.HouseholdPostcode = householdPostcode.String
	p.ContactName = contactName.String
	p.ContactAddress = contactAddress.String
	p.ContactPhone = contactPhone.String
	p.ContactRelation = contactRelation.String
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
