package memory

import (
	"context"
	"testing"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

func TestPatientSearchMatchesPatientNoExactly(t *testing.T) {
	store := New()
	patients := []domain.Patient{
		{PatientID: "patient-1", HospPkNo: "hosp-1", PatientNo: "P-20261014-001", CreatedAt: time.Date(2026, 10, 14, 0, 0, 1, 0, time.UTC)},
		{PatientID: "patient-2", HospPkNo: "hosp-2", PatientNo: "P-20261014-001X", CreatedAt: time.Date(2026, 10, 14, 0, 0, 2, 0, time.UTC)},
	}
	for _, p := range patients {
		if err := store.Create(context.Background(), p); err != nil {
			t.Fatal(err)
		}
	}

	query := port.PatientQuery{PatientNo: "P-20261014-001"}

	got, err := store.Search(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].PatientNo != "P-20261014-001" {
		t.Fatalf("exact patient number search = %+v", got)
	}
}
