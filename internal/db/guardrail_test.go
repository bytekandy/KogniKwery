package db

import (
	"strings"
	"testing"
)

func TestValidateAndNormalizeSQL_ValidQueries(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{
			input:    "SELECT * FROM sales;",
			expected: "SELECT * FROM sales LIMIT 1000",
		},
		{
			input:    "-- comment\nWITH cte AS (SELECT id FROM users) SELECT * FROM cte LIMIT 10;",
			expected: "WITH cte AS (SELECT id FROM users) SELECT * FROM cte LIMIT 10",
		},
	}

	for _, c := range cases {
		got, err := ValidateAndNormalizeSQL(c.input, 1000)
		if err != nil {
			t.Fatalf("expected valid SQL for %q, got error: %v", c.input, err)
		}
		if got != c.expected {
			t.Errorf("expected %q, got %q", c.expected, got)
		}
	}
}

func TestValidateAndNormalizeSQL_BlockedQueries(t *testing.T) {
	malicious := []string{
		"DROP TABLE sales;",
		"DELETE FROM users WHERE 1=1;",
		"SELECT * FROM users; DROP TABLE users;",
		"/* bypass */ UPDATE users SET role = 'admin'",
		"COPY users TO '/tmp/leak.csv'",
	}

	for _, sql := range malicious {
		_, err := ValidateAndNormalizeSQL(sql, 1000)
		if err == nil {
			t.Errorf("expected guardrail to block malicious query %q, but it passed!", sql)
		} else if !strings.Contains(err.Error(), "guardrail blocked") {
			t.Errorf("unexpected error message: %v", err)
		}
	}
}
