package app

import (
	"context"
	"testing"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

func TestSearchPatientsSupportsExactPatientNo(t *testing.T) {
	svc, store := newPatientSvc(t)
	spy := &patientSearchSpy{PatientStore: store}
	svc.Patients = spy
	ctx := context.Background()
	created, err := svc.CreatePatient(ctx, systemClaims(), validInput(), "req-pno-search", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}

	empty, err := svc.SearchPatients(ctx, systemClaims(), PatientSearch{PatientNo: " \t "})
	if err != nil || len(empty.Items) != 0 {
		t.Fatalf("blank patient number search = %+v, %v", empty, err)
	}
	if spy.calls != 0 {
		t.Fatalf("blank search reached patient store %d times", spy.calls)
	}

	query := PatientSearch{PatientNo: " " + created.PatientNo + " "}

	list, err := svc.SearchPatients(ctx, systemClaims(), query)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].PatientID != created.PatientID {
		t.Fatalf("exact patient number search = %+v, want patient %s", list.Items, created.PatientID)
	}

	query.PatientNo = created.PatientNo[:len(created.PatientNo)-1]
	list, err = svc.SearchPatients(ctx, systemClaims(), query)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 0 {
		t.Fatalf("partial patient number matched: %+v", list.Items)
	}
}

type patientSearchSpy struct {
	port.PatientStore
	calls int
}

func (s *patientSearchSpy) Search(ctx context.Context, query port.PatientQuery) ([]domain.Patient, error) {
	s.calls++
	return s.PatientStore.Search(ctx, query)
}
