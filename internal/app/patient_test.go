package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/store/memory"
	"github.com/cznUSER/B-HIS/internal/store/seed"
)

// patientClock is 2026-10-12 12:00 in Asia/Shanghai, so "today" for birthday is that date.
var patientClock = time.Date(2026, 10, 12, 4, 0, 0, 0, time.UTC)

func newPatientSvc(t *testing.T) (*Service, *memory.Store) {
	t.Helper()
	st := memory.New()
	svc := &Service{
		Users:    st,
		Audit:    st,
		Patients: st,
		Now:      func() time.Time { return patientClock },
	}
	return svc, st
}

func systemClaims() *auth.Claims {
	c := &auth.Claims{Username: seed.DemoUsername, Roles: []string{string(domain.RoleSystem)}}
	c.Subject = seed.DemoUserID
	return c
}

func strPtr(s string) *string { return &s }

func validInput() CreatePatientInput {
	return CreatePatientInput{
		Name: "张三", Sex: "M", Birthday: "1990-01-15", FareIdentity: "SELF",
		IDType: "ID_CARD", IDNumber: "110101199001151234", Phone: "13800000000",
		Address: strPtr("上海市浦东新区示例路 1 号"),
	}
}

func fieldNames(err error) []string {
	var ae *Error
	if !errors.As(err, &ae) || ae.Details == nil {
		return nil
	}
	switch v := ae.Details["fields"].(type) {
	case []string:
		return v
	default:
		return nil
	}
}

func TestCreatePatientValidation(t *testing.T) {
	svc, _ := newPatientSvc(t)
	ctx := context.Background()
	cases := []struct {
		name   string
		mutate func(*CreatePatientInput)
		fields []string
	}{
		{name: "missing sex", mutate: func(in *CreatePatientInput) { in.Sex = "" }, fields: []string{"sex"}},
		{name: "bad sex", mutate: func(in *CreatePatientInput) { in.Sex = "X" }, fields: []string{"sex"}},
		{name: "missing fareIdentity", mutate: func(in *CreatePatientInput) { in.FareIdentity = "  " }, fields: []string{"fareIdentity"}},
		{name: "missing name", mutate: func(in *CreatePatientInput) { in.Name = "" }, fields: []string{"name"}},
		{name: "birthday floor", mutate: func(in *CreatePatientInput) { in.Birthday = "1900-01-01" }, fields: []string{"birthday"}},
		{name: "birthday before floor", mutate: func(in *CreatePatientInput) { in.Birthday = "1899-12-31" }, fields: []string{"birthday"}},
		{name: "birthday future", mutate: func(in *CreatePatientInput) { in.Birthday = "2026-10-13" }, fields: []string{"birthday"}},
		{name: "birthday format", mutate: func(in *CreatePatientInput) { in.Birthday = "1990/01/15" }, fields: []string{"birthday"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := validInput()
			tc.mutate(&in)
			_, err := svc.CreatePatient(ctx, systemClaims(), in, "req-v", "127.0.0.1")
			var ae *Error
			if !errors.As(err, &ae) || ae.Status != 400 || ae.Code != CodeValidation {
				t.Fatalf("%v", err)
			}
			got := fieldNames(err)
			if strings.Join(got, ",") != strings.Join(tc.fields, ",") {
				t.Fatalf("fields %v want %v", got, tc.fields)
			}
		})
	}
}

