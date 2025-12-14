// Package calculator provides basic arithmetic operations
package calculator

// Add returns the sum of two integers
func Add(a, b int) int {
	return a + b
}

// Subtract returns the difference of two integers
func Subtract(a, b int) int {
	return a - b
}

// Multiply returns the product of two integers
func Multiply(a, b int) int {
	return a * b
}

// Divide returns the quotient of two integers
// Returns an error if b is zero
func Divide(a, b int) (int, error) {
	if b == 0 {
		return 0, ErrDivisionByZero
	}
	return a / b, nil
}

// ErrDivisionByZero is returned when division by zero is attempted
var ErrDivisionByZero = &DivisionError{message: "division by zero"}

// DivisionError represents a division error
type DivisionError struct {
	message string
}

func (e *DivisionError) Error() string {
	return e.message
}
