CREATE TABLE public.expense (
	id UUID NOT NULL DEFAULT gen_random_uuid(),
	year INT2 NOT NULL,
	week INT2 NOT NULL,
    day INT2 NOT NULL,
	amount INT4 NOT NULL DEFAULT 0,
    type VARCHAR(20) NOT NULL,
    note TEXT NOT NULL DEFAULT '',
	created_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	CONSTRAINT expense_pk PRIMARY KEY (id)
);
CREATE INDEX expense_year_week_day_idx ON public.expense (year, week, day);
CREATE INDEX expense_created_time_idx ON public.expense (created_time DESC);
CREATE INDEX expense_type_idx ON public.expense (type);

CREATE TABLE public.monthly_expense (
	id UUID NOT NULL DEFAULT gen_random_uuid(),
	year INT2 NOT NULL,
	month INT2 NOT NULL,
	amount INT4 NOT NULL DEFAULT 0:::INT8,
	type VARCHAR(20) NOT NULL,
	note STRING NOT NULL DEFAULT '':::STRING,
	created_time TIMESTAMP NOT NULL DEFAULT current_timestamp():::TIMESTAMP,
	CONSTRAINT expense_pk PRIMARY KEY (id ASC),
	INDEX expense_year_week_day_idx (year ASC, month ASC),
	INDEX expense_created_time_idx (created_time DESC),
	INDEX expense_type_idx (type ASC)
);