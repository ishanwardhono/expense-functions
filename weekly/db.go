package weekly

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

const collectionName = "expenses"

func connectDatabase(ctx context.Context, cfg *config) (*firestore.Client, error) {
	client, err := firestore.NewClientWithDatabase(ctx, cfg.projectID, cfg.databaseName, option.WithCredentialsFile("firebase-key.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to create firestore client: %w", err)
	}
	return client, nil
}

func getCurrentWeekExpense(ctx context.Context, client *firestore.Client, weekData WeekData) (WeeklyExpense, error) {
	collectionRef := client.Collection(collectionName)

	// Query for existing document with the same year and week
	query := collectionRef.Where("year", "==", weekData.year).Where("week", "==", weekData.week).Limit(1)
	iter := query.Documents(ctx)

	doc, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			// No document found, create a new one
			return insertCurrentWeekExpense(ctx, client, weekData)
		}
		return WeeklyExpense{}, fmt.Errorf("failed to get current week expense: %w", err)
	}

	var expense WeeklyExpense
	if err := doc.DataTo(&expense); err != nil {
		return WeeklyExpense{}, fmt.Errorf("failed to convert document to WeeklyExpense: %w", err)
	}

	expense.Id = doc.Ref.ID
	return expense, nil
}

func insertCurrentWeekExpense(ctx context.Context, client *firestore.Client, weekData WeekData) (WeeklyExpense, error) {
	collectionRef := client.Collection(collectionName)

	expense := WeeklyExpense{
		Year:        weekData.year,
		Week:        weekData.week,
		Weekday:     0,
		Weekend:     0,
		CreatedTime: time.Now(),
	}

	docRef, _, err := collectionRef.Add(ctx, expense)
	if err != nil {
		return WeeklyExpense{}, fmt.Errorf("failed to insert current week expense: %w", err)
	}

	expense.Id = docRef.ID
	return expense, nil
}
