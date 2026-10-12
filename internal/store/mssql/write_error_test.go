package mssql

import (
	"errors"
	"fmt"
	"testing"

	mssql "github.com/microsoft/go-mssqldb"

	"github.com/cznUSER/B-HIS/internal/port"
)

func TestClassifyWriteError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "id number",
			err:  mssql.Error{Number: 2601, Message: "Cannot insert duplicate key row in object 'dbo.patient' with unique index 'ux_patient_id_number'."},
			want: port.ErrDuplicateIDNumber,
		},
		{
			name: "card",
			err:  mssql.Error{Number: 2627, Message: "Violation of UNIQUE KEY constraint. unique index 'ux_patient_card_no'."},
			want: port.ErrDuplicateCardNo,
		},
		{
			name: "hosp",
			err:  fmt.Errorf("mssql: duplicate key ux_patient_hosp_pk"),
			want: port.ErrDuplicateBusinessNo,
		},
		{
			name: "register no",
			err:  mssql.Error{Number: 2601, Message: "Cannot insert duplicate key row in object 'dbo.register_bill' with unique index 'ux_register_no'."},
			want: port.ErrDuplicateBusinessNo,
		},
		{
			name: "other",
			err:  errors.New("connection reset"),
			want: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyWriteError(tc.err)
			if tc.want == nil {
				if got != tc.err {
					t.Fatalf("got %v", got)
				}
				return
			}
			if !errors.Is(got, tc.want) {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestDecimalCentsRoundTrip(t *testing.T) {
	for _, cents := range []int64{0, 1, 12, 1250, 100} {
		got, err := decimalToCents(centsToDecimal(cents))
		if err != nil || got != cents {
			t.Fatalf("%d -> %s -> %d %v", cents, centsToDecimal(cents), got, err)
		}
	}
}

func TestLikeContainsEscapes(t *testing.T) {
	got := likeContains(`100%_[`)
	if got != `%100\%\_\[%` {
		t.Fatalf("got %q", got)
	}
}
