package gormx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

const (
	sqlLogFastThreshold  = 200 * time.Millisecond
	sqlLogSlowThreshold  = 2 * time.Second
	sqlLogStdoutPrefix   = "\r\n"
	sqlLogStdoutFlags    = log.LstdFlags
	sqlLogRowsUnknown    = "-"
	sqlLogDurationFormat = "%.3fms"
)

// NewSQLLogger returns a GORM logger that prints every SQL statement with elapsed
// time in milliseconds. SQL text is color-coded on TTY: green <200ms, yellow
// 200ms–2s, red >=2s (reuses gorm.io/gorm/logger ANSI constants).
func NewSQLLogger() gormlogger.Interface {
	return newSQLLogger(os.Stdout, true)
}

func newSQLLogger(out io.Writer, colorful bool) gormlogger.Interface {
	return &sqlLogger{
		Writer: log.New(out, sqlLogStdoutPrefix, sqlLogStdoutFlags),
		Config: gormlogger.Config{
			LogLevel:                  gormlogger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  colorful,
		},
	}
}

type sqlLogger struct {
	gormlogger.Writer
	gormlogger.Config
}

func (l *sqlLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	next := *l
	next.LogLevel = level
	return &next
}

func (l *sqlLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.logLevelMessage(gormlogger.Info, gormlogger.Green, gormlogger.Green+"[info] "+msg, data...)
}

func (l *sqlLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.logLevelMessage(gormlogger.Warn, gormlogger.BlueBold, gormlogger.Magenta+"[warn] "+msg, data...)
}

func (l *sqlLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.logLevelMessage(gormlogger.Error, gormlogger.Magenta, gormlogger.Red+"[error] "+msg, data...)
}

func (l *sqlLogger) logLevelMessage(minLevel gormlogger.LogLevel, headColor, body string, data ...interface{}) {
	if l.LogLevel < minLevel {
		return
	}
	l.Printf(headColor+"%s\n"+gormlogger.Reset+body+gormlogger.Reset,
		append([]interface{}{utils.FileWithLineNum()}, data...)...)
}

func (l *sqlLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	ms := float64(elapsed.Nanoseconds()) / 1e6
	sql, rows := fc()
	rowsLabel := sqlLogRowsUnknown
	if rows != -1 {
		rowsLabel = strconv.FormatInt(rows, 10)
	}

	file := utils.FileWithLineNum()
	sqlColor, reset := sqlDurationStyle(elapsed, l.Colorful)

	if err != nil && l.LogLevel >= gormlogger.Error &&
		(!errors.Is(err, gormlogger.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError) {
		l.printTraceLine(file, err, ms, rowsLabel, sqlColor, sql, reset, true)
		return
	}

	if l.LogLevel >= gormlogger.Info {
		l.printTraceLine(file, nil, ms, rowsLabel, sqlColor, sql, reset, false)
	}
}

func (l *sqlLogger) printTraceLine(
	file string,
	err error,
	ms float64,
	rowsLabel string,
	sqlColor string,
	sql string,
	reset string,
	withError bool,
) {
	if l.Colorful {
		if withError {
			l.Printf(
				"%s %v\n"+gormlogger.Yellow+"["+sqlLogDurationFormat+"] "+gormlogger.BlueBold+"[rows:"+rowsLabel+"]"+reset+" %s%s%s",
				file, err, ms, sqlColor, sql, reset,
			)
			return
		}
		l.Printf(
			"%s\n"+gormlogger.Yellow+"["+sqlLogDurationFormat+"] "+gormlogger.BlueBold+"[rows:"+rowsLabel+"]"+reset+" %s%s%s",
			file, ms, sqlColor, sql, reset,
		)
		return
	}

	if withError {
		l.Printf(
			"%s %v\n["+sqlLogDurationFormat+"] [rows:"+rowsLabel+"] %s",
			file, err, ms, sql,
		)
		return
	}
	l.Printf(
		"%s\n["+sqlLogDurationFormat+"] [rows:"+rowsLabel+"] %s",
		file, ms, sql,
	)
}

func sqlDurationStyle(elapsed time.Duration, colorful bool) (color, reset string) {
	if !colorful {
		return "", ""
	}
	switch {
	case elapsed >= sqlLogSlowThreshold:
		return gormlogger.Red, gormlogger.Reset
	case elapsed >= sqlLogFastThreshold:
		return gormlogger.Yellow, gormlogger.Reset
	default:
		return gormlogger.Green, gormlogger.Reset
	}
}

// SQLLogThresholds returns the latency buckets used for SQL console coloring.
func SQLLogThresholds() (fast, slow time.Duration) {
	return sqlLogFastThreshold, sqlLogSlowThreshold
}

// SQLLogThresholdSummary is a human-readable description of SQL log colors.
func SQLLogThresholdSummary() string {
	return fmt.Sprintf(
		"green <%v, yellow %v–%v, red >=%v",
		sqlLogFastThreshold,
		sqlLogFastThreshold,
		sqlLogSlowThreshold,
		sqlLogSlowThreshold,
	)
}
