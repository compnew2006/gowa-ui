package handlers_test

import "github.com/zerodha/logf"

func createTestLogger() logf.Logger {
	return logf.New(logf.Opts{Level: logf.ErrorLevel})
}
