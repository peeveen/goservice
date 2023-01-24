package goservice

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kardianos/service"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Our logger. Initially logs to console, but our code will later
// "mute" the console output after some hooks are added.
var logger *logrus.Logger = createLogger(os.Stderr, logrus.DebugLevel)

func createLogger(out io.Writer, level logrus.Level) *logrus.Logger {
	return &logrus.Logger{
		Out:       out,
		Formatter: &logrus.TextFormatter{},
		Hooks:     make(logrus.LevelHooks),
		Level:     level,
	}
}

func getLoggerHookLevels(maximumLogLevel logrus.Level) []logrus.Level {
	levels := make([]logrus.Level, 0)
	for _, lvl := range logrus.AllLevels {
		if maximumLogLevel >= lvl {
			levels = append(levels, lvl)
		}
	}
	return levels
}

// NewServiceLoggerHook creates a logger hook for the operating system logging facility
func NewServiceLoggerHook(svcLogger service.Logger, includeDebug bool) logrus.Hook {
	var levels []logrus.Level
	if includeDebug {
		levels = getLoggerHookLevels(logrus.DebugLevel)
	} else {
		levels = getLoggerHookLevels(logrus.InfoLevel)
	}
	return &serviceLoggerHook{svcLogger, levels}
}

type serviceLoggerHook struct {
	svcLogger service.Logger
	levels    []logrus.Level
}

func (s *serviceLoggerHook) Levels() []logrus.Level {
	return s.levels
}
func (s *serviceLoggerHook) Fire(e *logrus.Entry) error {
	mthd := service.Logger.Error
	if e.Level == logrus.WarnLevel {
		mthd = service.Logger.Warning
	} else if e.Level >= logrus.InfoLevel {
		mthd = service.Logger.Info
	}
	msg := e.Message
	if len(e.Data) > 0 {
		msg = fmt.Sprintf("%s (%s)", msg, getFieldsAsString(e.Data))
	}
	// Don't need to log time/severity, etc ... all handled by service logging framework.
	return mthd(s.svcLogger, msg)
}

func getCurrentDirectory() (string, error) {
	// On Windows, running as a service reports current directory
	// as "C:\Windows\system32", so have to do this ...
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exePath), nil
}

// NewFileLoggerHook creates a logger hook for logging to files.
func NewFileLoggerHook(filenameOrPath string, asJSON bool, maximumLogLevel logrus.Level, maxLogFileSizeMegabytes int, maxLogAgeDays int) (logrus.Hook, error) {
	makePathAbsolute := func(path string, workingFolder string) string {
		if filepath.IsAbs(path) {
			return path
		}
		return filepath.Join(workingFolder, path)
	}

	currentDirectory, err := getCurrentDirectory()
	if err != nil {
		return nil, err
	}

	if currentDirectory != "" {
		filenameOrPath = makePathAbsolute(filenameOrPath, currentDirectory)
	}

	lumberjackLogger := &lumberjack.Logger{
		Filename: filenameOrPath,
		MaxSize:  maxLogFileSizeMegabytes,
		MaxAge:   maxLogAgeDays,
	}

	var formatter logrus.Formatter
	if asJSON {
		formatter = &logrus.JSONFormatter{}
	} else {
		formatter = &logrus.TextFormatter{}
	}

	return &fileLoggerHook{lumberjackLogger, formatter, getLoggerHookLevels(maximumLogLevel)}, nil
}

type fileLoggerHook struct {
	fileLogger *lumberjack.Logger
	formatter  logrus.Formatter
	levels     []logrus.Level
}

func (f *fileLoggerHook) Levels() []logrus.Level {
	return f.levels
}
func (f *fileLoggerHook) Fire(e *logrus.Entry) error {
	str, err := f.formatter.Format(e)
	if err != nil {
		return err
	}
	_, err = f.fileLogger.Write([]byte(str))
	if err != nil {
		return err
	}
	return nil
}

func getFieldsAsString(fields logrus.Fields) string {
	pairs := make([]string, 0)
	for k, v := range fields {
		pairs = append(pairs, fmt.Sprintf("%s=%v", k, v))
	}
	return strings.Join(pairs, ", ")
}
