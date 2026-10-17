-- Removes the detailed patient registration fields added by 016_patient_registration_detail.up.sql.
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
GO

USE BHIS;
GO

IF COL_LENGTH(N'dbo.patient', N'member_card_no') IS NOT NULL
BEGIN
    ALTER TABLE dbo.patient DROP COLUMN
        member_card_no, member_level, contract_unit, patient_certificate_no, birth_place,
        ethnicity, nationality, marital_status, occupation_category, occupation, education,
        work_unit, work_unit_phone, work_unit_postcode, household_address, household_phone,
        household_postcode, contact_name,
        contact_address, contact_phone, contact_relation;
END
GO
