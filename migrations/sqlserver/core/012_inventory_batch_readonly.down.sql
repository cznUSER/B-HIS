-- P2-6A rollback is restricted to the isolated test database.
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
GO

IF DB_NAME() IN (N'BHIS', N'HospDataV9')
    THROW 51026, 'P2-6A rollback is restricted to an isolated test database', 1;
GO

IF OBJECT_ID(N'dbo.access_role_permission', N'U') IS NOT NULL
    DELETE FROM dbo.access_role_permission WHERE permission_code = N'inventory:batch-read';
IF OBJECT_ID(N'dbo.access_permission', N'U') IS NOT NULL
    DELETE FROM dbo.access_permission WHERE permission_code = N'inventory:batch-read';
IF OBJECT_ID(N'dbo.inventory_batch', N'U') IS NOT NULL
    DROP TABLE dbo.inventory_batch;
GO
