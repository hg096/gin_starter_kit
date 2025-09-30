package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Level 로그 레벨
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

var (
	levelNames = map[Level]string{
		DEBUG: "DEBUG",
		INFO:  "INFO",
		WARN:  "WARN",
		ERROR: "ERROR",
		FATAL: "FATAL",
	}

	levelColors = map[Level]string{
		DEBUG: "\033[36m", // Cyan
		INFO:  "\033[32m", // Green
		WARN:  "\033[33m", // Yellow
		ERROR: "\033[31m", // Red
		FATAL: "\033[35m", // Magenta
	}

	reset = "\033[0m"
)

// Logger 구조체
type Logger struct {
	level      Level
	useColor   bool
	timeFormat string
}

var defaultLogger *Logger

func init() {
	defaultLogger = &Logger{
		level:      INFO,
		useColor:   true,
		timeFormat: "2006-01-02 15:04:05",
	}
}

// SetLevel 로그 레벨 설정
func SetLevel(level Level) {
	defaultLogger.level = level
}

// SetLevelFromString 문자열로 로그 레벨 설정
func SetLevelFromString(levelStr string) {
	switch levelStr {
	case "debug":
		SetLevel(DEBUG)
	case "info":
		SetLevel(INFO)
	case "warn":
		SetLevel(WARN)
	case "error":
		SetLevel(ERROR)
	case "fatal":
		SetLevel(FATAL)
	default:
		SetLevel(INFO)
	}
}

// log 내부 로깅 함수
func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	timestamp := time.Now().Format(l.timeFormat)
	levelName := levelNames[level]
	message := fmt.Sprintf(format, args...)

	var output string
	if l.useColor {
		color := levelColors[level]
		output = fmt.Sprintf("%s [%s]%s %s - %s",
			timestamp, color+levelName+reset, reset, message, "")
	} else {
		output = fmt.Sprintf("%s [%s] %s", timestamp, levelName, message)
	}

	if level == FATAL {
		log.Fatal(output)
		os.Exit(1)
	} else {
		log.Println(output)
	}
}

// Debug 디버그 로그
func Debug(format string, args ...interface{}) {
	defaultLogger.log(DEBUG, format, args...)
}

// Info 정보 로그
func Info(format string, args ...interface{}) {
	defaultLogger.log(INFO, format, args...)
}

// Warn 경고 로그
func Warn(format string, args ...interface{}) {
	defaultLogger.log(WARN, format, args...)
}

// Error 에러 로그
func Error(format string, args ...interface{}) {
	defaultLogger.log(ERROR, format, args...)
}

// Fatal 치명적 에러 로그 (프로그램 종료)
func Fatal(format string, args ...interface{}) {
	defaultLogger.log(FATAL, format, args...)
}

// WithField 필드와 함께 로그
func WithField(key string, value interface{}) *LogEntry {
	return &LogEntry{
		fields: map[string]interface{}{key: value},
	}
}

// WithFields 여러 필드와 함께 로그
func WithFields(fields map[string]interface{}) *LogEntry {
	return &LogEntry{fields: fields}
}

// LogEntry 필드를 포함한 로그 엔트리
type LogEntry struct {
	fields map[string]interface{}
}

// Debug 필드와 함께 디버그 로그
func (e *LogEntry) Debug(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	Debug("%s %v", message, e.fields)
}

// Info 필드와 함께 정보 로그
func (e *LogEntry) Info(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	Info("%s %v", message, e.fields)
}

// Warn 필드와 함께 경고 로그
func (e *LogEntry) Warn(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	Warn("%s %v", message, e.fields)
}

// Error 필드와 함께 에러 로그
func (e *LogEntry) Error(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	Error("%s %v", message, e.fields)
}

// Fatal 필드와 함께 치명적 에러 로그
func (e *LogEntry) Fatal(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	Fatal("%s %v", message, e.fields)
}