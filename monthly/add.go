package monthly

import (
	"context"
	"fmt"
	"time"

	"github.com/ishanwardhono/expense-function/common"
)

func Add(ctx context.Context, req AddRequest) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	db, err := common.ConnectDatabase(cfg.dbConfig)
	if err != nil {
		return err
	}
	defer db.Close()

	// Use provided date or current time
	t := cfg.time
	if req.Date != nil {
		t, err = time.Parse(time.DateTime, *req.Date)
		if err != nil {
			return fmt.Errorf("invalid date format: %v", err)
		}
		t = t.In(common.Loc)
	}

	monthData := getPayPeriodMonth(t)
	expense := req.ToMonthlyExpense(monthData, t)

	return addMonthlyExpense(ctx, db, expense)
}
