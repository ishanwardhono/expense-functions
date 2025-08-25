# Monthly Expense Module Implementation

## Summary

Successfully created a new monthly expense module following the same pattern as the weekly module, and refactored common functionality into the shared common package.

## Changes Made

### 1. Reorganized Common Package

The common package has been organized into separate files based on functionality:

#### `common/common.go`
- Main package declaration and documentation

#### `common/config.go`
- `LoadMaxExpense()` - Load MAX_EXPENSE environment variable
- `LoadTime()` - Load TIME environment variable with fallback to current time
- Configuration loading functions

#### `common/db.go`  
- `DatabaseConfig` struct for database configuration
- `LoadDatabaseConfig()` - Load database configuration from environment variables
- `ConnectDatabase()` - Establish database connection with proper settings

#### `common/time.go`
- `Loc` variable for Asia/Jakarta timezone
- `Now()` - Get current time in Jakarta timezone

#### `common/currency.go`
- `FormatRupiah()` - Format numbers as Indonesian Rupiah currency

#### Test Files
- `common/config_test.go` - Tests for configuration functions
- `common/db_test.go` - Tests for database configuration
- `common/time_test.go` - Tests for time functions
- `common/currency_test.go` - Tests for currency formatting

### 2. Created Monthly Module Files

#### `monthly/model.go`
- `MonthData` struct for year/month tracking
- `MonthlyExpense` struct matching the `monthly_expense` table schema
- `MonthlyExpenses` slice with helper methods
- Response structures: `expenseResponse`, `expenseRemaining`, `expenseDetail`, `dataLabel`
- `AddRequest` struct for adding new expenses
- Business logic methods like `GetTotalExpense()`, `ToDetailsResponse()`, etc.

#### `monthly/utils.go`
- `getMonthData()` function to extract month/year from time
- `getMonthName()` function to convert month number to name
- Month names mapping
- Kept legacy payroll period functions for compatibility

#### `monthly/config.go`
- Configuration loading using common package functions
- Simplified config struct using shared database config

#### `monthly/db.go`
- Database connection using common package
- `getCurrentMonthExpense()` function to fetch monthly expenses
- `addMonthlyExpense()` function to insert new expenses

#### `monthly/get.go`
- `Get()` function to retrieve current month expenses
- `calculateRemainingExpense()` function for expense calculations
- Returns structured response with remaining budget and expense details

#### `monthly/add.go`
- `Add()` function to add new monthly expenses
- Date parsing and validation
- Integration with database layer

#### `monthly/hello.go`
- Simple hello function for testing module integration

#### `monthly/utils_test.go`
- Comprehensive tests for monthly utility functions
- Tests for data transformation and calculation logic

### 3. Refactored Weekly Module
Updated weekly module to use common package functions:

#### `weekly/config.go`
- Simplified to use common configuration functions
- Removed duplicate database configuration

#### `weekly/db.go`
- Updated to use common database connection function
- Cleaner, more maintainable code

#### `weekly/utils.go`
- Updated to use `common.FormatRupiah()` and `common.Now()`
- Removed duplicate rupiah formatting function

#### `weekly/add.go`
- Updated to use common timezone handling
- Improved date parsing with proper error handling

## Database Schema Support

The implementation supports the `monthly_expense` table from `ddl.sql`:
```sql
CREATE TABLE public.monthly_expense (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    year INT2 NOT NULL,
    month INT2 NOT NULL,
    amount INT4 NOT NULL DEFAULT 0,
    type VARCHAR(20) NOT NULL,
    note STRING NOT NULL DEFAULT '',
    created_time TIMESTAMP NOT NULL DEFAULT current_timestamp(),
    CONSTRAINT expense_pk PRIMARY KEY (id ASC),
    INDEX expense_year_week_day_idx (year ASC, month ASC),
    INDEX expense_created_time_idx (created_time DESC),
    INDEX expense_type_idx (type ASC)
);
```

## API Usage

### Get Monthly Expenses
```go
response, err := monthly.Get(ctx)
// Returns current month expenses with remaining budget calculation
```

### Add Monthly Expense
```go
request := monthly.AddRequest{
    Amount: 50000,
    Type:   "food",
    Note:   "Lunch",
    Date:   &"2024-08-24", // optional
}
err := monthly.Add(ctx, request)
```

## Features

1. **Monthly Budget Tracking**: Track expenses by month with remaining budget calculations
2. **Expense Categorization**: Support for expense types and notes
3. **Date Flexibility**: Add expenses for specific dates or current time
4. **Formatted Currency**: Proper Rupiah formatting with thousand separators
5. **Color-coded Status**: Visual indicators for budget status (green/red)
6. **Comprehensive Testing**: Unit tests for all utility functions
7. **Shared Code**: Common functionality extracted to shared package

## Testing

All tests pass successfully:
- Common package: FormatRupiah function tests
- Monthly package: Utils, model, and data transformation tests  
- Weekly package: All existing tests still pass after refactoring

The implementation follows Go best practices and maintains consistency with the existing weekly module pattern while adding monthly-specific functionality.
