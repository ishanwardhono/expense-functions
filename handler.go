package expensefunction

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/ishanwardhono/expense-function/weekly"
)

func init() {
	functions.HTTP("WeeklyGet", baseHandler(weeklyGet))
	functions.HTTP("WeeklyAdd", baseHandler(weeklyAdd))
	functions.HTTP("Hello", baseHandler(hello))
}

func weeklyGet(r *http.Request) (interface{}, error) {
	res, err := weekly.Get(r.Context())
	if err != nil {
		log.Printf("failed to get weekly expenses: %v", err)
		return nil, err
	}
	log.Printf("successfully retrieved weekly expenses: %v", res)
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
	log.Printf("successfully added weekly expense: %v", req)
	return nil, nil
}

func hello(r *http.Request) (interface{}, error) {
	var req weekly.HelloRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("failed to decode request body: %v", err)
		return nil, err
	}
	if err := weekly.Hello(r.Context(), req); err != nil {
		return nil, err
	}
	return nil, nil
}
