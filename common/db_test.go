package common

import (
	"os"
	"testing"
)

func TestLoadDatabaseConfig(t *testing.T) {
	// Save original values to restore later
	originalValues := map[string]string{
		"DB_HOST":          os.Getenv("DB_HOST"),
		"DB_PORT":          os.Getenv("DB_PORT"),
		"DB_USER":          os.Getenv("DB_USER"),
		"DB_PASSWORD":      os.Getenv("DB_PASSWORD"),
		"DB_NAME":          os.Getenv("DB_NAME"),
		"DB_SSL_ROOT_CERT": os.Getenv("DB_SSL_ROOT_CERT"),
	}

	defer func() {
		for key, value := range originalValues {
			os.Setenv(key, value)
		}
	}()

	// Set test values
	testValues := map[string]string{
		"DB_HOST":          "localhost",
		"DB_PORT":          "5432",
		"DB_USER":          "test_user",
		"DB_PASSWORD":      "test_password",
		"DB_NAME":          "test_db",
		"DB_SSL_ROOT_CERT": "/path/to/cert",
	}

	for key, value := range testValues {
		os.Setenv(key, value)
	}

	config := LoadDatabaseConfig()

	if config.Host != testValues["DB_HOST"] {
		t.Errorf("Expected Host %s, got %s", testValues["DB_HOST"], config.Host)
	}
	if config.Port != testValues["DB_PORT"] {
		t.Errorf("Expected Port %s, got %s", testValues["DB_PORT"], config.Port)
	}
	if config.User != testValues["DB_USER"] {
		t.Errorf("Expected User %s, got %s", testValues["DB_USER"], config.User)
	}
	if config.Password != testValues["DB_PASSWORD"] {
		t.Errorf("Expected Password %s, got %s", testValues["DB_PASSWORD"], config.Password)
	}
	if config.Name != testValues["DB_NAME"] {
		t.Errorf("Expected Name %s, got %s", testValues["DB_NAME"], config.Name)
	}
	if config.SSLRootCert != testValues["DB_SSL_ROOT_CERT"] {
		t.Errorf("Expected SSLRootCert %s, got %s", testValues["DB_SSL_ROOT_CERT"], config.SSLRootCert)
	}
}

// Note: ConnectDatabase test requires a real database connection
// which we skip for unit tests. Integration tests would cover this.
