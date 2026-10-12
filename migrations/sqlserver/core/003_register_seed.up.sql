-- Demo seed: department + doctor (P1-H03-REGISTER / H02 挂号字段清单 §3).
-- IDs align with H02: dept-1, dept-2, doc-1, doc-2. No connection string.
-- Idempotent MERGE. Depends on 003_register.up.sql.
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
GO

USE BHIS;
GO

MERGE dbo.department AS t
USING (VALUES
    (N'dept-1', N'DEPT1', N'内科', N'ACTIVE'),
    (N'dept-2', N'DEPT2', N'外科', N'ACTIVE')
) AS s(department_id, dept_code, dept_name, status)
ON t.department_id = s.department_id
WHEN NOT MATCHED THEN
    INSERT (department_id, dept_code, dept_name, status, created_at, updated_at, is_deleted)
    VALUES (s.department_id, s.dept_code, s.dept_name, s.status, SYSUTCDATETIME(), SYSUTCDATETIME(), 0)
WHEN MATCHED AND t.is_deleted = 0 THEN
    UPDATE SET t.dept_code = s.dept_code, t.dept_name = s.dept_name, t.status = s.status,
               t.updated_at = SYSUTCDATETIME();
GO

MERGE dbo.doctor AS t
USING (VALUES
    (N'doc-1', N'dept-1', N'DOC1', N'张医生', N'ACTIVE'),
    (N'doc-2', N'dept-2', N'DOC2', N'李医生', N'ACTIVE')
) AS s(doctor_id, department_id, doctor_code, doctor_name, status)
ON t.doctor_id = s.doctor_id
WHEN NOT MATCHED THEN
    INSERT (doctor_id, department_id, doctor_code, doctor_name, status, created_at, updated_at, is_deleted)
    VALUES (s.doctor_id, s.department_id, s.doctor_code, s.doctor_name, s.status, SYSUTCDATETIME(), SYSUTCDATETIME(), 0)
WHEN MATCHED AND t.is_deleted = 0 THEN
    UPDATE SET t.department_id = s.department_id, t.doctor_code = s.doctor_code,
               t.doctor_name = s.doctor_name, t.status = s.status, t.updated_at = SYSUTCDATETIME();
GO