func TestCreatePatientOKAndDuplicates(t *testing.T) {
	svc, st := newPatientSvc(t)
	ctx := context.Background()
	addr := "浦东大道"
	in := validInput()
	in.Address = strPtr(addr)
	in.CardNo = "C202610120001"
	dto, err := svc.CreatePatient(ctx, systemClaims(), in, "req-ok", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if dto.Name != "张三" || dto.Sex != "M" || dto.Birthday != "1990-01-15" || dto.FareIdentity != "SELF" {
		t.Fatalf("%+v", dto)
	}
	if dto.Address != addr || dto.CardStatus != domain.CardStatusNormal || dto.Status != domain.PatientStatusActive {
		t.Fatalf("%+v", dto)
	}
	if !strings.HasPrefix(dto.HospPkNo, "H20261012-") || !strings.HasPrefix(dto.PatientNo, "P20261012-") {
		t.Fatalf("numbers %s %s", dto.HospPkNo, dto.PatientNo)
	}
	if dto.CreatedAt != "2026-10-12T04:00:00Z" || dto.PatientID == "" {
		t.Fatalf("%+v", dto)
	}
	var audited bool
	for _, e := range st.Audits() {
		if e.Action == domain.ActionPatientCreate && e.ResourceType == "Patient" && e.ResourceID == dto.PatientID && e.Outcome == domain.OutcomeSuccess {
			audited = true
		}
	}
	if !audited {
		t.Fatal("missing patient.create audit")
	}

	again := in
	again.CardNo = "C-other"
	_, err = svc.CreatePatient(ctx, systemClaims(), again, "req-dup-id", "127.0.0.1")
	var ae *Error
	if !errors.As(err, &ae) || ae.Status != 409 || ae.Code != CodeDuplicateIDNumber {
		t.Fatalf("id duplicate: %v", err)
	}

	otherType := in
	otherType.IDType = "PASSPORT"
	otherType.CardNo = "C-other-2"
	_, err = svc.CreatePatient(ctx, systemClaims(), otherType, "req-dup-id-type", "127.0.0.1")
	if !errors.As(err, &ae) || ae.Code != CodeDuplicateIDNumber {
		t.Fatalf("id number is unique without idType: %v", err)
	}

	card := validInput()
	card.IDNumber = ""
	card.CardNo = "C202610120001"
	_, err = svc.CreatePatient(ctx, systemClaims(), card, "req-dup-card", "127.0.0.1")
	if !errors.As(err, &ae) || ae.Status != 409 || ae.Code != CodeDuplicateCardNo {
		t.Fatalf("card duplicate: %v", err)
	}

	blank := validInput()
	blank.IDNumber = "  "
	blank.CardNo = ""
	blank.Address = nil
	first, err := svc.CreatePatient(ctx, systemClaims(), blank, "req-empty-1", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.CreatePatient(ctx, systemClaims(), blank, "req-empty-2", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if first.PatientID == second.PatientID || first.CardStatus != domain.CardStatusNotSent || first.Address != "" {
		t.Fatalf("empty id/address %+v %+v", first, second)
	}

	today := blank
	today.Birthday = "2026-10-12"
	today.Name = "当天"
	if _, err := svc.CreatePatient(ctx, systemClaims(), today, "req-today", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	dayAfterFloor := blank
	dayAfterFloor.Birthday = "1900-01-02"
	dayAfterFloor.Name = "最早"
	if _, err := svc.CreatePatient(ctx, systemClaims(), dayAfterFloor, "req-floor", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
}

func TestPatientSearchGetAndAuthz(t *testing.T) {
	svc, st := newPatientSvc(t)
	ctx := context.Background()
	in := validInput()
	in.CardNo = "CARD-1"
	in.Address = strPtr("上海市浦东新区")
	dto, err := svc.CreatePatient(ctx, systemClaims(), in, "req-s", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	other := validInput()
	other.Name = "李四"
	other.IDNumber = ""
	other.Address = strPtr("北京市")
	if _, err := svc.CreatePatient(ctx, systemClaims(), other, "req-s2", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}

	got, err := svc.GetPatient(ctx, systemClaims(), dto.PatientID)
	if err != nil || got.HospPkNo != dto.HospPkNo {
		t.Fatalf("%v %+v", err, got)
	}
	_, err = svc.GetPatient(ctx, systemClaims(), "missing")
	var ae *Error
	if !errors.As(err, &ae) || ae.Status != 404 || ae.Code != CodeNotFound {
		t.Fatalf("get missing: %v", err)
	}

	list, err := svc.SearchPatients(ctx, systemClaims(), PatientSearch{})
	if err != nil || len(list.Items) != 0 {
		t.Fatalf("empty query %+v %v", list, err)
	}
	byName, err := svc.SearchPatients(ctx, systemClaims(), PatientSearch{Name: "张"})
	if err != nil || len(byName.Items) != 1 || byName.Items[0].PatientID != dto.PatientID {
		t.Fatalf("name %+v %v", byName, err)
	}
	byID, err := svc.SearchPatients(ctx, systemClaims(), PatientSearch{IDNumber: in.IDNumber})
	if err != nil || len(byID.Items) != 1 {
		t.Fatalf("id %+v %v", byID, err)
	}
	byCard, err := svc.SearchPatients(ctx, systemClaims(), PatientSearch{CardNo: "CARD-1"})
	if err != nil || len(byCard.Items) != 1 {
		t.Fatalf("card %+v %v", byCard, err)
	}
	byHosp, err := svc.SearchPatients(ctx, systemClaims(), PatientSearch{HospPkNo: dto.HospPkNo})
	if err != nil || len(byHosp.Items) != 1 {
		t.Fatalf("hosp %+v %v", byHosp, err)
	}
	byAddr, err := svc.SearchPatients(ctx, systemClaims(), PatientSearch{Address: "浦东"})
	if err != nil || len(byAddr.Items) != 1 || byAddr.Items[0].PatientID != dto.PatientID {
		t.Fatalf("address %+v %v", byAddr, err)
	}

	pharm := domain.User{
		ID: "00000000-0000-4000-8000-000000000003", Username: "pharm", DisplayName: "药房",
		Roles: []domain.Role{domain.RolePharmacist}, Status: domain.StatusActive,
	}
	st.Add(pharm)
	pc := &auth.Claims{Roles: []string{"PHARMACIST"}}
	pc.Subject = pharm.ID
	_, err = svc.CreatePatient(ctx, pc, validInput(), "req-forbid", "127.0.0.1")
	if !errors.As(err, &ae) || ae.Status != 403 || ae.Code != CodeForbidden {
		t.Fatalf("pharmacist create: %v", err)
	}
	if _, err := svc.SearchPatients(ctx, pc, PatientSearch{Name: "张"}); err != nil {
		t.Fatalf("pharmacist search: %v", err)
	}

	_, err = svc.CreatePatient(ctx, nil, validInput(), "req-anon", "127.0.0.1")
	if !errors.As(err, &ae) || ae.Status != 401 || ae.Code != CodeUnauthorized {
		t.Fatalf("anon: %v", err)
	}

	dis := &auth.Claims{}
	dis.Subject = seed.DisabledUserID
	_, err = svc.GetPatient(ctx, dis, dto.PatientID)
	if !errors.As(err, &ae) || ae.Code != CodeAccountDisabled {
		t.Fatalf("disabled: %v", err)
	}
}

func TestCreatePatientIdempotency(t *testing.T) {
	svc, _ := newPatientSvc(t)
	ctx := context.Background()
	in := validInput()
	in.CardNo = ""
	in.IdempotencyKey = "idem-patient-1"
	a, err := svc.CreatePatient(ctx, systemClaims(), in, "req-i1", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.CreatePatient(ctx, systemClaims(), in, "req-i2", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if a.PatientID != b.PatientID {
		t.Fatalf("replay created another patient %s %s", a.PatientID, b.PatientID)
	}
	changed := in
	changed.Name = "王五"
	_, err = svc.CreatePatient(ctx, systemClaims(), changed, "req-i3", "127.0.0.1")
	var ae *Error
	if !errors.As(err, &ae) || ae.Status != 409 || ae.Code != CodeIdempotencyConflict {
		t.Fatalf("conflict: %v", err)
	}
}
