package validator

import "fmt"

// ResultContext is a builder that sets Category and a default File once,
// so individual results inherit location context automatically.
type ResultContext struct {
	Category string
	File     string // default file; methods like ErrorFile override it
}

func (c ResultContext) result(level Level, file string, line int, msg string) Result {
	if file == "" {
		file = c.File
	}
	return Result{
		Level:    level,
		Category: c.Category,
		Message:  msg,
		File:     file,
		Line:     line,
	}
}

// Pass creates a pass result using the default file.
func (c ResultContext) Pass(msg string) Result { return c.result(Pass, "", 0, msg) }

// Passf creates a formatted pass result using the default file.
func (c ResultContext) Passf(format string, args ...any) Result {
	return c.result(Pass, "", 0, fmt.Sprintf(format, args...))
}

// Info creates an info result using the default file.
func (c ResultContext) Info(msg string) Result { return c.result(Info, "", 0, msg) }

// Infof creates a formatted info result using the default file.
func (c ResultContext) Infof(format string, args ...any) Result {
	return c.result(Info, "", 0, fmt.Sprintf(format, args...))
}

// Warn creates a warning result using the default file.
func (c ResultContext) Warn(msg string) Result { return c.result(Warning, "", 0, msg) }

// Warnf creates a formatted warning result using the default file.
func (c ResultContext) Warnf(format string, args ...any) Result {
	return c.result(Warning, "", 0, fmt.Sprintf(format, args...))
}

// Error creates an error result using the default file.
func (c ResultContext) Error(msg string) Result { return c.result(Error, "", 0, msg) }

// Errorf creates a formatted error result using the default file.
func (c ResultContext) Errorf(format string, args ...any) Result {
	return c.result(Error, "", 0, fmt.Sprintf(format, args...))
}

// PassFile creates a pass result with an explicit file.
func (c ResultContext) PassFile(file, msg string) Result { return c.result(Pass, file, 0, msg) }

// WarnFile creates a warning result with an explicit file.
func (c ResultContext) WarnFile(file, msg string) Result { return c.result(Warning, file, 0, msg) }

// WarnFilef creates a formatted warning result with an explicit file.
func (c ResultContext) WarnFilef(file, format string, args ...any) Result {
	return c.result(Warning, file, 0, fmt.Sprintf(format, args...))
}

// ErrorFile creates an error result with an explicit file.
func (c ResultContext) ErrorFile(file, msg string) Result { return c.result(Error, file, 0, msg) }

// ErrorFilef creates a formatted error result with an explicit file.
func (c ResultContext) ErrorFilef(file, format string, args ...any) Result {
	return c.result(Error, file, 0, fmt.Sprintf(format, args...))
}

// ErrorAtLine creates an error result with an explicit file and line number.
func (c ResultContext) ErrorAtLine(file string, line int, msg string) Result {
	return c.result(Error, file, line, msg)
}

// ErrorAtLinef creates a formatted error result with an explicit file and line number.
func (c ResultContext) ErrorAtLinef(file string, line int, format string, args ...any) Result {
	return c.result(Error, file, line, fmt.Sprintf(format, args...))
}
