package mssql

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
	"github.com/google/uuid"
)

func TestPatientCreateIdempotencySQL(t *testing.T) {
	dsn := os.Getenv("BHIS_DASHBOARD_TEST_DSN")
	if dsn == "" {
		t.Skip("set BHIS_DASHBOARD_TEST_DSN to isolated BHIS_PHASE2_TEST_20261015")
	}
	db, err := Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	var databaseName string
	if err := db.QueryRowContext(ctx, `SELECT DB_NAME()`).Scan(&databaseName); err != nil {
		t.Fatal(err)
	}
	if databaseName != "BHIS_PHASE2_TEST_20261015" {
		t.Fatalf("refusing patient idempotency fixture in %q", databaseName)
	}

	suffix := uuid.NewString()[:20]
	p := domain.Patient{
		PatientID:            "idem-patient-" + suffix,
		HospPkNo:             "IDH" + suffix,
		PatientNo:            "IDP" + suffix,
		Name:                 "SQL幂等患者",
		Sex:                  domain.SexUnknown,
		Birthday:             time.Date(1990, 1, 2, 0, 0, 0, 0, time.UTC),
		FareIdentity:         "SELF",
		IDType:               "ID_CARD",
		IDNumber:             "",
		Phone:                "13800000000",
		Address:              "太原市示范路",
		MemberCardNo:         "M-" + suffix,
		MemberLevel:          "普通会员",
		ContractUnit:         "测试单位",
		PatientCertificateNo: "PC-" + suffix,
		BirthPlace:           "山西省太原市",
		Ethnicity:            "汉族",
		Nationality:          "中国",
		MaritalStatus:        "已婚",
		OccupationCategory:   "专业技术人员",
		Occupation:           "测试职业",
		Education:            "本科",
		WorkUnit:             "测试医院",
		WorkUnitPhone:        "03511234567",
		WorkUnitPostcode:     "030000",
		HouseholdAddress:     "太原市小店区",
		HouseholdPhone:       "03517654321",
		HouseholdPostcode:    "030012",
		ContactName:          "联系人",
		ContactAddress:       "太原市迎泽区",
		ContactPhone:         "13900000000",
		ContactRelation:      "配偶",
		CardStatus:           domain.CardStatusNotSent,
		Status:               domain.PatientStatusActive,
		CreatedAt:            time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	cleanup := func() {
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.idempotency_record WHERE scope=@scope AND idem_key=@key`, sql.Named("scope", "patient.create"), sql.Named("key", "sql-idem-"+suffix))
		_, _ = db.ExecContext(ctx, `DELETE FROM dbo.patient WHERE patient_id=@id`, sql.Named("id", p.PatientID))
	}
	cleanup()
	defer cleanup()

	store := New(db)
	hash := strings.Repeat("a", 64)
	first, replayed, err := store.CreateWithIdempotency(ctx, p, "patient.create", "sql-idem-"+suffix, hash, time.Now().UTC().Add(time.Hour))
	if err != nil || replayed || first.PatientID != p.PatientID {
		t.Fatalf("first create = %#v, replayed=%v, err=%v", first, replayed, err)
	}
	second, replayed, err := store.CreateWithIdempotency(ctx, p, "patient.create", "sql-idem-"+suffix, hash, time.Now().UTC().Add(time.Hour))
	if err != nil || !replayed || second.PatientID != p.PatientID {
		t.Fatalf("replay = %#v, replayed=%v, err=%v", second, replayed, err)
	}
	gotDetails := [...]string{
		second.Address, second.MemberCardNo, second.MemberLevel, second.ContractUnit,
		second.PatientCertificateNo, second.BirthPlace, second.Ethnicity, second.Nationality,
		second.MaritalStatus, second.OccupationCategory, second.Occupation, second.Education,
		second.WorkUnit, second.WorkUnitPhone, second.WorkUnitPostcode, second.HouseholdAddress,
		second.HouseholdPhone, second.HouseholdPostcode, second.ContactName, second.ContactAddress,
		second.ContactPhone, second.ContactRelation,
	}
	wantDetails := [...]string{
		p.Address, p.MemberCardNo, p.MemberLevel, p.ContractUnit, p.PatientCertificateNo,
		p.BirthPlace, p.Ethnicity, p.Nationality, p.MaritalStatus, p.OccupationCategory,
		p.Occupation, p.Education, p.WorkUnit, p.WorkUnitPhone, p.WorkUnitPostcode,
		p.HouseholdAddress, p.HouseholdPhone, p.HouseholdPostcode, p.ContactName,
		p.ContactAddress, p.ContactPhone, p.ContactRelation,
	}
	if gotDetails != wantDetails {
		t.Fatalf("detailed patient fields were not stored and loaded: got %#v want %#v", second, p)
	}
	changed := p
	changed.Name = "SQL幂等冲突"
	_, _, err = store.CreateWithIdempotency(ctx, changed, "patient.create", "sql-idem-"+suffix, strings.Repeat("b", 64), time.Now().UTC().Add(time.Hour))
	if !errors.Is(err, port.ErrIdempotencyConflict) {
		t.Fatalf("changed payload error = %v, want %v", err, port.ErrIdempotencyConflict)
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.patient WHERE patient_id=@id`, sql.Named("id", p.PatientID)).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("patient rows = %d, want 1", count)
	}
}
