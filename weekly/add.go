package weekly

import (
	"context"
	"fmt"
	"time"
)

func Add(ctx context.Context, req AddRequest) error {
	if req.Amount == 0 {
		return fmt.Errorf("amount cannot be zero")
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	t := cfg.time
	if req.Date != nil {
		t, err = time.Parse(time.DateTime, *req.Date)
		if err != nil {
			return fmt.Errorf("invalid date format: %v", err)
		}
	}

	db, err := connectDatabase(cfg)
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
