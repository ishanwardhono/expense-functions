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

	t := now()
	if req.Date != nil {
		t, err = time.Parse(time.DateOnly, *req.Date)
		if err != nil {
			return err
		}
	}

	db, err := connectDatabase(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	weekData := getWeekData(t)
	if weekData.day < 5 {
		err = addWeekdayExpense(ctx, db, weekData.year, weekData.week, req.Amount)
	} else {
		err = addWeekendExpense(ctx, db, weekData.year, weekData.week, req.Amount)
	}
	if err != nil {
		return err
	}

	return nil
}
