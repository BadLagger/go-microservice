package utils

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

type LogLevel byte

const (
	Off   LogLevel = 0
	Debug LogLevel = 1 << iota
	Trace
	Info
	Error
	Critical
	All LogLevel = Debug | Trace | Info | Error | Critical
)

func (l LogLevel) String() string {

	if l == Off {
		return "OFF"
	}

	if l == All {
		return "ALL"
	}

	var strParts []string

	if l&Debug != 0 {
		strParts = append(strParts, "DEBUG")
	}

	if l&Trace != 0 {
		strParts = append(strParts, "TRACE")
	}

	if l&Info != 0 {
		strParts = append(strParts, "INFO")
	}

	if l&Error != 0 {
		strParts = append(strParts, "ERROR")
	}

	if l&Critical != 0 {
		strParts = append(strParts, "CRITICAL")
	}

	if len(strParts) == 0 {
		return "UNKNOWN"
	}

	return strings.Join(strParts, " | ")
}

type LogEntry struct {
	Time    time.Time
	Level   LogLevel
	Message string
	Args    []any
}

type Logger struct {
	level    LogLevel
	mtx      sync.RWMutex
	async    bool
	logChan  chan LogEntry  // Канал для асинхронной обработки
	stopChan chan struct{}  // Канал для остановки
	wg       sync.WaitGroup // Для graceful shutdown
}

func NewLogger() *Logger {
	return &Logger{
		level: Info,
		async: false,
	}
}

func NewAsyncLogger(level LogLevel, bufferSize int) *Logger {

	logger := &Logger{
		level:    level,
		logChan:  make(chan LogEntry, bufferSize), // Буферизированный канал
		stopChan: make(chan struct{}),
		async:    true,
	}

	logger.wg.Add(1)
	go logger.processLogs()

	return logger
}

// Worker для обработки логов
func (l *Logger) processLogs() {
	defer l.wg.Done()

	for {
		select {
		case <-l.stopChan:
			// Обрабатываем оставшиеся логи перед выходом
			for {
				select {
				case entry := <-l.logChan:
					l.writeLog(entry)
				default:
					return
				}
			}
		case entry := <-l.logChan:
			l.writeLog(entry)
		}
	}
}

// Синхронная запись лога (только здесь!)
func (l *Logger) writeLog(entry LogEntry) {
	timestamp := entry.Time.Format("2006-01-02 15:04:05.000")
	formattedMsg := fmt.Sprintf(entry.Message, entry.Args...)
	log.Printf("[%s] [%s]: %s", timestamp, entry.Level.String(), formattedMsg)
}

func (l *Logger) SetLevel(level LogLevel) *Logger {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	l.level = level
	return l
}

func (l *Logger) GetLevel() LogLevel {
	l.mtx.RLock()
	defer l.mtx.RUnlock()
	return l.level
}

func (l *Logger) Log(level LogLevel, message string, args ...any) {
	l.mtx.RLock()
	shouldLog := (l.GetLevel() != Off && l.GetLevel() <= level) || (l.GetLevel() == All)
	l.mtx.RUnlock()

	if !l.async {
		if shouldLog {
			log.Printf("["+level.String()+"]: "+message, args...)
		}
	} else {
		if !shouldLog {
			return
		}

		select {
		case l.logChan <- LogEntry{
			Time:    time.Now(),
			Level:   level,
			Message: message,
			Args:    args,
		}:
			// Успешно отправлено
		default:
			// Если канал переполнен, пишем напрямую (редкий случай)
			log.Printf("[WARNING]: Log buffer overflow: %s", fmt.Sprintf(message, args...))
		}
	}
}

func (l *Logger) Debug(message string, args ...any) {
	l.Log(Debug, message, args...)
}

func (l *Logger) Info(message string, args ...any) {
	l.Log(Info, message, args...)
}

func (l *Logger) Trace(message string, args ...any) {
	l.Log(Trace, message, args...)
}

func (l *Logger) Error(message string, args ...any) {
	l.Log(Error, message, args...)
}

func (l *Logger) Critical(message string, args ...any) {
	l.Log(Critical, message, args...)
}

var (
	instance *Logger
	once     sync.Once
)

func GlobalLogger() *Logger {
	once.Do(func() {
		//instance = NewLogger()
		instance = NewAsyncLogger(Info, 10000)
	})
	return instance
}
