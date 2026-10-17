/*
   HIS -> BHIS outpatient history import, 2026-10-14.

   Run a dry-run first:
     sqlcmd -S localhost -E -I -b -f 65001 -v COMMIT=0 -i scripts/import-his-to-bhis-20261014.sql

   Commit only after reviewing the dry-run counts:
     sqlcmd -S localhost -E -I -b -f 65001 -v COMMIT=1 -i scripts/import-his-to-bhis-20261014.sql

   The script imports only source rows that satisfy the current BHIS constraints.
   Missing birthdays, unresolvable charges/payments, and inventory batches remain
   in deliverables/h01/his-history-import-exceptions-20261014.csv.
*/
:setvar IMPORT_BATCH "HIS_IMPORT_20261014"

SET NOCOUNT ON;
SET XACT_ABORT ON;
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;

USE BHIS;

DECLARE @batch varchar(64) = '$(IMPORT_BATCH)';
DECLARE @now datetime2 = SYSUTCDATETIME();

IF DB_ID(N'BHIS') IS NULL OR DB_ID(N'HIS') IS NULL
    THROW 51000, 'Both BHIS and HIS databases are required.', 1;

BEGIN TRANSACTION;

/* The source hospital has one tenant in this database. All IDs are preserved. */
SELECT
    id = LTRIM(RTRIM(CONVERT(varchar(64), p.id))),
    mrn = LTRIM(RTRIM(CONVERT(varchar(32), p.mrn))),
    name = LEFT(p.name, 64),
    sex = CASE p.gender WHEN 'male' THEN 'M' WHEN 'female' THEN 'F' ELSE 'U' END,
    birthday = p.birth_date,
    id_type = NULLIF(CASE WHEN p.id_type = 'none' THEN '' ELSE LEFT(p.id_type, 16) END, ''),
    id_number = LEFT(p.id_no, 32),
    phone = LEFT(p.phone, 32),
    address = LEFT(p.address, 256),
    fare_identity = LEFT(p.fee_category, 32),
    status = CASE WHEN p.cancelled_at IS NULL THEN 'ACTIVE' ELSE 'INACTIVE' END,
    created_at = p.created_at,
    updated_at = p.updated_at
INTO #valid_patient
FROM HIS.dbo.patient p
WHERE p.birth_date IS NOT NULL
  AND p.birth_date > CONVERT(date, '19000101')
  AND p.birth_date <= CONVERT(date, SYSDATETIME());

SELECT
    id = LTRIM(RTRIM(CONVERT(varchar(64), r.id))),
    register_no = LEFT(r.reg_no, 32),
    patient_id = LTRIM(RTRIM(CONVERT(varchar(64), r.patient_id))),
    department_id = LTRIM(RTRIM(CONVERT(varchar(64), r.department_id))),
    doctor_id = LTRIM(RTRIM(CONVERT(varchar(64), r.doctor_id))),
    visit_date = CONVERT(date, r.created_at),
    status = CASE r.status
        WHEN 'waiting' THEN 'NORMAL'
        WHEN 'in_visit' THEN 'DIAGNOSING'
        WHEN 'finished' THEN 'DIAGNOSED'
        WHEN 'cancelled' THEN 'BACK'
        ELSE 'NORMAL' END,
    register_fee = CONVERT(decimal(18,2), ISNULL(r.fee_cents, 0)) / 100,
    register_charged = CONVERT(bit, CASE WHEN ISNULL(r.fee_cents, 0) > 0 AND ISNULL(r.paid_cents, 0) >= r.fee_cents THEN 1 ELSE 0 END),
    registered_at = r.created_at,
    created_at = r.created_at,
    updated_at = r.updated_at
INTO #valid_registration
FROM HIS.dbo.registration r
JOIN #valid_patient p ON p.id = LTRIM(RTRIM(CONVERT(varchar(64), r.patient_id)))
JOIN HIS.dbo.department sd ON LTRIM(RTRIM(CONVERT(varchar(64), sd.id))) = LTRIM(RTRIM(CONVERT(varchar(64), r.department_id)))
JOIN HIS.dbo.staff ss ON LTRIM(RTRIM(CONVERT(varchar(64), ss.id))) = LTRIM(RTRIM(CONVERT(varchar(64), r.doctor_id)))
WHERE ss.department_id IS NOT NULL
  AND LTRIM(RTRIM(CONVERT(varchar(64), ss.department_id))) = LTRIM(RTRIM(CONVERT(varchar(64), r.department_id)));

SELECT
    id = LTRIM(RTRIM(CONVERT(varchar(64), pr.id))),
    rx_no = LEFT(pr.rx_no, 32),
    visit_id = LTRIM(RTRIM(CONVERT(varchar(64), v.id))),
    registration_id = LTRIM(RTRIM(CONVERT(varchar(64), v.registration_id))),
    patient_id = vr.patient_id,
    department_id = vr.department_id,
    doctor_id = vr.doctor_id,
    status = CASE pr.status WHEN 'executed' THEN 'PROC' ELSE 'COMMIT' END,
    charged = CONVERT(bit, CASE WHEN pr.status IN ('charged', 'executed') THEN 1 ELSE 0 END),
    total_fee = CONVERT(decimal(18,2), ISNULL(pr.amount_cents, 0)) / 100,
    submitted_at = pr.created_at,
    created_at = pr.created_at,
    updated_at = pr.updated_at
