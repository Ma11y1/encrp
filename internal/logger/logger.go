package logger

import (
	"errors"
	"fmt"
	"io"
	"time"
)

var outs = make([]io.Writer, 0)
var outsErr = make([]io.Writer, 0)

func AddOutput(writer io.Writer) error {
	if writer == nil {
		return errors.New("Logger.AddWriter() writer is nil")
	}
	for _, w := range outs {
		if w == writer {
			return errors.New("Logger.AddWriter() writer already exists")
		}
	}
	outs = append(outs, writer)
	return nil
}

func RemoveOutput(writer io.Writer) {
	if writer == nil {
		return
	}
	for i, w := range outs {
		if w == writer {
			outs = append(outs[:i], outs[i+1:]...)
			return
		}
	}
}

func AddOutputErr(writer io.Writer) error {
	if writer == nil {
		return errors.New("Logger.AddWriterErr() writer is nil")
	}
	for _, o := range outsErr {
		if o == writer {
			return errors.New("Logger.AddWriterErr() writer already exists")
		}
	}
	outsErr = append(outsErr, writer)
	return nil
}

func RemoveOutputErr(writer io.Writer) {
	if writer == nil {
		return
	}
	for i, o := range outsErr {
		if o == writer {
			outsErr = append(outsErr[:i], outsErr[i+1:]...)
			return
		}
	}
}

func Info(location, msg string) {
	log(outs, "INFO", location, msg)
}

func Infof(location, format string, args ...interface{}) {
	logf(outs, "INFO", location, format, args...)
}

func Err(location, msg string) {
	log(outsErr, "ERROR", location, msg)
}

func Errf(location, format string, args ...interface{}) {
	logf(outsErr, "ERROR", location, format, args...)
}

func Warn(location, msg string) {
	log(outsErr, "WARN", location, msg)
}

func Warnf(location, format string, args ...interface{}) {
	logf(outsErr, "WARN", location, format, args...)
}

func Debug(location, msg string) {
	log(outs, "DEBUG", location, msg)
}

func Debugf(location, format string, args ...interface{}) {
	logf(outs, "DEBUG", location, format, args...)
}

func log(outs []io.Writer, prefix, location, msg string) {
	for _, out := range outs {
		fmt.Fprintf(out, "%s: [%s] %s: %s\n", time.Now().Format("2006.01.02 15:04:05 -0700"), prefix, location, msg)
	}
}

func logf(outs []io.Writer, prefix, location, format string, args ...interface{}) {
	for _, out := range outs {
		fmt.Fprintf(out, "%s: [%s] %s: %s\n", time.Now().Format("2006.01.02 15:04:05 -0700"), prefix, location, fmt.Sprintf(format, args...))
	}
}
