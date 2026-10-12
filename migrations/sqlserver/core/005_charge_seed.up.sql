-- P1-H03-CHARGE seed placeholder.
-- No demo Charge / Payment rows: Phase1 charges are created from COMMIT orders at runtime.
-- Do not invent fake payments. Depends on 005_charge.up.sql (+ 004_order for catalog/orders).
-- This file is intentionally a no-op comment seed so migrate tooling can list 005_charge_seed.
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
GO

USE BHIS;
GO

-- (no INSERT / MERGE — empty seed by design)
GO
