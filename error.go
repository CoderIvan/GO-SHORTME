package main

// Error *
type Error interface {
	error

	Status() int
}

// StatusError *
type StatusError struct {
	Code int
	Err  error
}

// Error *
func (se StatusError) Error() string {
	return se.Err.Error()
}

// Status *
func (se StatusError) Status() int {
	return se.Code
}
