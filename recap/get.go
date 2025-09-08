package recap

import (
	"context"
	"time"

	"github.com/ishanwardhono/expense-function/common"
	"github.com/ishanwardhono/expense-function/monthly"
	"github.com/ishanwardhono/expense-function/weekly"
)

func Get(ctx context.Context, req GetRequest) (RecapResponse, error) {
	resp := RecapResponse{}

	cfg, err := common.LoadConfig()
	if err != nil {
		return resp, err
	}
	if req.Year != 0 && req.Month != 0 {
		cfg.Time = time.Date(req.Year, time.Month(req.Month), 24, 0, 0, 0, 0, common.Loc)
	}

	db, err := common.ConnectDatabase(cfg.DbConfig)
	if err != nil {
		return resp, err
	}
	defer db.Close()

	monthlyRecap, err := monthly.Recapitulation(ctx, db, cfg.Time)
	if err != nil {
		return resp, err
	}

	weeklyRecap, err := weekly.Recapitulation(ctx, db, monthlyRecap.StartYear, monthlyRecap.StartWeek, monthlyRecap.EndYear, monthlyRecap.EndWeek)
	if err != nil {
		return resp, err
	}

	return toRecapResponse(monthlyRecap, weeklyRecap), nil
}
