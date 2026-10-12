-- Drops BHIS inventory_ledger / inventory_item. Does not DROP DATABASE. No credentials in this file.
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
GO

USE BHIS;
GO

DROP TABLE IF EXISTS dbo.inventory_ledger;
GO

DROP TABLE IF EXISTS dbo.inventory_item;
GO
