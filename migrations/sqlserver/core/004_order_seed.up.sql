-- Demo seed: catalog_item DRUG-A / DRUG-B / TREAT-1 (P1-H03-ORDER / H02 开单字段清单 §4).
-- IDs: cat-drug-a, cat-drug-b, cat-treat-1. No passwords. No connection string.
-- Idempotent MERGE. Depends on 004_order.up.sql.
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
GO

USE BHIS;
GO

MERGE dbo.catalog_item AS t
USING (VALUES
    (N'cat-drug-a',  N'DRUG-A',  N'演示阿莫西林胶囊', N'DRUG',  N'0.25g×24', N'盒', CAST(10.00 AS decimal(18,2)), N'ACTIVE'),
    (N'cat-drug-b',  N'DRUG-B',  N'演示布洛芬片',     N'DRUG',  N'0.2g×20',  N'盒', CAST(8.00 AS decimal(18,2)),  N'ACTIVE'),
    (N'cat-treat-1', N'TREAT-1', N'演示诊查费',       N'TREAT', NULL,         N'次', CAST(15.00 AS decimal(18,2)), N'ACTIVE')
) AS s(catalog_item_id, item_no, item_name, item_type, spec, units, price, status)
ON t.catalog_item_id = s.catalog_item_id
WHEN NOT MATCHED THEN
    INSERT (catalog_item_id, item_no, item_name, item_type, spec, units, price, status, created_at, updated_at, is_deleted)
    VALUES (s.catalog_item_id, s.item_no, s.item_name, s.item_type, s.spec, s.units, s.price, s.status,
            SYSUTCDATETIME(), SYSUTCDATETIME(), 0)
WHEN MATCHED AND t.is_deleted = 0 THEN
    UPDATE SET t.item_no = s.item_no, t.item_name = s.item_name, t.item_type = s.item_type,
               t.spec = s.spec, t.units = s.units, t.price = s.price, t.status = s.status,
               t.updated_at = SYSUTCDATETIME();
GO
