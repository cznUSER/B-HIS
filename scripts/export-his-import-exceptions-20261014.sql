/* Read-only exception export for the HIS -> BHIS import.
   Output columns are source_table,source_id,reason.
   This file intentionally contains SELECT/CTE statements only. */
USE HIS;
SET NOCOUNT ON;
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;

WITH valid_patient AS (
    SELECT LTRIM(RTRIM(CONVERT(varchar(64), p.id))) AS id
    FROM dbo.patient p
    WHERE p.birth_date IS NOT NULL
      AND p.birth_date > CONVERT(date, '19000101')
      AND p.birth_date <= CONVERT(date, SYSDATETIME())
), valid_registration AS (
    SELECT LTRIM(RTRIM(CONVERT(varchar(64), r.id))) AS id
    FROM dbo.registration r
    JOIN valid_patient p ON p.id = LTRIM(RTRIM(CONVERT(varchar(64), r.patient_id)))
    JOIN dbo.department sd ON LTRIM(RTRIM(CONVERT(varchar(64), sd.id))) = LTRIM(RTRIM(CONVERT(varchar(64), r.department_id)))
    JOIN dbo.staff ss ON LTRIM(RTRIM(CONVERT(varchar(64), ss.id))) = LTRIM(RTRIM(CONVERT(varchar(64), r.doctor_id)))
    WHERE ss.department_id IS NOT NULL
      AND LTRIM(RTRIM(CONVERT(varchar(64), ss.department_id))) = LTRIM(RTRIM(CONVERT(varchar(64), r.department_id)))
), valid_prescription AS (
    SELECT LTRIM(RTRIM(CONVERT(varchar(64), pr.id))) AS id
    FROM dbo.prescription pr
    JOIN dbo.visit v ON LTRIM(RTRIM(CONVERT(varchar(64), v.id))) = LTRIM(RTRIM(CONVERT(varchar(64), pr.visit_id)))
    JOIN valid_registration vr ON vr.id = LTRIM(RTRIM(CONVERT(varchar(64), v.registration_id)))
), valid_charge AS (
    SELECT LTRIM(RTRIM(CONVERT(varchar(64), c.id))) AS id
    FROM dbo.charge c
    JOIN valid_patient vp ON vp.id = LTRIM(RTRIM(CONVERT(varchar(64), c.patient_id)))
    OUTER APPLY (
        SELECT TOP (1) vr.id
        FROM valid_registration vr
        WHERE vr.id = NULLIF(LTRIM(RTRIM(CONVERT(varchar(64), c.registration_id))), '')
          AND EXISTS (
              SELECT 1
              FROM dbo.registration r
              WHERE LTRIM(RTRIM(CONVERT(varchar(64), r.id))) = vr.id
                AND LTRIM(RTRIM(CONVERT(varchar(64), r.patient_id))) = vp.id
          )
    ) direct_reg
    OUTER APPLY (
        SELECT TOP (1) pr.id
        FROM valid_prescription pr
        WHERE pr.id = NULLIF(LTRIM(RTRIM(CONVERT(varchar(64), c.prescription_id))), '')
          AND EXISTS (
              SELECT 1
              FROM dbo.prescription source_pr
              JOIN dbo.visit source_visit
                ON LTRIM(RTRIM(CONVERT(varchar(64), source_visit.id))) = LTRIM(RTRIM(CONVERT(varchar(64), source_pr.visit_id)))
              JOIN dbo.registration source_reg
                ON LTRIM(RTRIM(CONVERT(varchar(64), source_reg.id))) = LTRIM(RTRIM(CONVERT(varchar(64), source_visit.registration_id)))
              WHERE LTRIM(RTRIM(CONVERT(varchar(64), source_pr.id))) = pr.id
                AND LTRIM(RTRIM(CONVERT(varchar(64), source_reg.patient_id))) = vp.id
          )
    ) rx_reg
    WHERE COALESCE(direct_reg.id, rx_reg.id) IS NOT NULL
), valid_dispense AS (
    SELECT LTRIM(RTRIM(CONVERT(varchar(64), d.id))) AS id
    FROM dbo.dispense d
    JOIN valid_prescription vp ON vp.id = LTRIM(RTRIM(CONVERT(varchar(64), d.prescription_id)))
    WHERE EXISTS (
        SELECT 1
        FROM dbo.dispense_item di
        JOIN dbo.prescription_item pi ON LTRIM(RTRIM(CONVERT(varchar(64), pi.id))) = LTRIM(RTRIM(CONVERT(varchar(64), di.prescription_item_id)))
        WHERE LTRIM(RTRIM(CONVERT(varchar(64), di.dispense_id))) = LTRIM(RTRIM(CONVERT(varchar(64), d.id)))
          AND LTRIM(RTRIM(CONVERT(varchar(64), pi.prescription_id))) = LTRIM(RTRIM(CONVERT(varchar(64), d.prescription_id)))
          AND ISNULL(di.quantity, 0) > 0
    )
)
SELECT 'patient', LTRIM(RTRIM(CONVERT(varchar(64), p.id))), 'MISSING_BIRTH_DATE'
FROM dbo.patient p
WHERE NOT EXISTS (SELECT 1 FROM valid_patient v WHERE v.id = LTRIM(RTRIM(CONVERT(varchar(64), p.id))))
UNION ALL
SELECT 'registration', LTRIM(RTRIM(CONVERT(varchar(64), r.id))),
       CASE WHEN EXISTS (SELECT 1 FROM valid_patient p WHERE p.id = LTRIM(RTRIM(CONVERT(varchar(64), r.patient_id))))
            THEN 'DEPARTMENT_OR_DOCTOR_EXCLUDED' ELSE 'PATIENT_EXCLUDED' END
