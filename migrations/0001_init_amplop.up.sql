-- 0001_init_amplop.sql — Amplop v2 schema (spec §5).
--
-- Apply order:
--   1. devdb  (local / testing)      — run first, exercise integration tests.
--   2. defaultdb (production)        — only at Phase 4, first production deploy.
--
-- No v1 data is migrated (decision D4); legacy public.expense /
-- public.monthly_expense are left untouched and dropped later.

CREATE SCHEMA IF NOT EXISTS amplop;

-- Subscriptions: stable identity --------------------------------------------
CREATE TABLE IF NOT EXISTS amplop.subscription (
    id         UUID        NOT NULL DEFAULT gen_random_uuid(),
    name       STRING      NOT NULL,
    color      STRING      NOT NULL DEFAULT '',   -- cosmetic; NOT effective-dated
    created_at TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),
    CONSTRAINT subscription_pk PRIMARY KEY (id)
);

-- Subscriptions: effective-dated attributes ---------------------------------
-- A version applies from (effective_year, effective_month) onward until the
-- next version. "active=false" ends the subscription as of that month.
-- The UNIQUE constraint below doubles as the index for §5.1 resolution.
CREATE TABLE IF NOT EXISTS amplop.subscription_version (
    id              UUID        NOT NULL DEFAULT gen_random_uuid(),
    subscription_id UUID        NOT NULL REFERENCES amplop.subscription (id) ON DELETE CASCADE,
    effective_year  INT2        NOT NULL,
    effective_month INT2        NOT NULL,         -- 1..12
    alloc           INT8        NOT NULL,         -- monthly allocation, > 0
    due_day         INT2        NOT NULL,         -- recurring day-of-month (1..31)
    active          BOOL        NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),
    CONSTRAINT subscription_version_pk PRIMARY KEY (id),
    CONSTRAINT subscription_version_alloc_positive CHECK (alloc > 0),
    CONSTRAINT subscription_version_due_day_valid  CHECK (due_day BETWEEN 1 AND 31),
    CONSTRAINT subscription_version_month_valid    CHECK (effective_month BETWEEN 1 AND 12),
    CONSTRAINT subscription_version_uniq UNIQUE (subscription_id, effective_year, effective_month)
);

-- Expenses: single source of ALL transactions (incl. subscription payments) --
CREATE TABLE IF NOT EXISTS amplop.expense (
    id              UUID        NOT NULL DEFAULT gen_random_uuid(),
    occurred_date   DATE        NOT NULL,
    occurred_time   TIME        NULL,             -- stored as SQL TIME; serialized as RFC3339 (§7.2)
    amount          INT8        NOT NULL,         -- Rupiah, > 0
    category        STRING      NOT NULL,         -- Makan|Belanja|Jajan|Cash|Lainnya|Langganan
    subscription_id UUID        NULL REFERENCES amplop.subscription (id),  -- "subcategory"; set iff Langganan
    note            STRING      NOT NULL DEFAULT '',
    -- calendar period of the transaction (drives the once-per-month rule below)
    occurred_year   INT2        AS (EXTRACT(year  FROM occurred_date)::INT2) STORED,
    occurred_month  INT2        AS (EXTRACT(month FROM occurred_date)::INT2) STORED,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),
    CONSTRAINT expense_pk PRIMARY KEY (id),
    CONSTRAINT expense_amount_positive CHECK (amount > 0),
    CONSTRAINT expense_category_valid
        CHECK (category IN ('Makan','Belanja','Jajan','Cash','Lainnya','Langganan')),
    -- subscription_id present  <=>  category is Langganan
    CONSTRAINT expense_langganan_link
        CHECK ((category = 'Langganan') = (subscription_id IS NOT NULL))
);
CREATE INDEX IF NOT EXISTS expense_occurred_date_idx   ON amplop.expense (occurred_date);
CREATE INDEX IF NOT EXISTS expense_category_idx        ON amplop.expense (category);
CREATE INDEX IF NOT EXISTS expense_subscription_id_idx ON amplop.expense (subscription_id);

-- A subscription can be paid at most ONCE per calendar month (Langganan rows
-- only; non-subscription expenses have subscription_id NULL and are excluded).
CREATE UNIQUE INDEX IF NOT EXISTS expense_one_sub_payment_per_month
    ON amplop.expense (subscription_id, occurred_year, occurred_month)
    WHERE subscription_id IS NOT NULL;

-- Budget config: effective-dated --------------------------------------------
-- The UNIQUE constraint doubles as the index for §5.1 resolution.
CREATE TABLE IF NOT EXISTS amplop.budget_config (
    id              UUID        NOT NULL DEFAULT gen_random_uuid(),
    effective_year  INT2        NOT NULL,
    effective_month INT2        NOT NULL,         -- 1..12
    monthly         INT8        NOT NULL,
    shop_weekly     INT8        NOT NULL,
    weekend_budget  INT8        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),
    CONSTRAINT budget_config_pk PRIMARY KEY (id),
    CONSTRAINT budget_config_month_valid CHECK (effective_month BETWEEN 1 AND 12),
    CONSTRAINT budget_config_effective_uniq UNIQUE (effective_year, effective_month)
);

-- Baseline (D8): locked at 2025-01 so every viewable month resolves; defaults
-- from the prototype CFG.
INSERT INTO amplop.budget_config (effective_year, effective_month, monthly, shop_weekly, weekend_budget)
VALUES (2025, 1, 5000000, 600000, 200000)
ON CONFLICT (effective_year, effective_month) DO NOTHING;