INTO #valid_prescription
FROM HIS.dbo.prescription pr
JOIN HIS.dbo.visit v ON LTRIM(RTRIM(CONVERT(varchar(64), v.id))) = LTRIM(RTRIM(CONVERT(varchar(64), pr.visit_id)))
JOIN #valid_registration vr ON vr.id = LTRIM(RTRIM(CONVERT(varchar(64), v.registration_id)));

SELECT
    id = LTRIM(RTRIM(CONVERT(varchar(64), c.id))),
    square_no = LEFT(c.charge_no, 32),
    patient_id = LTRIM(RTRIM(CONVERT(varchar(64), c.patient_id))),
    prescription_id = NULLIF(LTRIM(RTRIM(CONVERT(varchar(64), c.prescription_id))), ''),
    register_id = COALESCE(direct_reg.id, rx_reg.registration_id),
    status = CASE WHEN c.status = 'refunded' THEN 'ABATE' ELSE 'NORMAL' END,
    total_fee = CONVERT(decimal(18,2), ISNULL(c.amount_cents, 0)) / 100,
    created_at = c.created_at,
    updated_at = c.updated_at
INTO #valid_charge
FROM HIS.dbo.charge c
JOIN #valid_patient vp ON vp.id = LTRIM(RTRIM(CONVERT(varchar(64), c.patient_id)))
OUTER APPLY (
    SELECT TOP (1) vr.id, vr.patient_id
    FROM #valid_registration vr
    WHERE vr.id = NULLIF(LTRIM(RTRIM(CONVERT(varchar(64), c.registration_id))), '')
      AND vr.patient_id = vp.id
) direct_reg
OUTER APPLY (
    SELECT TOP (1) pr.registration_id
    FROM #valid_prescription pr
    WHERE pr.id = NULLIF(LTRIM(RTRIM(CONVERT(varchar(64), c.prescription_id))), '')
      AND pr.patient_id = vp.id
) rx_reg
WHERE COALESCE(direct_reg.id, rx_reg.registration_id) IS NOT NULL;

SELECT DISTINCT
    id = LTRIM(RTRIM(CONVERT(varchar(64), d.id))),
    item_no = LEFT(d.code, 32),
    item_name = LEFT(d.generic_name, 128),
    spec = LEFT(d.spec, 64),
    units = LEFT(d.unit, 16),
    price = CONVERT(decimal(18,2), ISNULL(d.price_cents, 0)) / 100,
    status = CASE WHEN d.cancelled_at IS NULL AND ISNULL(d.stopped, 0) = 0 THEN 'ACTIVE' ELSE 'INACTIVE' END,
    created_at = d.created_at,
    updated_at = d.updated_at
INTO #valid_drug
FROM HIS.dbo.drug d
WHERE EXISTS (
    SELECT 1 FROM HIS.dbo.prescription_item pi
    JOIN #valid_prescription vp ON vp.id = LTRIM(RTRIM(CONVERT(varchar(64), pi.prescription_id)))
    WHERE LTRIM(RTRIM(CONVERT(varchar(64), pi.drug_id))) = LTRIM(RTRIM(CONVERT(varchar(64), d.id)))
)
OR EXISTS (
    SELECT 1 FROM HIS.dbo.charge_item ci
    JOIN #valid_charge vc ON vc.id = LTRIM(RTRIM(CONVERT(varchar(64), ci.charge_id)))
    WHERE ci.drug_id IS NOT NULL
      AND LTRIM(RTRIM(CONVERT(varchar(64), ci.drug_id))) = LTRIM(RTRIM(CONVERT(varchar(64), d.id)))
);

SELECT DISTINCT
    id = LTRIM(RTRIM(CONVERT(varchar(64), d.id))),
    dispense_no = LEFT(d.dispense_no, 32),
    prescription_id = LTRIM(RTRIM(CONVERT(varchar(64), d.prescription_id))),
    status = CASE WHEN d.status = 'returned' THEN 'RETURNED' ELSE 'DONE' END,
    dispensed_at = d.created_at,
    operator_id = LTRIM(RTRIM(CONVERT(varchar(64), d.pharmacist_id))),
    created_at = d.created_at,
    updated_at = d.updated_at
INTO #valid_dispense
FROM HIS.dbo.dispense d
JOIN #valid_prescription vp ON vp.id = LTRIM(RTRIM(CONVERT(varchar(64), d.prescription_id)))
WHERE EXISTS (
    SELECT 1
    FROM HIS.dbo.dispense_item di
    JOIN HIS.dbo.prescription_item pi ON LTRIM(RTRIM(CONVERT(varchar(64), pi.id))) = LTRIM(RTRIM(CONVERT(varchar(64), di.prescription_item_id)))
    WHERE LTRIM(RTRIM(CONVERT(varchar(64), di.dispense_id))) = LTRIM(RTRIM(CONVERT(varchar(64), d.id)))
      AND LTRIM(RTRIM(CONVERT(varchar(64), pi.prescription_id))) = LTRIM(RTRIM(CONVERT(varchar(64), d.prescription_id)))
      AND ISNULL(di.quantity, 0) > 0
);

