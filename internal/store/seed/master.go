package seed

import "github.com/cznUSER/B-HIS/internal/domain"

// Departments and Doctors match migrations/sqlserver/core/003_register_seed.up.sql.
// 内科/外科、张医生/李医生。内存库与 SQL 种子使用同一组 id。2026-10-12

func Departments() []domain.Department {
	return []domain.Department{
		{DepartmentID: "dept-1", DeptCode: "DEPT1", DeptName: "内科", Status: "ACTIVE"},
		{DepartmentID: "dept-2", DeptCode: "DEPT2", DeptName: "外科", Status: "ACTIVE"},
	}
}

func Doctors() []domain.Doctor {
	return []domain.Doctor{
		{DoctorID: "doc-1", DepartmentID: "dept-1", DoctorCode: "DOC1", DoctorName: "张医生", Status: "ACTIVE"},
		{DoctorID: "doc-2", DepartmentID: "dept-2", DoctorCode: "DOC2", DoctorName: "李医生", Status: "ACTIVE"},
	}
}
