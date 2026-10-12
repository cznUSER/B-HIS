-- Drops BHIS dispense_item / dispense. Does not DROP DATABASE. No credentials in this file.
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
GO

USE BHIS;
GO

DROP TABLE IF EXISTS dbo.dispense_item;
GO

DROP TABLE IF EXISTS dbo.dispense;
GO