/* Target columns are narrower than several HIS text columns. Only rows that
   survived the mapping sets are checked; excluded history remains an export
   exception instead of blocking an otherwise valid batch. */
IF EXISTS (SELECT 1 FROM HIS.dbo.patient p JOIN #valid_patient v ON v.id = LTRIM(RTRIM(CONVERT(varchar(64), p.id))) WHERE LEN(p.name) > 64 OR LEN(p.id_type) > 16 OR LEN(p.id_no) > 32 OR LEN(p.phone) > 32 OR LEN(p.address) > 256 OR LEN(p.fee_category) > 32)
    THROW 51031, 'mapped patient text exceeds the BHIS target width.', 1;
IF EXISTS (SELECT 1 FROM HIS.dbo.registration r JOIN #valid_registration v ON v.id = LTRIM(RTRIM(CONVERT(varchar(64), r.id))) WHERE LEN(r.reg_no) > 32)
    THROW 51032, 'mapped registration number exceeds the BHIS target width.', 1;
IF EXISTS (SELECT 1 FROM HIS.dbo.prescription pr JOIN #valid_prescription v ON v.id = LTRIM(RTRIM(CONVERT(varchar(64), pr.id))) WHERE LEN(pr.rx_no) > 32)
    THROW 51033, 'mapped prescription number exceeds the BHIS target width.', 1;
IF EXISTS (SELECT 1 FROM HIS.dbo.charge c JOIN #valid_charge v ON v.id = LTRIM(RTRIM(CONVERT(varchar(64), c.id))) WHERE LEN(c.charge_no) > 32)
    THROW 51034, 'mapped charge number exceeds the BHIS target width.', 1;
IF EXISTS (SELECT 1 FROM HIS.dbo.drug d JOIN #valid_drug v ON v.id = LTRIM(RTRIM(CONVERT(varchar(64), d.id))) WHERE LEN(d.code) > 32 OR LEN(d.generic_name) > 128 OR LEN(d.spec) > 64 OR LEN(d.unit) > 16)
    THROW 51035, 'mapped drug text exceeds the BHIS target width.', 1;
IF EXISTS (SELECT 1 FROM HIS.dbo.dispense d JOIN #valid_dispense v ON v.id = LTRIM(RTRIM(CONVERT(varchar(64), d.id))) WHERE LEN(d.dispense_no) > 32)
    THROW 51036, 'mapped dispense number exceeds the BHIS target width.', 1;
IF EXISTS (SELECT 1 FROM HIS.dbo.prescription_item pi JOIN #valid_prescription v ON v.id = LTRIM(RTRIM(CONVERT(varchar(64), pi.prescription_id))) WHERE LEN(pi.usage_text) > 64 OR LEN(pi.frequency) > 64)
    THROW 51037, 'mapped prescription item text exceeds the BHIS target width.', 1;
IF EXISTS (SELECT 1 FROM HIS.dbo.charge_item ci JOIN #valid_charge v ON v.id = LTRIM(RTRIM(CONVERT(varchar(64), ci.charge_id))) WHERE LEN(ci.generic_name) > 128)
    THROW 51038, 'mapped charge item text exceeds the BHIS target width.', 1;

/* Cash-only history import: never silently coerce an unknown source state or
   payment into a BHIS value. The current source is known to satisfy these
   guards; future source changes must be reviewed before COMMIT=1. */
IF EXISTS (
    SELECT 1
    FROM HIS.dbo.registration r
    JOIN #valid_patient p ON p.id = LTRIM(RTRIM(CONVERT(varchar(64), r.patient_id)))
    WHERE LOWER(LTRIM(RTRIM(ISNULL(r.status, '')))) NOT IN ('waiting', 'in_visit', 'finished', 'cancelled')
)
    THROW 51025, 'registration status is outside the approved import mapping.', 1;
IF EXISTS (
    SELECT 1
    FROM HIS.dbo.prescription pr
    JOIN HIS.dbo.visit v ON LTRIM(RTRIM(CONVERT(varchar(64), v.id))) = LTRIM(RTRIM(CONVERT(varchar(64), pr.visit_id)))
    JOIN #valid_registration vr ON vr.id = LTRIM(RTRIM(CONVERT(varchar(64), v.registration_id)))
    WHERE LOWER(LTRIM(RTRIM(ISNULL(pr.status, '')))) NOT IN ('charged', 'executed')
)
    THROW 51026, 'prescription status is outside the approved import mapping.', 1;
IF EXISTS (
    SELECT 1
    FROM HIS.dbo.charge c
    JOIN #valid_charge vc ON vc.id = LTRIM(RTRIM(CONVERT(varchar(64), c.id)))
    WHERE LOWER(LTRIM(RTRIM(ISNULL(c.status, '')))) NOT IN ('paid', 'refunded')
)
    THROW 51027, 'charge status is outside the approved import mapping.', 1;
IF EXISTS (
    SELECT 1
    FROM HIS.dbo.dispense d
    JOIN #valid_dispense vd ON vd.id = LTRIM(RTRIM(CONVERT(varchar(64), d.id)))
    WHERE LOWER(LTRIM(RTRIM(ISNULL(d.status, '')))) NOT IN ('dispensed', 'returned')
)
    THROW 51028, 'dispense status is outside the approved import mapping.', 1;
IF EXISTS (
    SELECT 1
    FROM HIS.dbo.payment p
    JOIN #valid_charge vc ON vc.id = LTRIM(RTRIM(CONVERT(varchar(64), p.charge_id)))
    WHERE LOWER(LTRIM(RTRIM(ISNULL(p.method, '')))) <> 'cash'
)
    THROW 51029, 'non-cash payment is outside the current import scope.', 1;
IF EXISTS (
    SELECT 1
    FROM HIS.dbo.payment p
    JOIN #valid_charge vc ON vc.id = LTRIM(RTRIM(CONVERT(varchar(64), p.charge_id)))
    WHERE ISNULL(p.amount_cents, 0) = 0
)
    THROW 51030, 'zero-value payment is not importable as a cash charge/refund.', 1;

SELECT
    'valid_patients' AS entity, COUNT(*) AS source_rows FROM #valid_patient
UNION ALL SELECT 'valid_registrations', COUNT(*) FROM #valid_registration
UNION ALL SELECT 'valid_prescriptions', COUNT(*) FROM #valid_prescription
UNION ALL SELECT 'valid_charges', COUNT(*) FROM #valid_charge
UNION ALL SELECT 'valid_drugs', COUNT(*) FROM #valid_drug
UNION ALL SELECT 'valid_dispenses', COUNT(*) FROM #valid_dispense;

/* Fail early on collisions with non-import data. A rerun is allowed only for this batch. */
IF EXISTS (SELECT 1 FROM dbo.patient t JOIN #valid_patient s ON s.id = t.patient_id WHERE ISNULL(t.created_by, '') <> @batch)
    THROW 51001, 'patient_id collision with non-import data.', 1;
IF EXISTS (SELECT 1 FROM dbo.patient t JOIN #valid_patient s ON s.mrn = t.hosp_pk_no WHERE t.patient_id <> s.id AND t.is_deleted = 0)
    THROW 51002, 'patient MRN/hosp_pk_no collision.', 1;
IF EXISTS (
    SELECT 1 FROM #valid_patient s
    JOIN #valid_patient s2 ON s2.mrn = s.mrn AND s2.id <> s.id
    WHERE s.mrn <> ''
)
    THROW 51010, 'source patient MRN duplicate.', 1;
IF EXISTS (
    SELECT 1
    FROM dbo.patient t
    JOIN #valid_patient s ON NULLIF(s.id_number, '') = t.id_number
    WHERE t.is_deleted = 0 AND t.patient_id <> s.id
)
    THROW 51011, 'patient id_number collision.', 1;
IF EXISTS (
    SELECT 1
    FROM dbo.department t
    JOIN HIS.dbo.department s ON t.department_id = LTRIM(RTRIM(CONVERT(varchar(64), s.id)))
    JOIN (SELECT DISTINCT department_id FROM #valid_registration) x
        ON x.department_id = LTRIM(RTRIM(CONVERT(varchar(64), s.id)))
    WHERE ISNULL(t.created_by, '') <> @batch
)
    THROW 51003, 'department_id collision with non-import data.', 1;
IF EXISTS (
    SELECT 1
    FROM dbo.doctor t
    JOIN HIS.dbo.staff s ON t.doctor_id = LTRIM(RTRIM(CONVERT(varchar(64), s.id)))
    JOIN (SELECT DISTINCT doctor_id FROM #valid_registration) x
        ON x.doctor_id = LTRIM(RTRIM(CONVERT(varchar(64), s.id)))
    WHERE ISNULL(t.created_by, '') <> @batch
)
    THROW 51004, 'doctor_id collision with non-import data.', 1;
IF EXISTS (SELECT 1 FROM dbo.register_bill t JOIN #valid_registration s ON s.id = t.register_id WHERE ISNULL(t.created_by, '') <> @batch)
    THROW 51005, 'register_id collision with non-import data.', 1;
IF EXISTS (SELECT 1 FROM dbo.order_sheet t JOIN #valid_prescription s ON s.id = t.order_id WHERE ISNULL(t.created_by, '') <> @batch)
    THROW 51006, 'order_id collision with non-import data.', 1;
IF EXISTS (SELECT 1 FROM dbo.charge t JOIN #valid_charge s ON s.id = t.charge_id WHERE ISNULL(t.created_by, '') <> @batch)
    THROW 51007, 'charge_id collision with non-import data.', 1;
IF EXISTS (SELECT 1 FROM dbo.dispense t JOIN #valid_dispense s ON s.id = t.dispense_id WHERE ISNULL(t.created_by, '') <> @batch)
    THROW 51008, 'dispense_id collision with non-import data.', 1;
IF EXISTS (
    SELECT 1 FROM dbo.catalog_item t
    JOIN #valid_drug s ON s.item_no = t.item_no
    WHERE t.catalog_item_id <> s.id AND t.is_deleted = 0
)
    THROW 51009, 'catalog item_no collision.', 1;
IF EXISTS (
    SELECT 1
    FROM dbo.catalog_item t
    JOIN #valid_drug s ON s.id = t.catalog_item_id
    WHERE ISNULL(t.created_by, '') <> @batch
)
    THROW 51012, 'catalog_item_id collision with non-import data.', 1;
IF EXISTS (
    SELECT 1
    FROM dbo.department t
    JOIN (SELECT DISTINCT department_id FROM #valid_registration) x ON x.department_id = t.department_id
    JOIN HIS.dbo.department s ON x.department_id = LTRIM(RTRIM(CONVERT(varchar(64), s.id)))
    WHERE t.dept_code = LEFT(REPLACE(LTRIM(RTRIM(CONVERT(varchar(36), s.id))), '-', ''), 32)
      AND t.department_id <> x.department_id AND t.is_deleted = 0
)
    THROW 51013, 'department dept_code collision.', 1;
IF EXISTS (
    SELECT 1
    FROM dbo.doctor t
    JOIN (SELECT DISTINCT doctor_id FROM #valid_registration) x ON x.doctor_id = t.doctor_id
    JOIN HIS.dbo.staff s ON x.doctor_id = LTRIM(RTRIM(CONVERT(varchar(64), s.id)))
    WHERE t.doctor_code = LEFT(REPLACE(LTRIM(RTRIM(CONVERT(varchar(36), s.id))), '-', ''), 32)
      AND t.doctor_id <> x.doctor_id AND t.is_deleted = 0
)
    THROW 51014, 'doctor doctor_code collision.', 1;
IF EXISTS (SELECT 1 FROM dbo.register_bill t JOIN #valid_registration s ON s.register_no = t.register_no WHERE t.register_id <> s.id AND t.is_deleted = 0)
    THROW 51015, 'register_no collision.', 1;
IF EXISTS (SELECT 1 FROM dbo.order_sheet t JOIN #valid_prescription s ON s.rx_no = t.order_no WHERE t.order_id <> s.id AND t.is_deleted = 0)
    THROW 51016, 'order_no collision.', 1;
IF EXISTS (SELECT 1 FROM dbo.charge t JOIN #valid_charge s ON s.square_no = t.square_no WHERE t.charge_id <> s.id AND t.is_deleted = 0)
    THROW 51017, 'square_no collision.', 1;
IF EXISTS (SELECT 1 FROM dbo.dispense t JOIN #valid_dispense s ON s.dispense_no = t.dispense_no WHERE t.dispense_id <> s.id AND t.is_deleted = 0)
    THROW 51018, 'dispense_no collision.', 1;
IF EXISTS (
    SELECT 1 FROM dbo.payment t
    JOIN HIS.dbo.payment s ON LEFT(REPLACE(LTRIM(RTRIM(CONVERT(varchar(36), s.id))), '-', ''), 32) = t.payment_no
    JOIN #valid_charge vc ON vc.id = LTRIM(RTRIM(CONVERT(varchar(64), s.charge_id)))
    WHERE t.payment_id <> LTRIM(RTRIM(CONVERT(varchar(64), s.id)))
)
    THROW 51019, 'payment_no collision.', 1;

/* Base master data: only departments and doctors referenced by valid registrations. */
INSERT dbo.department (department_id, dept_code, dept_name, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT DISTINCT
    LTRIM(RTRIM(CONVERT(varchar(64), d.id))),
    LEFT(REPLACE(LTRIM(RTRIM(CONVERT(varchar(36), d.id))), '-', ''), 32),
    LEFT(d.name, 64),
    CASE WHEN d.cancelled_at IS NULL THEN 'ACTIVE' ELSE 'INACTIVE' END,
    d.created_at, @batch, d.updated_at, @batch, 0
FROM HIS.dbo.department d
JOIN (SELECT DISTINCT department_id FROM #valid_registration) x
    ON x.department_id = LTRIM(RTRIM(CONVERT(varchar(64), d.id)))
WHERE NOT EXISTS (SELECT 1 FROM dbo.department t WHERE t.department_id = x.department_id);

INSERT dbo.doctor (doctor_id, department_id, doctor_code, doctor_name, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT DISTINCT
    LTRIM(RTRIM(CONVERT(varchar(64), s.id))),
    LTRIM(RTRIM(CONVERT(varchar(64), s.department_id))),
    LEFT(REPLACE(LTRIM(RTRIM(CONVERT(varchar(36), s.id))), '-', ''), 32),
    LEFT(s.name, 64),
    CASE WHEN s.cancelled_at IS NULL AND s.left_at IS NULL THEN 'ACTIVE' ELSE 'INACTIVE' END,
    s.created_at, @batch, s.updated_at, @batch, 0
FROM HIS.dbo.staff s
JOIN (SELECT DISTINCT doctor_id FROM #valid_registration) x
    ON x.doctor_id = LTRIM(RTRIM(CONVERT(varchar(64), s.id)))
WHERE NOT EXISTS (SELECT 1 FROM dbo.doctor t WHERE t.doctor_id = x.doctor_id);

INSERT dbo.patient (patient_id, hosp_pk_no, patient_no, card_no, name, sex, birthday, id_type, id_number, phone, address, fare_identity, card_status, status, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT id, mrn, mrn, NULL, name, sex, birthday, id_type, id_number, phone, address, fare_identity, 'NOT_SENT', status, created_at, @batch, updated_at, @batch, 0
FROM #valid_patient s
WHERE NOT EXISTS (SELECT 1 FROM dbo.patient t WHERE t.patient_id = s.id);

INSERT dbo.catalog_item (catalog_item_id, item_no, item_name, item_type, spec, units, price, status, is_controlled, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT id, item_no, item_name, 'DRUG', spec, units, price, status, 0, created_at, @batch, updated_at, @batch, 0
FROM #valid_drug s
WHERE NOT EXISTS (SELECT 1 FROM dbo.catalog_item t WHERE t.catalog_item_id = s.id);

/* Registration and prescription history. */
INSERT dbo.register_bill (register_id, register_no, patient_id, department_id, doctor_id, register_sort, visit_date, status, register_fee, register_charged, registered_at, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT id, register_no, patient_id, department_id, doctor_id, NULL, visit_date, status, register_fee, register_charged, registered_at, created_at, @batch, updated_at, @batch, 0
FROM #valid_registration s
WHERE NOT EXISTS (SELECT 1 FROM dbo.register_bill t WHERE t.register_id = s.id);

INSERT dbo.order_sheet (order_id, order_no, register_id, patient_id, department_id, doctor_id, diagnose_name, recipe_sort, status, total_fee, charged, submitted_at, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT s.id, s.rx_no, s.registration_id, s.patient_id, s.department_id, s.doctor_id, dx.name, 'NORMAL', s.status, s.total_fee, s.charged, s.submitted_at, s.created_at, @batch, s.updated_at, @batch, 0
FROM #valid_prescription s
OUTER APPLY (
    SELECT TOP (1) d.name
    FROM HIS.dbo.diagnosis d
    JOIN HIS.dbo.visit v ON LTRIM(RTRIM(CONVERT(varchar(64), v.id))) = s.visit_id
    WHERE LTRIM(RTRIM(CONVERT(varchar(64), d.visit_id))) = s.visit_id
    ORDER BY d.is_main DESC, d.id
) dx
WHERE NOT EXISTS (SELECT 1 FROM dbo.order_sheet t WHERE t.order_id = s.id);

INSERT dbo.order_item (order_item_id, order_id, line_no, item_no, item_type, item_name, spec, units, amount, price, total_fee, usage_text, frequency_text, dosage, dosage_unit, proc_department_id, created_at, updated_at)
SELECT
    LTRIM(RTRIM(CONVERT(varchar(64), pi.id))),
    vp.id,
    pi.line_no,
    LEFT(d.code, 32),
    'DRUG',
    LEFT(d.generic_name, 128),
    LEFT(d.spec, 64),
    LEFT(d.unit, 16),
    CONVERT(decimal(18,3), pi.quantity),
    CONVERT(decimal(18,2), ISNULL(pi.price_cents, 0)) / 100,
    CONVERT(decimal(18,2), ISNULL(pi.amount_cents, 0)) / 100,
    LEFT(pi.usage_text, 64),
    LEFT(pi.frequency, 64),
    NULL, NULL, NULL,
    vp.created_at, vp.updated_at
FROM HIS.dbo.prescription_item pi
JOIN #valid_prescription vp ON vp.id = LTRIM(RTRIM(CONVERT(varchar(64), pi.prescription_id)))
JOIN HIS.dbo.drug d ON LTRIM(RTRIM(CONVERT(varchar(64), d.id))) = LTRIM(RTRIM(CONVERT(varchar(64), pi.drug_id)))
WHERE ISNULL(pi.quantity, 0) > 0
  AND NOT EXISTS (SELECT 1 FROM dbo.order_item t WHERE t.order_item_id = LTRIM(RTRIM(CONVERT(varchar(64), pi.id))));

/* Charges and line items. Favourable fee is intentionally zero in this import. */
INSERT dbo.charge (charge_id, square_no, order_id, register_id, patient_id, total_fee, square_fee, favourable_fee, status, paid_at, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT
    s.id, s.square_no,
    CASE WHEN EXISTS (SELECT 1 FROM #valid_prescription vp WHERE vp.id = s.prescription_id) THEN s.prescription_id ELSE NULL END,
    s.register_id, s.patient_id,
    s.total_fee, s.total_fee, CONVERT(decimal(18,2), 0), s.status,
    CASE WHEN EXISTS (SELECT 1 FROM HIS.dbo.payment p WHERE LTRIM(RTRIM(CONVERT(varchar(64), p.charge_id))) = s.id AND ISNULL(p.amount_cents, 0) > 0) THEN s.created_at ELSE NULL END,
    s.created_at, @batch, s.updated_at, @batch, 0
FROM #valid_charge s
WHERE NOT EXISTS (SELECT 1 FROM dbo.charge t WHERE t.charge_id = s.id);

UPDATE c
SET paid_at = CASE
    WHEN EXISTS (
        SELECT 1
        FROM HIS.dbo.payment p
        WHERE LTRIM(RTRIM(CONVERT(varchar(64), p.charge_id))) = c.charge_id
          AND ISNULL(p.amount_cents, 0) > 0
    ) THEN c.created_at
    ELSE NULL
END
FROM dbo.charge c
JOIN #valid_charge s ON s.id = c.charge_id
WHERE c.created_by = @batch;

INSERT dbo.charge_item (charge_item_id, charge_id, order_item_id, line_no, item_no, item_name, qty, unit_price, total_fee, can_back, refunded_qty, created_at, updated_at)
SELECT
    LTRIM(RTRIM(CONVERT(varchar(64), ci.id))),
    vc.id,
    oi.order_item_id,
    ci.line_no,
    CASE WHEN ci.drug_id IS NOT NULL AND d.code IS NOT NULL THEN LEFT(d.code, 32) ELSE LEFT('HIS-' + REPLACE(LTRIM(RTRIM(CONVERT(varchar(36), ci.id))), '-', ''), 32) END,
    LEFT(ci.generic_name, 128),
    CONVERT(decimal(18,3), ci.quantity),
    CONVERT(decimal(18,2), ISNULL(ci.price_cents, 0)) / 100,
    CONVERT(decimal(18,2), ISNULL(ci.amount_cents, 0)) / 100,
    1,
    CASE WHEN vc.status = 'ABATE' THEN CONVERT(decimal(18,3), ci.quantity) ELSE CONVERT(decimal(18,3), 0) END,
    vc.created_at,
    vc.updated_at
FROM HIS.dbo.charge_item ci
JOIN #valid_charge vc ON vc.id = LTRIM(RTRIM(CONVERT(varchar(64), ci.charge_id)))
LEFT JOIN HIS.dbo.drug d ON ci.drug_id IS NOT NULL AND LTRIM(RTRIM(CONVERT(varchar(64), d.id))) = LTRIM(RTRIM(CONVERT(varchar(64), ci.drug_id)))
OUTER APPLY (
    SELECT TOP (1) LTRIM(RTRIM(CONVERT(varchar(64), pi.id))) AS order_item_id
    FROM HIS.dbo.prescription_item pi
    WHERE ci.drug_id IS NOT NULL
      AND ci.drug_id = pi.drug_id
      AND EXISTS (SELECT 1 FROM #valid_prescription vp WHERE vp.id = vc.prescription_id AND vp.id = LTRIM(RTRIM(CONVERT(varchar(64), pi.prescription_id))))
    ORDER BY pi.line_no, pi.id
) oi
WHERE ISNULL(ci.quantity, 0) > 0
  AND ISNULL(ci.amount_cents, 0) >= 0
  AND NOT EXISTS (SELECT 1 FROM dbo.charge_item t WHERE t.charge_item_id = LTRIM(RTRIM(CONVERT(varchar(64), ci.id))));

UPDATE target_item
SET order_item_id = oi.order_item_id
FROM dbo.charge_item target_item
JOIN #valid_charge vc ON vc.id = target_item.charge_id
JOIN HIS.dbo.charge_item source_item
    ON LTRIM(RTRIM(CONVERT(varchar(64), source_item.id))) = target_item.charge_item_id
OUTER APPLY (
    SELECT TOP (1) LTRIM(RTRIM(CONVERT(varchar(64), pi.id))) AS order_item_id
    FROM HIS.dbo.prescription_item pi
    WHERE source_item.drug_id IS NOT NULL
      AND source_item.drug_id = pi.drug_id
      AND EXISTS (SELECT 1 FROM #valid_prescription vp WHERE vp.id = vc.prescription_id AND vp.id = LTRIM(RTRIM(CONVERT(varchar(64), pi.prescription_id))))
    ORDER BY pi.line_no, pi.id
) oi
WHERE target_item.order_item_id IS NULL
  AND target_item.charge_id = vc.id;

INSERT dbo.payment (payment_id, payment_no, charge_id, direction, pay_type, amount, status, paid_at, created_at, created_by, updated_at, updated_by)
SELECT
    LTRIM(RTRIM(CONVERT(varchar(64), p.id))),
    LEFT(REPLACE(LTRIM(RTRIM(CONVERT(varchar(36), p.id))), '-', ''), 32),
    vc.id,
    CASE WHEN ISNULL(p.amount_cents, 0) < 0 THEN 'REFUND' ELSE 'CHARGE' END,
    'CASH',
    CONVERT(decimal(18,2), ABS(ISNULL(p.amount_cents, 0))) / 100,
    'SUCCEEDED',
    p.paid_at,
    p.paid_at,
    @batch,
    p.paid_at,
    @batch
FROM HIS.dbo.payment p
JOIN #valid_charge vc ON vc.id = LTRIM(RTRIM(CONVERT(varchar(64), p.charge_id)))
WHERE NOT EXISTS (SELECT 1 FROM dbo.payment t WHERE t.payment_id = LTRIM(RTRIM(CONVERT(varchar(64), p.id))));

/* Dispensing is imported without source batch/expiry references. Inventory stays unchanged. */
INSERT dbo.dispense (dispense_id, dispense_no, order_id, charge_id, register_id, patient_id, status, dispensed_at, operator_id, return_reason, created_at, created_by, updated_at, updated_by, is_deleted)
SELECT
    d.id,
    d.dispense_no,
    vp.id,
    vc.id,
    vp.registration_id,
    vp.patient_id,
    d.status,
    d.dispensed_at,
    d.operator_id,
    CASE WHEN d.status = 'RETURNED' THEN N'HIS_SOURCE_RETURNED' ELSE NULL END,
    d.created_at,
    @batch,
    d.updated_at,
    @batch,
    0
FROM #valid_dispense d
JOIN #valid_prescription vp ON vp.id = d.prescription_id
OUTER APPLY (
    SELECT TOP (1) c.id
    FROM #valid_charge c
    WHERE c.prescription_id = d.prescription_id
    ORDER BY c.created_at, c.id
) vc
WHERE NOT EXISTS (SELECT 1 FROM dbo.dispense t WHERE t.dispense_id = d.id);

INSERT dbo.dispense_item (dispense_item_id, dispense_id, order_item_id, charge_item_id, item_no, item_name, units, qty_charged, qty_dispensed, qty_returned, line_status, created_at, updated_at)
SELECT
    LTRIM(RTRIM(CONVERT(varchar(64), di.id))),
    vd.id,
    LTRIM(RTRIM(CONVERT(varchar(64), di.prescription_item_id))),
    ci.id,
    LEFT(dr.code, 32),
    LEFT(dr.generic_name, 128),
    LEFT(dr.unit, 16),
    CONVERT(decimal(18,3), pi.quantity),
    CONVERT(decimal(18,3), di.quantity),
    CASE WHEN vd.status = 'RETURNED' THEN CONVERT(decimal(18,3), di.quantity) ELSE CONVERT(decimal(18,3), 0) END,
    CASE WHEN vd.status = 'RETURNED' THEN 'NONE' WHEN di.quantity >= pi.quantity THEN 'FULL' ELSE 'PARTIAL' END,
    vd.created_at,
    vd.updated_at
FROM HIS.dbo.dispense_item di
JOIN #valid_dispense vd ON vd.id = LTRIM(RTRIM(CONVERT(varchar(64), di.dispense_id)))
JOIN HIS.dbo.prescription_item pi ON LTRIM(RTRIM(CONVERT(varchar(64), pi.id))) = LTRIM(RTRIM(CONVERT(varchar(64), di.prescription_item_id)))
JOIN HIS.dbo.drug dr ON LTRIM(RTRIM(CONVERT(varchar(64), dr.id))) = LTRIM(RTRIM(CONVERT(varchar(64), di.drug_id)))
OUTER APPLY (
    SELECT TOP (1) ci.id
    FROM HIS.dbo.charge_item ci
    JOIN #valid_charge vc ON vc.id = LTRIM(RTRIM(CONVERT(varchar(64), ci.charge_id)))
    JOIN #valid_prescription vp ON vp.id = vd.prescription_id
    WHERE ci.drug_id = di.drug_id
      AND vc.prescription_id = vd.prescription_id
    ORDER BY ci.line_no, ci.id
) ci
WHERE ISNULL(di.quantity, 0) > 0
  AND di.quantity <= pi.quantity
  AND di.drug_id = pi.drug_id
  AND NOT EXISTS (SELECT 1 FROM dbo.dispense_item t WHERE t.dispense_item_id = LTRIM(RTRIM(CONVERT(varchar(64), di.id))));

SELECT
    'patients' entity, COUNT(*) rows_written FROM dbo.patient WHERE created_by = @batch
UNION ALL SELECT 'departments', COUNT(*) FROM dbo.department WHERE created_by = @batch
UNION ALL SELECT 'doctors', COUNT(*) FROM dbo.doctor WHERE created_by = @batch
UNION ALL SELECT 'catalog_items', COUNT(*) FROM dbo.catalog_item WHERE created_by = @batch
UNION ALL SELECT 'registers', COUNT(*) FROM dbo.register_bill WHERE created_by = @batch
UNION ALL SELECT 'orders', COUNT(*) FROM dbo.order_sheet WHERE created_by = @batch
UNION ALL SELECT 'charges', COUNT(*) FROM dbo.charge WHERE created_by = @batch
UNION ALL SELECT 'payments', COUNT(*) FROM dbo.payment WHERE created_by = @batch
UNION ALL SELECT 'dispenses', COUNT(*) FROM dbo.dispense WHERE created_by = @batch;

IF '$(COMMIT)' = '1'
BEGIN
    COMMIT TRANSACTION;
    PRINT 'HIS_IMPORT_COMMITTED';
END
ELSE
BEGIN
    ROLLBACK TRANSACTION;
    PRINT 'HIS_IMPORT_DRY_RUN_ROLLED_BACK';
END;
