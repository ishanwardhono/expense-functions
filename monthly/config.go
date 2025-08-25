package monthly

import (
	"time"

	"github.com/ishanwardhono/expense-function/common"
)

type config struct {
	maxExpense int64
	time       time.Time
	dbConfig   *common.DatabaseConfig
}

func loadConfig() (*config, error) {
	maxExpense, err := common.LoadMaxMonthlyExpense()
	if err != nil {
		return nil, err
	}

	t, err := common.LoadTime()
	if err != nil {
		return nil, err
	}

	return &config{
		maxExpense: maxExpense,
		time:       t,
		dbConfig:   common.LoadDatabaseConfig(),
	}, nil
}
