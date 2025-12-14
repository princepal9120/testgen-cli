/**
 * Calculator module providing basic arithmetic operations.
 */

/**
 * Returns the sum of two numbers.
 * @param {number} a - First number
 * @param {number} b - Second number
 * @returns {number} The sum
 */
export function add(a, b) {
    return a + b;
}

/**
 * Returns the difference of two numbers.
 * @param {number} a - First number
 * @param {number} b - Second number
 * @returns {number} The difference
 */
export function subtract(a, b) {
    return a - b;
}

/**
 * Returns the product of two numbers.
 * @param {number} a - First number
 * @param {number} b - Second number
 * @returns {number} The product
 */
export function multiply(a, b) {
    return a * b;
}

/**
 * Returns the quotient of two numbers.
 * @param {number} a - The dividend
 * @param {number} b - The divisor
 * @returns {number} The quotient
 * @throws {Error} If b is zero
 */
export function divide(a, b) {
    if (b === 0) {
        throw new Error("Cannot divide by zero");
    }
    return a / b;
}
