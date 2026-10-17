-- Extends patient archive fields used by detailed registration.
-- Apply after 015_local_inventory_batch_readonly in database BHIS.
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
GO

USE BHIS;
GO

IF COL_LENGTH(N'dbo.patient', N'member_card_no') IS NULL ALTER TABLE dbo.patient ADD member_card_no varchar(32) NULL;
IF COL_LENGTH(N'dbo.patient', N'member_level') IS NULL ALTER TABLE dbo.patient ADD member_level nvarchar(32) NULL;
IF COL_LENGTH(N'dbo.patient', N'contract_unit') IS NULL ALTER TABLE dbo.patient ADD contract_unit nvarchar(128) NULL;
IF COL_LENGTH(N'dbo.patient', N'patient_certificate_no') IS NULL ALTER TABLE dbo.patient ADD patient_certificate_no varchar(32) NULL;
IF COL_LENGTH(N'dbo.patient', N'birth_place') IS NULL ALTER TABLE dbo.patient ADD birth_place nvarchar(128) NULL;
IF COL_LENGTH(N'dbo.patient', N'ethnicity') IS NULL ALTER TABLE dbo.patient ADD ethnicity nvarchar(32) NULL;
IF COL_LENGTH(N'dbo.patient', N'nationality') IS NULL ALTER TABLE dbo.patient ADD nationality nvarchar(32) NULL;
IF COL_LENGTH(N'dbo.patient', N'marital_status') IS NULL ALTER TABLE dbo.patient ADD marital_status nvarchar(32) NULL;
IF COL_LENGTH(N'dbo.patient', N'occupation_category') IS NULL ALTER TABLE dbo.patient ADD occupation_category nvarchar(64) NULL;
IF COL_LENGTH(N'dbo.patient', N'occupation') IS NULL ALTER TABLE dbo.patient ADD occupation nvarchar(128) NULL;
IF COL_LENGTH(N'dbo.patient', N'education') IS NULL ALTER TABLE dbo.patient ADD education nvarchar(32) NULL;
IF COL_LENGTH(N'dbo.patient', N'work_unit') IS NULL ALTER TABLE dbo.patient ADD work_unit nvarchar(128) NULL;
IF COL_LENGTH(N'dbo.patient', N'work_unit_phone') IS NULL ALTER TABLE dbo.patient ADD work_unit_phone varchar(32) NULL;
IF COL_LENGTH(N'dbo.patient', N'work_unit_postcode') IS NULL ALTER TABLE dbo.patient ADD work_unit_postcode varchar(16) NULL;
IF COL_LENGTH(N'dbo.patient', N'household_address') IS NULL ALTER TABLE dbo.patient ADD household_address nvarchar(256) NULL;
IF COL_LENGTH(N'dbo.patient', N'household_phone') IS NULL ALTER TABLE dbo.patient ADD household_phone varchar(32) NULL;
IF COL_LENGTH(N'dbo.patient', N'household_postcode') IS NULL ALTER TABLE dbo.patient ADD household_postcode varchar(16) NULL;
IF COL_LENGTH(N'dbo.patient', N'contact_name') IS NULL ALTER TABLE dbo.patient ADD contact_name nvarchar(64) NULL;
IF COL_LENGTH(N'dbo.patient', N'contact_address') IS NULL ALTER TABLE dbo.patient ADD contact_address nvarchar(256) NULL;
IF COL_LENGTH(N'dbo.patient', N'contact_phone') IS NULL ALTER TABLE dbo.patient ADD contact_phone varchar(32) NULL;
IF COL_LENGTH(N'dbo.patient', N'contact_relation') IS NULL ALTER TABLE dbo.patient ADD contact_relation nvarchar(32) NULL;
GO
