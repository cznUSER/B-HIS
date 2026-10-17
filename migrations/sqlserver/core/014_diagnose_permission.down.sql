SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
USE BHIS;
GO

DELETE FROM dbo.access_role_permission
WHERE permission_code = N'diagnose:write';

DELETE FROM dbo.access_permission
WHERE permission_code = N'diagnose:write';
GO
