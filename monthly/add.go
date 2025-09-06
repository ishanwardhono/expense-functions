package monthly

import (
	"context"
	"fmt"
	"time"

	"github.com/ishanwardhono/expense-function/common"
)

func Add(ctx context.Context, req AddRequest) error {
	cfg, err := common.LoadConfig()
	if err != nil {
		return err
	}

	db, err := common.ConnectDatabase(cfg.DbConfig)
	if err != nil {
		return err
	}
	defer db.Close()

	// Use provided date or current time
	t := cfg.Time
	if req.Date != nil {
		t, err = time.Parse(time.DateTime, *req.Date)
		if err != nil {
			return fmt.Errorf("invalid date format: %v", err)
		}
	}

	monthData := getPayPeriodMonth(t)
	expense := req.ToMonthlyExpense(monthData, t)

	return addMonthlyExpense(ctx, db, expense)
}
