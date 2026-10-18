package mssql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"strings"
	"testing"

	"github.com/cznUSER/B-HIS/internal/port"
)

func TestPatientSearchUsesExactPatientNoSQLParameter(t *testing.T) {
	capture := &patientSearchCapture{}
	db := sql.OpenDB(patientSearchConnector{capture: capture})
	t.Cleanup(func() { _ = db.Close() })
	store := New(db)

	query := port.PatientQuery{PatientNo: "P-20261014-001", Offset: 23}
	if _, err := store.Search(context.Background(), query); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(capture.query, "patient_no = @patient_no") {
		t.Fatalf("patient search SQL has no exact patient_no predicate: %s", capture.query)
	}
	if !strings.Contains(capture.query, "OFFSET @offset ROWS FETCH NEXT @limit ROWS ONLY") {
		t.Fatalf("patient search SQL has no offset pagination: %s", capture.query)
	}
	var sawOffset bool
	for _, arg := range capture.args {
		if arg.Name == "offset" && arg.Value == int64(23) {
			sawOffset = true
		}
		if arg.Name == "patient_no" {
			if arg.Value != "P-20261014-001" {
				t.Fatalf("patient_no SQL argument = %#v", arg.Value)
			}
		}
	}
	if !sawOffset {
		t.Fatalf("patient offset SQL argument missing: %+v", capture.args)
	}
}

type patientSearchCapture struct {
	query string
	args  []driver.NamedValue
}

type patientSearchConnector struct{ capture *patientSearchCapture }

func (c patientSearchConnector) Connect(context.Context) (driver.Conn, error) {
	return patientSearchConn{capture: c.capture}, nil
}
func (patientSearchConnector) Driver() driver.Driver { return patientSearchDriver{} }

type patientSearchDriver struct{}

func (patientSearchDriver) Open(string) (driver.Conn, error) { return nil, driver.ErrSkip }

type patientSearchConn struct{ capture *patientSearchCapture }

func (patientSearchConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (patientSearchConn) Close() error                        { return nil }
func (patientSearchConn) Begin() (driver.Tx, error)           { return nil, driver.ErrSkip }
func (c patientSearchConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.capture.query = query
	c.capture.args = append([]driver.NamedValue(nil), args...)
	return &emptyPatientSearchRows{}, nil
}

type emptyPatientSearchRows struct{}

func (*emptyPatientSearchRows) Columns() []string         { return nil }
func (*emptyPatientSearchRows) Close() error              { return nil }
func (*emptyPatientSearchRows) Next([]driver.Value) error { return io.EOF }
