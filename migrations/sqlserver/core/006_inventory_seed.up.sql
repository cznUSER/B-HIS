-- Demo seed: inventory for DRUG-A / DRUG-B only (P1-H03-DISPENSE / 拍板#23).
-- TREAT-1 不入库存表（治疗费不发药）。qty_on_hand=100 演示可用量。
-- IDs: sku-drug-a / sku-drug-b. No passwords. No connection string.
-- Idempotent MERGE. Depends on 006_inventory.up.sql (+ 004_order_seed for catalog names).
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
GO

USE BHIS;
GO

MERGE dbo.inventory_item AS t
USING (VALUES
    (N'sku-drug-a', N'DRUG-A', N'演示阿莫西林胶囊', CAST(100.000 AS decimal(18,3)), N'盒', N'ACTIVE'),
    (N'sku-drug-b', N'DRUG-B', N'演示布洛芬片',     CAST(100.000 AS decimal(18,3)), N'盒', N'ACTIVE')
) AS s(sku_id, item_no, item_name, qty_on_hand, units, status)
ON t.sku_id = s.sku_id
WHEN NOT MATCHED THEN
    INSERT (sku_id, item_no, item_name, qty_on_hand, units, status, created_at, updated_at, is_deleted)
    VALUES (s.sku_id, s.item_no, s.item_name, s.qty_on_hand, s.units, s.status,
            SYSUTCDATETIME(), SYSUTCDATETIME(), 0)
WHEN MATCHED AND t.is_deleted = 0 THEN
    UPDATE SET t.item_no = s.item_no, t.item_name = s.item_name, t.units = s.units,
               t.status = s.status, t.updated_at = SYSUTCDATETIME();
    -- 不强制重置 qty_on_hand，避免覆盖演示已扣减库存
GO
