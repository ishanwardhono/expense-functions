package envelope

import "time"

// SubStatus is the read-only payment status of a subscription for a month
// (spec §6.4), derived from this month's single linked Langganan expense.
type SubStatus string

const (
	StatusPaid   SubStatus = "paid"
	StatusUnpaid SubStatus = "unpaid"
)

// SubPayment is the per-subscription payment summary for a month.
type SubPayment struct {
	Paid       bool
	PaidDate   time.Time
	PaidAmount int64
	Diff       int64 // alloc − paid
	Status     SubStatus
}

// SubscriptionStatus returns the payment status of sub for (year, month),
// derived from the at-most-one linked Langganan expense in that calendar month.
func SubscriptionStatus(sub Subscription, expenses []Expense, year, month int) SubPayment {
	for _, e := range expenses {
		if e.Category != CatLangganan || e.SubscriptionID != sub.ID {
			continue
		}
		if !inMonth(e.Date, year, month) {
			continue
		}
		return SubPayment{
			Paid:       true,
			PaidDate:   e.Date,
			PaidAmount: e.Amount,
			Diff:       sub.Alloc - e.Amount,
			Status:     StatusPaid,
		}
	}
	return SubPayment{Status: StatusUnpaid}
}
