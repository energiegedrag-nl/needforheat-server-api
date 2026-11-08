package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat"
	"github.com/sirupsen/logrus"
)

// A HandlerError contains information about an error that occured inside a [Handler].
type HandlerError struct {
	Err             error
	ResponseMessage string
	ResponseCode    int
	LogMessage      string
	LogLevel        logrus.Level
}

// Create a new HandlerError.
// The log level will be set to info.
func NewHandlerError(err error, responseMessage string, responseCode int) *HandlerError {
	return &HandlerError{
		Err:             err,
		ResponseMessage: responseMessage,
		ResponseCode:    responseCode,
		LogLevel:        logrus.InfoLevel,
	}
}

// Returns the error message.
func (e HandlerError) Error() string {
	if e.LogMessage != "" {
		return e.LogMessage
	}
	return e.Err.Error()
}

// Log the HandlerError with according level.
func (e HandlerError) Log() {
	switch e.LogLevel {
	case logrus.TraceLevel:
		logrus.Trace(e.Error())
	case logrus.DebugLevel:
		logrus.Debug(e.Error())
	case logrus.InfoLevel:
		logrus.Info(e.Error())
	case logrus.WarnLevel:
		logrus.Warn(e.Error())
	case logrus.ErrorLevel:
		logrus.Error(e.Error())
	case logrus.FatalLevel:
		logrus.Fatal(e.Error())
	case logrus.PanicLevel:
		logrus.Panic(e.Error())
	}
}

// Set the log level of a HandlerError.
func (e *HandlerError) WithLevel(level logrus.Level) *HandlerError {
	e.LogLevel = level
	return e
}

// Set the log message of a HandlerError.
func (e *HandlerError) WithMessage(message string) *HandlerError {
	e.LogMessage = message
	return e
}

// Handler is an HTTP handler that returns an error.
type Handler func(http.ResponseWriter, *http.Request) error

// Implement the http.Handler interface.
func (fn Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	err := fn(w, r)

	if err != nil {
		if handlerErr, ok := err.(*HandlerError); ok {
			w.WriteHeader(handlerErr.ResponseCode)

			needforheatError := needforheat.Error{Message: handlerErr.ResponseMessage}
			err := json.NewEncoder(w).Encode(&needforheatError)
			if err != nil {
				logrus.Error("failed when returning error to client")
				return
			}

			handlerErr.Log()
			return
		}

		// Return error to client, without giving away too much information.
		http.Error(w, "internal server error", http.StatusInternalServerError)
		logrus.Error(err)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// Helper function to easily create a HandlerError with status 500 (internal server error).
func InternalServerError(err error) *HandlerError {
	return NewHandlerError(err, "internal server error", http.StatusInternalServerError)
}
