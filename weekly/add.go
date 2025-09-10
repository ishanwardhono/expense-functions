package weekly

import (
	"context"
	"fmt"
	"time"

	"github.com/ishanwardhono/expense-function/common"
)

func Add(ctx context.Context, req AddRequest) error {
	if req.Amount == 0 {
		return fmt.Errorf("amount cannot be zero")
	}

	cfg, err := common.LoadConfig()
	if err != nil {
		return err
	}

	t := cfg.Time
	if req.Date != nil {
		t, err = time.Parse(time.DateTime, *req.Date)
		if err != nil {
			return fmt.Errorf("invalid date format: %v", err)
		}
	}

	db, err := common.ConnectDatabase(cfg.DbConfig)
	if err != nil {
		return err
	}
	defer db.Close()

	weekData := getWeekData(t)
	expense := req.ToExpense(weekData, t)
	err = addExpense(ctx, db, expense)
	if err != nil {
		return err
	}

	return nil
}
