package recap

import (
	"context"
	"fmt"
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

func toRecapResponse(monthRecap monthly.RecapResp, weekRecap []weekly.RecapResp) RecapResponse {
	expense := int64(0)
	remaining := int64(0)
	recapData := make([]RecapData, 0)

	for i := 0; i < monthRecap.TotalWeeks; i++ {
		if i >= len(weekRecap) {
			remaining += weekly.MaxExpense * 2
			recapData = append(recapData, EmptyRecapData(fmt.Sprintf("Minggu ke-%d", i+1)))
			continue
		}

		expense += weekRecap[i].Amount
		remaining += weekRecap[i].Remaining

		recapData = append(recapData, RecapData{
			Description: fmt.Sprintf("Minggu ke-%d", i+1),
			Amount:      common.FormatRupiah(weekRecap[i].Amount),
			Remaining:   common.ToDataLabel(weekRecap[i].Remaining, true),
		})
	}

	expense += monthRecap.Amount
	remaining += monthRecap.Remaining
	recapData = append(recapData, RecapData{
		Description: "Bulanan",
		Amount:      common.FormatRupiah(monthRecap.Amount),
		Remaining:   common.ToDataLabel(monthRecap.Remaining, true),
	})

	return RecapResponse{
		DateLabel: fmt.Sprintf("%s %d", monthRecap.MonthLabel, monthRecap.Year),
		Expense:   common.FormatRupiah(expense),
		Remaining: common.ToDataLabel(remaining, true),
		Details:   recapData,
		PrevMonth: newPrevMonth(monthRecap.Month, monthRecap.Year),
	}
}