FROM dbo.registration r
WHERE NOT EXISTS (SELECT 1 FROM valid_registration v WHERE v.id = LTRIM(RTRIM(CONVERT(varchar(64), r.id))))
UNION ALL
SELECT 'prescription', LTRIM(RTRIM(CONVERT(varchar(64), pr.id))), 'PATIENT_CHAIN_EXCLUDED'
FROM dbo.prescription pr
WHERE NOT EXISTS (SELECT 1 FROM valid_prescription v WHERE v.id = LTRIM(RTRIM(CONVERT(varchar(64), pr.id))))
UNION ALL
SELECT 'charge', LTRIM(RTRIM(CONVERT(varchar(64), c.id))), 'NO_VALID_REGISTER'
FROM dbo.charge c
WHERE NOT EXISTS (SELECT 1 FROM valid_charge v WHERE v.id = LTRIM(RTRIM(CONVERT(varchar(64), c.id))))
UNION ALL
SELECT 'payment', LTRIM(RTRIM(CONVERT(varchar(64), p.id))),
       CASE WHEN NOT EXISTS (SELECT 1 FROM valid_charge c WHERE c.id = LTRIM(RTRIM(CONVERT(varchar(64), p.charge_id)))) THEN 'NO_CHARGE'
            WHEN LOWER(LTRIM(RTRIM(ISNULL(p.method, '')))) <> 'cash' THEN 'NON_CASH_PAYMENT'
            WHEN ISNULL(p.amount_cents, 0) = 0 THEN 'ZERO_VALUE_PAYMENT'
            ELSE 'NOT_IMPORTABLE' END
FROM dbo.payment p
WHERE NOT EXISTS (SELECT 1 FROM valid_charge c WHERE c.id = LTRIM(RTRIM(CONVERT(varchar(64), p.charge_id))))
   OR LOWER(LTRIM(RTRIM(ISNULL(p.method, '')))) <> 'cash'
   OR ISNULL(p.amount_cents, 0) = 0
UNION ALL
SELECT 'dispense', LTRIM(RTRIM(CONVERT(varchar(64), d.id))), 'PATIENT_CHAIN_EXCLUDED'
FROM dbo.dispense d
WHERE NOT EXISTS (SELECT 1 FROM valid_dispense v WHERE v.id = LTRIM(RTRIM(CONVERT(varchar(64), d.id))))
UNION ALL
SELECT 'registration', LTRIM(RTRIM(CONVERT(varchar(64), r.id))), 'UNKNOWN_STATUS'
FROM dbo.registration r
WHERE LOWER(LTRIM(RTRIM(ISNULL(r.status, '')))) NOT IN ('waiting', 'in_visit', 'finished', 'cancelled')
UNION ALL
SELECT 'prescription', LTRIM(RTRIM(CONVERT(varchar(64), pr.id))), 'UNKNOWN_STATUS'
FROM dbo.prescription pr
WHERE LOWER(LTRIM(RTRIM(ISNULL(pr.status, '')))) NOT IN ('charged', 'executed')
UNION ALL
SELECT 'charge', LTRIM(RTRIM(CONVERT(varchar(64), c.id))), 'UNKNOWN_STATUS'
FROM dbo.charge c
WHERE LOWER(LTRIM(RTRIM(ISNULL(c.status, '')))) NOT IN ('paid', 'refunded')
UNION ALL
SELECT 'dispense', LTRIM(RTRIM(CONVERT(varchar(64), d.id))), 'UNKNOWN_STATUS'
FROM dbo.dispense d
WHERE LOWER(LTRIM(RTRIM(ISNULL(d.status, '')))) NOT IN ('dispensed', 'returned')
ORDER BY 1, 2, 3;

