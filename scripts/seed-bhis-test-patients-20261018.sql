-- Synthetic BHIS patient seed for local testing only.
-- All rows are marked TEST-HOSP-20261018 and contain no real identity data.
SET NOCOUNT ON;
SET XACT_ABORT ON;
SET ANSI_NULLS ON;
SET ANSI_PADDING ON;
SET ANSI_WARNINGS ON;
SET ARITHABORT ON;
SET CONCAT_NULL_YIELDS_NULL ON;
SET QUOTED_IDENTIFIER ON;
SET NUMERIC_ROUNDABORT OFF;

IF DB_NAME() <> 'BHIS'
    THROW 51002, 'This seed script may only run in the BHIS database.', 1;

BEGIN TRANSACTION;

DECLARE @Batch varchar(32) = 'TEST-HOSP-20261018';

IF EXISTS (
    SELECT 1
    FROM dbo.patient
    WHERE hosp_pk_no LIKE @Batch + '-%'
)
    THROW 51000, 'The test batch already exists; refusing to duplicate it.', 1;

;WITH Numbers AS (
    SELECT TOP (100)
        ROW_NUMBER() OVER (ORDER BY (SELECT NULL)) AS n
    FROM sys.all_objects AS a
    CROSS JOIN sys.all_objects AS b
), Surnames AS (
    SELECT *
    FROM (VALUES
        (0, N'赵'), (1, N'钱'), (2, N'孙'), (3, N'李'), (4, N'周'),
        (5, N'吴'), (6, N'郑'), (7, N'王'), (8, N'冯'), (9, N'陈')
    ) AS s(i, value)
), GivenNames AS (
    SELECT *
    FROM (VALUES
        (0, N'安'), (1, N'宁'), (2, N'伟'), (3, N'敏'), (4, N'静'),
        (5, N'磊'), (6, N'晨'), (7, N'悦'), (8, N'婷'), (9, N'鑫')
    ) AS g(i, value)
)
INSERT INTO dbo.patient (
    patient_id, hosp_pk_no, patient_no, card_no, name, sex, birthday,
    id_type, id_number, phone, address, fare_identity, card_status, status,
    created_at, created_by, updated_at, updated_by, is_deleted
)
SELECT
    CONVERT(varchar(64), NEWID()),
    CONCAT(@Batch, '-', RIGHT(CONCAT('000', CONVERT(varchar(3), n.n)), 3)),
    CONCAT('TEST-PAT-20261018-', RIGHT(CONCAT('000', CONVERT(varchar(3), n.n)), 3)),
    CASE WHEN n.n % 4 = 0 THEN NULL
         ELSE CONCAT('TEST-CARD-20261018-', RIGHT(CONCAT('000', CONVERT(varchar(3), n.n)), 3)) END,
    CONCAT(s.value, g.value),
    CASE WHEN n.n % 10 = 0 THEN 'U' WHEN n.n % 2 = 0 THEN 'F' ELSE 'M' END,
    DATEFROMPARTS(1958 + ((n.n * 7) % 52), 1 + ((n.n * 11) % 12), 1 + ((n.n * 13) % 27)),
    'OTHER',
    CONCAT('TEST-ID-20261018-', RIGHT(CONCAT('000', CONVERT(varchar(3), n.n)), 3)),
    CONCAT('1990000', RIGHT(CONCAT('0000', CONVERT(varchar(4), n.n)), 4)),
    CONCAT(N'山西省太原市小店区测试街', CONVERT(nvarchar(4), n.n), N'号'),
    CASE WHEN n.n % 5 = 0 THEN 'EMPLOYEE' ELSE 'SELF' END,
    CASE WHEN n.n % 4 = 0 THEN 'NOT_SENT' ELSE 'NORMAL' END,
    'ACTIVE',
    DATEADD(day, -(n.n % 365), SYSUTCDATETIME()),
    'system',
    DATEADD(day, -(n.n % 365), SYSUTCDATETIME()),
    'system',
    0
FROM Numbers AS n
JOIN Surnames AS s ON s.i = (n.n - 1) / 10
JOIN GivenNames AS g ON g.i = (n.n - 1) % 10;

DECLARE @Inserted int = @@ROWCOUNT;
IF @Inserted <> 100
    THROW 51001, 'The synthetic patient seed did not insert exactly 100 rows.', 1;

COMMIT TRANSACTION;

SELECT
    'BHIS' AS database_name,
    @Batch AS test_batch,
    COUNT(*) AS inserted_patients,
    MIN(hosp_pk_no) AS first_hosp_pk_no,
    MAX(hosp_pk_no) AS last_hosp_pk_no
FROM dbo.patient
WHERE hosp_pk_no LIKE @Batch + '-%';

-- Rollback after verification, if needed:
-- DELETE FROM dbo.patient WHERE hosp_pk_no LIKE 'TEST-HOSP-20261018-%';
