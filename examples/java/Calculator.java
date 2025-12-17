package com.example.calculator;

/**
 * A simple calculator class demonstrating basic arithmetic operations.
 * Used as an example for testgen-cli Java support.
 */
public class Calculator {

    /**
     * Adds two integers.
     * @param a first operand
     * @param b second operand
     * @return sum of a and b
     */
    public int add(int a, int b) {
        return a + b;
    }

    /**
     * Subtracts second integer from first.
     * @param a first operand
     * @param b second operand
     * @return difference of a and b
     */
    public int subtract(int a, int b) {
        return a - b;
    }

    /**
     * Multiplies two integers.
     * @param a first operand
     * @param b second operand
     * @return product of a and b
     */
    public int multiply(int a, int b) {
        return a * b;
    }

    /**
     * Divides first integer by second.
     * @param a dividend
     * @param b divisor
     * @return quotient of a divided by b
     * @throws ArithmeticException if b is zero
     */
    public int divide(int a, int b) {
        if (b == 0) {
            throw new ArithmeticException("Division by zero");
        }
        return a / b;
    }
}
