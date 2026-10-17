-- P2-H01-DIAGNOSE: isolate clinical diagnosis write access from encounter check-in.
-- Safe for an existing BHIS database and idempotent for local re-runs.
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
USE BHIS;
GO

IF NOT EXISTS (
    SELECT 1 FROM dbo.access_permission WHERE permission_code = N'diagnose:write'
)
BEGIN
    INSERT INTO dbo.access_permission (
        permission_code, permission_name, description, module_code, category_id,
        mode_code, second_permission_code, actions, status, hidden
    ) VALUES (
        N'diagnose:write', N'诊断录入', N'保存、提交和作废门诊诊断', N'outpatient', N'patient',
        N'password', NULL, N'N,M,D,S', N'ACTIVE', 0
    );
END
GO

IF NOT EXISTS (
    SELECT 1 FROM dbo.access_role_permission
    WHERE role_code = N'DOCTOR' AND permission_code = N'diagnose:write'
)
BEGIN
    INSERT INTO dbo.access_role_permission(role_code, permission_code)
    VALUES (N'DOCTOR', N'diagnose:write');
END
GO
