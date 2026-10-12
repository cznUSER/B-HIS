-- Drops BHIS order_item / order_sheet / catalog_item. Does not DROP DATABASE. No credentials in this file.
-- Order: order_item → order_sheet → catalog_item (FK dependents first).
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
GO

USE BHIS;
GO

DROP TABLE IF EXISTS dbo.order_item;
GO

DROP TABLE IF EXISTS dbo.order_sheet;
GO

DROP TABLE IF EXISTS dbo.catalog_item;
GO
