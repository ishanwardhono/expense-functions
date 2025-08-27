# GitHub Copilot Custom Instructions for Expense Function

## Project Overview
This is a Google Cloud Functions-based expense tracking application written in Go. The application manages both weekly and monthly expenses with PostgreSQL database storage.

## Architecture & Tech Stack
- **Language**: Go 1.21.3
- **Framework**: Google Cloud Functions Framework (`functions-framework-go`)
- **Database**: PostgreSQL with `sqlx` and `lib/pq` drivers
- **ID Generation**: UUID (`google/uuid`)
- **Deployment**: Google Cloud Functions
- **Testing**: Go standard testing package

## Project Structure
```
├── cmd/main.go                 # Local development server entry point
├── handler.go                  # HTTP function handlers registration
├── base.go                     # Base handler with error handling and CORS
├── debug.go                    # Debug utilities
├── common/                     # Shared utilities
│   ├── config.go              # Environment configuration
│   ├── db.go                  # Database connection and utilities
│   ├── currency.go            # Currency formatting utilities
│   └── time.go                # Time utilities and mocking
├── weekly/                    # Weekly expense management
│   ├── model.go               # Data models and business logic
│   ├── add.go                 # Add weekly expenses
│   ├── get.go                 # Retrieve weekly expenses
│   ├── db.go                  # Database operations
│   ├── config.go              # Configuration loading
│   └── utils.go               # Helper functions
├── monthly/                   # Monthly expense management
│   └── [similar structure to weekly/]
└── data/ddl.sql              # Database schema
```

## Code Conventions & Patterns

### Database Operations
- Use `sqlx` for database operations with struct scanning
- All database functions should accept `context.Context` as first parameter
- Use prepared statements and proper error handling
- Database models use struct tags: `db:"column_name"`

### Function Handlers
- All HTTP functions are registered in `handler.go` using `functions.HTTP()`
- Use `baseHandler()` wrapper for consistent error handling and CORS
- Request/Response models defined in respective package model files
- Proper JSON marshaling/unmarshaling for API communication

### Error Handling
- Always log errors with context using `log.Printf()`
- Return meaningful error messages
- Use proper HTTP status codes in base handler

### Environment Configuration
- Configuration loaded from environment variables in `common/config.go`
- Support for time mocking via `TIME` environment variable
- Expense limits configured via `MAX_EXPENSE` and `MAX_MONTHLY_EXPENSE`

### Testing
- Unit tests in `*_test.go` files
- Use Go standard testing package
- Test configuration, utilities, and database operations

## Key Business Logic

### Weekly Expenses
- Track expenses by year, week, and day (1=Monday to 7=Sunday)
- Separate tracking for weekday, Saturday, and Sunday expenses
- Expense limits enforced per day type

### Monthly Expenses
- Track expenses by year and month
- Aggregate totals and category breakdowns
- Monthly expense limits enforced

### Common Features
- UUID-based primary keys
- Timestamp tracking with `created_time`
- Expense categorization with `type` field
- Optional notes for each expense
- Currency formatting utilities
- Time zone handling and mocking support

## Development Guidelines

### When adding new features:
1. Follow the existing package structure (`weekly/` or `monthly/`)
2. Add models to `model.go` with proper struct tags
3. Implement database operations in `db.go`
4. Add business logic functions (add/get) in separate files
5. Register HTTP handlers in main `handler.go`
6. Add configuration loading if needed
7. Write unit tests for utilities and business logic

### Database Schema Considerations:
- Use appropriate PostgreSQL data types (INT2 for year/week/month, INT4 for amounts)
- Add proper indexes for query performance
- Use UUID for primary keys with `gen_random_uuid()`
- Include created_time with default CURRENT_TIMESTAMP

### API Design:
- RESTful endpoints with clear naming
- Proper HTTP methods and status codes
- Consistent JSON request/response formats
- Error responses with meaningful messages

## Common Patterns to Follow
- Configuration: Load from environment with validation
- Database: Context-aware operations with proper error handling  
- Models: Clear separation of data models and business logic
- Testing: Comprehensive unit tests for utilities and edge cases
- Logging: Structured logging with appropriate levels
- Error Handling: Graceful degradation with user-friendly messages

## Dependencies Management
- Keep `go.mod` clean and up-to-date
- Use semantic versioning for dependencies
- Minimal external dependencies for cloud function efficiency
