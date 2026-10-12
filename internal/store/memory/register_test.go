package memory

import (
	"os"
	"strings"
	"testing"
)

func TestRegisterMasterSeedMatchesSQL(t *testing.T) {
	st := New()
	raw, err := os.ReadFile("../../../migrations/sqlserver/core/003_register_seed.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sqlText := string(raw)
	depts := map[string]string{"dept-1": "内科", "dept-2": "外科"}
	for id, name := range depts {
		d, ok := st.departments[id]
		if !ok || d.DeptName != name || d.DeptCode == "" || d.Status != "ACTIVE" || d.IsDeleted {
			t.Fatalf("%s %+v", id, d)
		}
		if !strings.Contains(sqlText, id) || !strings.Contains(sqlText, name) {
			t.Fatalf("sql seed missing %s %s", id, name)
		}
	}
	docs := map[string]struct{ dept, name string }{
		"doc-1": {"dept-1", "张医生"},
		"doc-2": {"dept-2", "李医生"},
	}
	for id, want := range docs {
		d, ok := st.doctors[id]
		if !ok || d.DoctorName != want.name || d.DepartmentID != want.dept || d.Status != "ACTIVE" || d.IsDeleted {
			t.Fatalf("%s %+v", id, d)
		}
		if !strings.Contains(sqlText, id) || !strings.Contains(sqlText, want.name) {
			t.Fatalf("sql seed missing %s %s", id, want.name)
		}
	}
}
