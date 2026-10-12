-- Drops BHIS payment / charge_item / charge. Does not DROP DATABASE. No credentials in this file.
-- Order: payment → charge_item → charge (FK dependents first; charge self-FK cleared by DROP TABLE).
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
GO

USE BHIS;
GO

DROP TABLE IF EXISTS dbo.payment;
GO

DROP TABLE IF EXISTS dbo.charge_item;
GO

DROP TABLE IF EXISTS dbo.charge;
GO