SELECT 'patient', LTRIM(RTRIM(CONVERT(varchar(64), p.id))), 'FIELD_TOO_LONG:name'
FROM dbo.patient p WHERE LEN(p.name) > 64
UNION ALL SELECT 'patient', LTRIM(RTRIM(CONVERT(varchar(64), p.id))), 'FIELD_TOO_LONG:id_type' FROM dbo.patient p WHERE LEN(p.id_type) > 16
UNION ALL SELECT 'patient', LTRIM(RTRIM(CONVERT(varchar(64), p.id))), 'FIELD_TOO_LONG:id_no' FROM dbo.patient p WHERE LEN(p.id_no) > 32
UNION ALL SELECT 'patient', LTRIM(RTRIM(CONVERT(varchar(64), p.id))), 'FIELD_TOO_LONG:phone' FROM dbo.patient p WHERE LEN(p.phone) > 32
UNION ALL SELECT 'patient', LTRIM(RTRIM(CONVERT(varchar(64), p.id))), 'FIELD_TOO_LONG:address' FROM dbo.patient p WHERE LEN(p.address) > 256
UNION ALL SELECT 'patient', LTRIM(RTRIM(CONVERT(varchar(64), p.id))), 'FIELD_TOO_LONG:fee_category' FROM dbo.patient p WHERE LEN(p.fee_category) > 32
UNION ALL SELECT 'registration', LTRIM(RTRIM(CONVERT(varchar(64), r.id))), 'FIELD_TOO_LONG:reg_no' FROM dbo.registration r WHERE LEN(r.reg_no) > 32
UNION ALL SELECT 'prescription', LTRIM(RTRIM(CONVERT(varchar(64), pr.id))), 'FIELD_TOO_LONG:rx_no' FROM dbo.prescription pr WHERE LEN(pr.rx_no) > 32
UNION ALL SELECT 'charge', LTRIM(RTRIM(CONVERT(varchar(64), c.id))), 'FIELD_TOO_LONG:charge_no' FROM dbo.charge c WHERE LEN(c.charge_no) > 32
UNION ALL SELECT 'drug', LTRIM(RTRIM(CONVERT(varchar(64), d.id))), 'FIELD_TOO_LONG:code' FROM dbo.drug d WHERE LEN(d.code) > 32
UNION ALL SELECT 'drug', LTRIM(RTRIM(CONVERT(varchar(64), d.id))), 'FIELD_TOO_LONG:generic_name' FROM dbo.drug d WHERE LEN(d.generic_name) > 128
UNION ALL SELECT 'drug', LTRIM(RTRIM(CONVERT(varchar(64), d.id))), 'FIELD_TOO_LONG:spec' FROM dbo.drug d WHERE LEN(d.spec) > 64
UNION ALL SELECT 'drug', LTRIM(RTRIM(CONVERT(varchar(64), d.id))), 'FIELD_TOO_LONG:unit' FROM dbo.drug d WHERE LEN(d.unit) > 16
UNION ALL SELECT 'dispense', LTRIM(RTRIM(CONVERT(varchar(64), d.id))), 'FIELD_TOO_LONG:dispense_no' FROM dbo.dispense d WHERE LEN(d.dispense_no) > 32
UNION ALL SELECT 'prescription_item', LTRIM(RTRIM(CONVERT(varchar(64), pi.id))), 'FIELD_TOO_LONG:usage_text' FROM dbo.prescription_item pi WHERE LEN(pi.usage_text) > 64
UNION ALL SELECT 'prescription_item', LTRIM(RTRIM(CONVERT(varchar(64), pi.id))), 'FIELD_TOO_LONG:frequency' FROM dbo.prescription_item pi WHERE LEN(pi.frequency) > 64
UNION ALL SELECT 'charge_item', LTRIM(RTRIM(CONVERT(varchar(64), ci.id))), 'FIELD_TOO_LONG:generic_name' FROM dbo.charge_item ci WHERE LEN(ci.generic_name) > 128
ORDER BY 1, 2, 3;

IF OBJECT_ID(N'dbo.inventory_batch', N'U') IS NOT NULL
BEGIN
    SELECT 'inventory_batch', LTRIM(RTRIM(CONVERT(varchar(64), id))), 'BATCH_MODEL_DEFERRED'
    FROM dbo.inventory_batch
    ORDER BY id;
END;
