package db

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	blockCommentRe = regexp.MustCompile(`(?s)/\*.*?\*/`)
	lineCommentRe  = regexp.MustCompile(`(?m)--.*$`)
	whitespaceRe   = regexp.MustCompile(`\s+`)
	forbiddenRe    = regexp.MustCompile(`(?i)\b(DROP|DELETE|UPDATE|INSERT|ALTER|TRUNCATE|CREATE|REPLACE|GRANT|REVOKE|ATTACH|DETACH|COPY|EXPORT|CALL|EXECUTE|PRAGMA)\b`)
	limitRe        = regexp.MustCompile(`(?i)\bLIMIT\s+\d+`)
)

// ValidateAndNormalizeSQL strips comments, enforces single-statement read-only SELECT/WITH rules,
// and appends a default LIMIT if none exists.
func ValidateAndNormalizeSQL(rawSQL string, defaultLimit int) (string, error) {
	// 1. Strip block and line comments
	cleaned := blockCommentRe.ReplaceAllString(rawSQL, " ")
	cleaned = lineCommentRe.ReplaceAllString(cleaned, " ")
	cleaned = strings.TrimSpace(cleaned)

	// 2. Trim trailing semicolons and reject multi-statement queries
	cleaned = strings.TrimRight(cleaned, "; \t\n\r")
	if strings.Contains(cleaned, ";") {
		return "", errors.New("guardrail blocked: multi-statement SQL queries are not allowed")
	}

	if cleaned == "" {
		return "", errors.New("guardrail blocked: empty SQL query")
	}

	// 3. Normalize whitespace
	normalized := whitespaceRe.ReplaceAllString(cleaned, " ")
	upper := strings.ToUpper(normalized)

	// 4. Must start with SELECT, WITH, DESCRIBE, or SHOW
	if !strings.HasPrefix(upper, "SELECT ") &&
		!strings.HasPrefix(upper, "WITH ") &&
		!strings.HasPrefix(upper, "DESCRIBE ") &&
		!strings.HasPrefix(upper, "SHOW ") {
		return "", fmt.Errorf("guardrail blocked: query must begin with SELECT or WITH (got: %s)", strings.Split(normalized, " ")[0])
	}

	// 5. Block any destructive DDL/DML or file-write keywords
	if match := forbiddenRe.FindString(upper); match != "" {
		return "", fmt.Errorf("guardrail blocked: forbidden keyword '%s' detected", strings.ToUpper(match))
	}

	// 6. Auto-append LIMIT for SELECT/WITH queries if not already present
	if defaultLimit > 0 && (strings.HasPrefix(upper, "SELECT ") || strings.HasPrefix(upper, "WITH ")) {
		if !limitRe.MatchString(upper) {
			normalized = fmt.Sprintf("%s LIMIT %d", normalized, defaultLimit)
		}
	}

	return normalized, nil
}
