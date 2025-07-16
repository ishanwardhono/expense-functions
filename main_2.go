package expensefunction

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"github.com/spf13/viper"
)

type WeekData struct {
	Year int
	Week int
	Day  int
}

type WeeklyExpense struct {
	ID          int       `db:"id"`
	Year        int       `db:"year"`
	Week        int       `db:"week"`
	Weekday     int64     `db:"weekday"`
	Weekend     int64     `db:"weekend"`
	CreatedTime time.Time `db:"created_time"`
}

type WeeklyExpenseResponse struct {
	Year     int                    `json:"year"`
	Week     int                    `json:"week"`
	DayLabel string                 `json:"day_label"`
	Remaning WeeklyExpenseRemaining `json:"remaining"`
}

type WeeklyExpenseRemaining struct {
	Weekday string   `json:"weekday"`
	Weekend string   `json:"weekend"`
	Days    []string `json:"days"`
}

func main2() {
	config, err := loadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.FuncTimeout)*time.Second)
	defer cancel()

	db, err := connectDatabase(config)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	weekData := getCurrentWeekData()
	weeklyExpense, err := getCurrentWeekExpense(ctx, db, weekData)
	if err != nil {
		log.Fatal("Failed to get current week expense:", err)
	}

	remaining := calculateRemainingExpense(weekData.Day, weeklyExpense, config.MaxExpense)
	response := buildResponse(weekData, remaining)
	jsonResponse, _ := json.Marshal(response)
	fmt.Printf("response: %s\n", jsonResponse)
}

func getCurrentWeekData() WeekData {
	now := time.Now()
	year, week := now.ISOWeek()
	day := int(now.Weekday())
	return WeekData{Year: year, Week: week, Day: day}
}

func calculateRemainingExpense(day int, expense WeeklyExpense, maxExpense int64) WeeklyExpenseRemaining {
	weekdayRemaining := maxExpense - expense.Weekday
	weekendRemaining := maxExpense - expense.Weekend

	response := WeeklyExpenseRemaining{
		Weekday: formatRupiah(weekdayRemaining),
		Weekend: formatRupiah(weekendRemaining),
		Days:    make([]string, 7),
	}

	// If today is a weekday (Monday to Friday)
	if day >= 1 && day <= 5 {
		response.weekdayExpense(day, weekdayRemaining, weekendRemaining)
		return response
	}

	// If today is a weekend (Saturday or Sunday)
	response.weekendExpense(day, weekendRemaining)
	return response
}

func (r *WeeklyExpenseRemaining) weekdayExpense(day int, weekdayRemaining, weekendRemaining int64) {
	weekdayRemainingDay := 6 - day
	weekdayRemainingPerDay := weekdayRemaining / int64(weekdayRemainingDay)
	for i := day; i <= 5; i++ {
		strDay := "Ga ada jajan"
		if weekdayRemainingPerDay > 0 {
			strDay = formatRupiah(weekdayRemainingPerDay)
		}
		r.Days[i] = strDay
	}
	r.weekendExpense(day, weekendRemaining)
}

func (r *WeeklyExpenseRemaining) weekendExpense(day int, weekendRemaining int64) {
	weekendRemainingPerDay := weekendRemaining
	if day != 0 {
		weekendRemainingPerDay = weekendRemaining / 2
		r.Days[6] = formatRupiah(weekendRemainingPerDay)
	}
	r.Days[0] = formatRupiah(weekendRemainingPerDay)
}

func buildResponse(weekData WeekData, remaining WeeklyExpenseRemaining) WeeklyExpenseResponse {
	dayLabel := mapDayLabel[weekData.Day]

	return WeeklyExpenseResponse{
		Year:     weekData.Year,
		Week:     weekData.Week,
		DayLabel: dayLabel,
		Remaning: remaining,
	}
}

var mapDayLabel = map[int]string{
	0: "Minggu",
	1: "Senin",
	2: "Selasa",
	3: "Rabu",
	4: "Kamis",
	5: "Jumat",
	6: "Sabtu",
}

type Config struct {
	FuncTimeout      int
	DatabaseHost     string
	DatabasePort     string
	DatabaseUser     string
	DatabasePassword string
	DatabaseName     string
	DatabaseTimeout  int

	MaxExpense int64
}

func loadConfig() (*Config, error) {
	v := viper.New()
	v.SetConfigFile(".env")
	v.SetConfigType("env")

	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	return &Config{
		FuncTimeout:      v.GetInt("FUNC_TIMEOUT"),
		DatabaseHost:     v.GetString("DB_HOST"),
		DatabasePort:     v.GetString("DB_PORT"),
		DatabaseUser:     v.GetString("DB_USER"),
		DatabasePassword: v.GetString("DB_PASSWORD"),
		DatabaseName:     v.GetString("DB_NAME"),
		DatabaseTimeout:  v.GetInt("DB_TIMEOUT"),
		MaxExpense:       v.GetInt64("MAX_EXPENSE"),
	}, nil
}

func connectDatabase(config *Config) (*sql.DB, error) {
	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DatabaseHost, config.DatabasePort, config.DatabaseUser, config.DatabasePassword, config.DatabaseName)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(0)
	db.SetConnMaxLifetime(time.Second * time.Duration(config.DatabaseTimeout))

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func getCurrentWeekExpense(ctx context.Context, db *sql.DB, weekData WeekData) (WeeklyExpense, error) {
	var expense WeeklyExpense
	query := `SELECT id, year, week, weekday, weekend, created_time FROM weekly_expense 
			  WHERE year = $1 AND week = $2 LIMIT 1`
	err := db.QueryRowContext(ctx, query, weekData.Year, weekData.Week).Scan(
		&expense.ID, &expense.Year, &expense.Week, &expense.Weekday, &expense.Weekend, &expense.CreatedTime)
	if err != nil {
		if err == sql.ErrNoRows {
			return insertCurrentWeekExpense(ctx, db, weekData)
		}
		return WeeklyExpense{}, fmt.Errorf("failed to get current week expense: %w", err)
	}
	return expense, nil
}

func insertCurrentWeekExpense(ctx context.Context, db *sql.DB, weekData WeekData) (WeeklyExpense, error) {
	var expense WeeklyExpense
	query := `INSERT INTO weekly_expense (year, week) 
			  VALUES ($1, $2)
			  RETURNING id, year, week, weekday, weekend, created_time`
	err := db.QueryRowContext(ctx, query, weekData.Year, weekData.Week).Scan(
		&expense.ID, &expense.Year, &expense.Week, &expense.Weekday, &expense.Weekend, &expense.CreatedTime)
	if err != nil {
		return WeeklyExpense{}, fmt.Errorf("failed to insert current week expense: %w", err)
	}
	return expense, nil
}

func formatRupiah(amount int64) string {
	str := fmt.Sprintf("%d", amount)
	n := len(str)
	if n <= 3 {
		return "Rp " + str
	}

	result := ""
	for i, digit := range str {
		if i > 0 && (n-i)%3 == 0 {
			result += "."
		}
		result += string(digit)
	}
	return "Rp " + result
}
