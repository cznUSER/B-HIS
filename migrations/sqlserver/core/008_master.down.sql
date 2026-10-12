-- Rollback P1-H03-MASTER additions. Does not DROP DATABASE. No credentials in this file.
-- Order: drop new objects/columns only; does not touch 003/004/006 base tables' core columns.
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
GO

USE BHIS;
GO

-- dict_item（含 seed）
IF OBJECT_ID(N'dbo.dict_item', N'U') IS NOT NULL
BEGIN
    DROP TABLE dbo.dict_item;
END
GO

-- catalog_item.is_controlled
IF COL_LENGTH(N'dbo.catalog_item', N'is_controlled') IS NOT NULL
BEGIN
    ALTER TABLE dbo.catalog_item DROP CONSTRAINT df_catalog_item_is_controlled;
    ALTER TABLE dbo.catalog_item DROP COLUMN is_controlled;
END
GO

-- department.sort_no
IF COL_LENGTH(N'dbo.department', N'sort_no') IS NOT NULL
BEGIN
    ALTER TABLE dbo.department DROP CONSTRAINT df_department_sort_no;
    ALTER TABLE dbo.department DROP COLUMN sort_no;
END
GO
