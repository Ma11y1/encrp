package errors

import (
	"encrp/internal/logger"
	"fmt"
)

func New(location, msg string) error {
	logger.Err(location, msg)
	return fmt.Errorf("[%s] %s", location, msg)
}

func Newf(location, format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	logger.Err(location, msg)
	return fmt.Errorf("[%s] %s", location, msg)
}

func Wrap(err error, location, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("[%s] %s: %w", location, msg, err)
}

func Wrapf(err error, location, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("[%s] %s: %w", location, fmt.Sprintf(format, args...), err)
}

func WrapLog(err error, location, msg string) error {
	if err == nil {
		return nil
	}
	logger.Err(location, fmt.Sprintf("%s: %s", msg, err))
	return fmt.Errorf("[%s] %s: %w", location, msg, err)
}

func WrapfLog(err error, location, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	msg := fmt.Sprintf(format, args...)
	logger.Err(location, fmt.Sprintf("%s: %s", msg, err))
	return fmt.Errorf("[%s] %s: %w", location, msg, err)
}
