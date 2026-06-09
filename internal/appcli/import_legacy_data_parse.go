package appcli

import (
	"fmt"
	"regexp"
	"strings"
)

var insertTableRe = regexp.MustCompile(`(?is)^INSERT INTO\s+("?[\w\.]+"?)\s*\(`)

func parseLegacyInsertStatements(sqlText string) ([]legacyInsertStmt, error) {
	parts := splitSQLStatements(sqlText)
	out := make([]legacyInsertStmt, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s == "" || strings.HasPrefix(s, "--") || strings.HasPrefix(s, "/*") {
			continue
		}
		st, ok := parseLegacyInsertStatement(s)
		if !ok {
			continue
		}
		out = append(out, st)
	}
	return out, nil
}

func splitCSVRespectingQuotes(s string) []string {
	st := csvSplitState{}
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if st.consumeEscapedSingle(s, i) {
			i++
			continue
		}
		switch ch {
		case '\'':
			st.toggleSingle()
		case '"':
			st.toggleDouble()
		case '(':
			st.adjustDepth(1)
		case ')':
			st.adjustDepth(-1)
		case ',':
			if st.shouldSplit() {
				st.pushToken()
				continue
			}
		}
		st.builder.WriteByte(ch)
	}
	if strings.TrimSpace(st.builder.String()) != "" {
		st.pushToken()
	}
	return st.out
}

func (st *csvSplitState) consumeEscapedSingle(s string, i int) bool {
	if st.inSingle && !st.inDouble && s[i] == '\'' && i+1 < len(s) && s[i+1] == '\'' {
		st.builder.WriteString("''")
		return true
	}
	return false
}

func (st *csvSplitState) toggleSingle() {
	if !st.inDouble {
		st.inSingle = !st.inSingle
	}
}

func (st *csvSplitState) toggleDouble() {
	if !st.inSingle {
		st.inDouble = !st.inDouble
	}
}

func (st *csvSplitState) adjustDepth(delta int) {
	if st.inSingle || st.inDouble {
		return
	}
	if delta > 0 {
		st.depth += delta
		return
	}
	if st.depth > 0 {
		st.depth--
	}
}

func (st *csvSplitState) shouldSplit() bool {
	return !st.inSingle && !st.inDouble && st.depth == 0
}

func (st *csvSplitState) pushToken() {
	st.out = append(st.out, strings.TrimSpace(st.builder.String()))
	st.builder.Reset()
}

func normalizeSQLIdentifier(s string) string {
	t := strings.TrimSpace(s)
	t = strings.Trim(t, `"`)
	t = strings.TrimPrefix(strings.ToLower(t), "public.")
	return t
}

func parseLegacyInsertStatement(s string) (legacyInsertStmt, bool) {
	s = strings.TrimSpace(s)
	if idx := strings.Index(strings.ToUpper(s), "INSERT INTO"); idx > 0 {
		s = strings.TrimSpace(s[idx:])
	}
	loc := insertTableRe.FindStringSubmatchIndex(s)
	if loc == nil {
		return legacyInsertStmt{}, false
	}
	table := normalizeSQLIdentifier(s[loc[2]:loc[3]])
	colOpen := loc[1] - 1
	colClose, ok := findBalancedClosingParen(s, colOpen)
	if !ok {
		return legacyInsertStmt{}, false
	}
	cols := splitCSVRespectingQuotes(s[colOpen+1 : colClose])
	rest := strings.TrimSpace(s[colClose+1:])
	if !strings.HasPrefix(strings.ToUpper(rest), "VALUES") {
		return legacyInsertStmt{}, false
	}
	rest = strings.TrimSpace(rest[len("VALUES"):])
	if rest == "" || rest[0] != '(' {
		return legacyInsertStmt{}, false
	}
	valClose, ok := findBalancedClosingParen(rest, 0)
	if !ok {
		return legacyInsertStmt{}, false
	}
	vals := splitCSVRespectingQuotes(rest[1:valClose])
	if len(cols) != len(vals) {
		return legacyInsertStmt{}, false
	}
	for i := range cols {
		cols[i] = normalizeSQLIdentifier(cols[i])
		vals[i] = strings.TrimSpace(vals[i])
	}
	return legacyInsertStmt{Table: table, Columns: cols, Values: vals}, true
}

func findBalancedClosingParen(s string, openIdx int) (int, bool) {
	if openIdx < 0 || openIdx >= len(s) || s[openIdx] != '(' {
		return 0, false
	}
	st := csvSplitState{depth: 1}
	for i := openIdx + 1; i < len(s); i++ {
		if st.consumeEscapedSingle(s, i) {
			i++
			continue
		}
		switch s[i] {
		case '\'':
			st.toggleSingle()
		case '"':
			st.toggleDouble()
		case '(':
			st.adjustDepth(1)
		case ')':
			st.adjustDepth(-1)
			if st.depth == 0 {
				return i, true
			}
		}
	}
	return 0, false
}

func splitSQLStatements(sqlText string) []string {
	st := csvSplitState{}
	var b strings.Builder
	out := make([]string, 0, 256)
	for i := 0; i < len(sqlText); i++ {
		if st.consumeEscapedSingle(sqlText, i) {
			b.WriteString("''")
			i++
			continue
		}
		ch := sqlText[i]
		switch ch {
		case '\'':
			st.toggleSingle()
		case '"':
			st.toggleDouble()
		case ';':
			if st.shouldSplit() {
				out = append(out, b.String())
				b.Reset()
				continue
			}
		}
		b.WriteByte(ch)
	}
	if strings.TrimSpace(b.String()) != "" {
		out = append(out, b.String())
	}
	return out
}

func buildInsertRow(cols, vals []string) (map[string]any, error) {
	row := make(map[string]any, len(cols))
	for i, c := range cols {
		v, err := parseSQLLiteral(vals[i])
		if err != nil {
			return nil, fmt.Errorf("column %s: %w", c, err)
		}
		row[c] = v
	}
	return row, nil
}

func parseSQLLiteral(v string) (any, error) {
	s := strings.TrimSpace(v)
	if strings.EqualFold(s, "null") {
		return nil, nil
	}
	if strings.EqualFold(s, "true") {
		return true, nil
	}
	if strings.EqualFold(s, "false") {
		return false, nil
	}
	if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") && len(s) >= 2 {
		raw := s[1 : len(s)-1]
		raw = strings.ReplaceAll(raw, "''", "'")
		return raw, nil
	}
	return s, nil
}
