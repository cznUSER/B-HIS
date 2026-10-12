-- Drops BHIS register / doctor / department tables. Does not DROP DATABASE. No credentials in this file.
-- Order: register_bill → doctor → department (FK dependents first).
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
GO

USE BHIS;
GO

DROP TABLE IF EXISTS dbo.register_bill;
GO

DROP TABLE IF EXISTS dbo.doctor;
GO

DROP TABLE IF EXISTS dbo.department;
GO
