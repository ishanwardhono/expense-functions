package expensefunction

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/ishanwardhono/expense-function/hello"
	"github.com/ishanwardhono/expense-function/monthly"
	"github.com/ishanwardhono/expense-function/recap"
	"github.com/ishanwardhono/expense-function/weekly"
)

func init() {
	functions.HTTP("WeeklyGet", baseHandler(weeklyGet))
	functions.HTTP("WeeklyAdd", baseHandler(weeklyAdd))
	functions.HTTP("MonthlyGet", baseHandler(monthlyGet))
	functions.HTTP("MonthlyAdd", baseHandler(monthlyAdd))
	functions.HTTP("RecapGet", baseHandler(recapGet))
	functions.HTTP("Hello", baseHandler(helloFunc))
}

func weeklyGet(r *http.Request) (interface{}, error) {
	res, err := weekly.Get(r.Context())
	if err != nil {
		log.Printf("failed to get weekly expenses: %v", err)
		return nil, err
	}
	log.Printf("successfully retrieved weekly expenses")
	return res, nil
}

func weeklyAdd(r *http.Request) (interface{}, error) {
	var req weekly.AddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("failed to decode request body: %v", err)
		return nil, err
	}
	if err := weekly.Add(r.Context(), req); err != nil {
		log.Printf("failed to add weekly expense: %v", err)
		return nil, err
	}
	log.Printf("successfully added weekly expense")
	return nil, nil
}

func helloFunc(r *http.Request) (interface{}, error) {
	var req hello.HelloRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("failed to decode request body: %v", err)
		return nil, err
	}
	res, err := hello.Hello(r.Context(), req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func monthlyGet(r *http.Request) (interface{}, error) {
	res, err := monthly.Get(r.Context())
	if err != nil {
		log.Printf("failed to get monthly expenses: %v", err)
		return nil, err
	}
	log.Printf("successfully retrieved monthly expenses")
	return res, nil
}

func monthlyAdd(r *http.Request) (interface{}, error) {
	var req monthly.AddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("failed to decode request body: %v", err)
		return nil, err
	}
	if err := monthly.Add(r.Context(), req); err != nil {
		log.Printf("failed to add monthly expense: %v", err)
		return nil, err
	}
	log.Printf("successfully added monthly expense")
	return nil, nil
}

func recapGet(r *http.Request) (interface{}, error) {
	var req recap.GetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("failed to decode request body: %v", err)
		return nil, err
	}
	res, err := recap.Get(r.Context(), req)
	if err != nil {
		log.Printf("failed to get recap: %v", err)
		return nil, err
	}
	log.Printf("successfully retrieved recap")
	return res, nil
}
