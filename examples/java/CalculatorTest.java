package com.example.calculator;

import org.junit.jupiter.api.*;
import static org.junit.jupiter.api.Assertions.*;

public class CalculatorTest {

    private Calculator calculator;

    @BeforeEach
    void setup() {
        calculator = new Calculator();
    }

    @Test
    @DisplayName("testAdd_twoPositiveNumbers_returnsSum")
    void testAdd_twoPositiveNumbers_returnsSum() {
        // Test adding two positive numbers
        int result = calculator.add(5, 7);
        assertEquals(12, result);
    }

    @Test
    @DisplayName("testAdd_twoNegativeNumbers_returnsSum")
    void testAdd_twoNegativeNumbers_returnsSum() {
        // Test adding two negative numbers
        int result = calculator.add(-5, -7);
        assertEquals(-12, result);
    }

    @Test
    @DisplayName("testAdd_positiveAndNegativeNumbers_returnsSum")
    void testAdd_positiveAndNegativeNumbers_returnsSum() {
        // Test adding a positive and a negative number
        int result = calculator.add(5, -7);
        assertEquals(-2, result);
    }

    @Test
    @DisplayName("testAdd_zeroAndPositiveNumber_returnsSum")
    void testAdd_zeroAndPositiveNumber_returnsSum() {
        // Test adding zero and a positive number
        int result = calculator.add(0, 7);
        assertEquals(7, result);
    }

    @Test
    @DisplayName("testAdd_zeroAndNegativeNumber_returnsSum")
    void testAdd_zeroAndNegativeNumber_returnsSum() {
        // Test adding zero and a negative number
        int result = calculator.add(0, -7);
        assertEquals(-7, result);
    }

    @Test
    @DisplayName("testAdd_twoZeros_returnsZero")
    void testAdd_twoZeros_returnsZero() {
        // Test adding two zeros
        int result = calculator.add(0, 0);
        assertEquals(0, result);
    }
}

package com.example.calculator;

import org.junit.jupiter.api.*;
import static org.junit.jupiter.api.Assertions.*;

public class CalculatorTest {

    private Calculator calculator;

    @BeforeEach
    void setup() {
        calculator = new Calculator();
    }

    @Test
    @DisplayName("testSubtract_TwoPositiveNumbers_ReturnsDifference")
    void testSubtract_twoPositiveNumbers_returnsDifference() {
        // Test purpose: Verify subtract method returns correct difference for two positive numbers
        int a = 10;
        int b = 5;
        int expectedResult = 5;
        int result = calculator.subtract(a, b);
        assertEquals(expectedResult, result);
    }

    @Test
    @DisplayName("testSubtract_TwoNegativeNumbers_ReturnsDifference")
    void testSubtract_twoNegativeNumbers_returnsDifference() {
        // Test purpose: Verify subtract method returns correct difference for two negative numbers
        int a = -10;
        int b = -5;
        int expectedResult = -5;
        int result = calculator.subtract(a, b);
        assertEquals(expectedResult, result);
    }

    @Test
    @DisplayName("testSubtract_PositiveAndNegativeNumbers_ReturnsDifference")
    void testSubtract_positiveAndNegativeNumbers_returnsDifference() {
        // Test purpose: Verify subtract method returns correct difference for positive and negative numbers
        int a = 10;
        int b = -5;
        int expectedResult = 15;
        int result = calculator.subtract(a, b);
        assertEquals(expectedResult, result);
    }

    @Test
    @DisplayName("testSubtract_EqualNumbers_ReturnsZero")
    void testSubtract_equalNumbers_returnsZero() {
        // Test purpose: Verify subtract method returns zero for equal numbers
        int a = 10;
        int b = 10;
        int expectedResult = 0;
        int result = calculator.subtract(a, b);
        assertEquals(expectedResult, result);
    }

    @Test
    @DisplayName("testSubtract_InvalidInput_ThrowsException")
    void testSubtract_invalidInput_throwsException() {
        // Test purpose: Verify subtract method throws exception for invalid input
        assertThrows(NullPointerException.class, () -> calculator.subtract(null, 5));
    }
}

package com.example.calculator;

import org.junit.jupiter.api.*;
import static org.junit.jupiter.api.Assertions.*;

public class CalculatorTest {

    private Calculator calculator;

    @BeforeEach
    void setup() {
        calculator = new Calculator();
    }

    @Test
    @DisplayName("testMultiply_twoPositiveNumbers_product")
    void testMultiply_twoPositiveNumbers_product() {
        // Test purpose: Verify multiplication of two positive numbers
        int a = 5;
        int b = 3;
        int expectedResult = 15;
        int result = calculator.multiply(a, b);
        assertEquals(expectedResult, result);
    }

    @Test
    @DisplayName("testMultiply_twoNegativeNumbers_product")
    void testMultiply_twoNegativeNumbers_product() {
        // Test purpose: Verify multiplication of two negative numbers
        int a = -5;
        int b = -3;
        int expectedResult = 15;
        int result = calculator.multiply(a, b);
        assertEquals(expectedResult, result);
    }

    @Test
    @DisplayName("testMultiply_onePositiveAndOneNegativeNumber_product")
    void testMultiply_onePositiveAndOneNegativeNumber_product() {
        // Test purpose: Verify multiplication of one positive and one negative number
        int a = 5;
        int b = -3;
        int expectedResult = -15;
        int result = calculator.multiply(a, b);
        assertEquals(expectedResult, result);
    }

    @Test
    @DisplayName("testMultiply_zeroAndPositiveNumber_product")
    void testMultiply_zeroAndPositiveNumber_product() {
        // Test purpose: Verify multiplication of zero and a positive number
        int a = 0;
        int b = 5;
        int expectedResult = 0;
        int result = calculator.multiply(a, b);
        assertEquals(expectedResult, result);
    }

    @Test
    @DisplayName("testMultiply_zeroAndNegativeNumber_product")
    void testMultiply_zeroAndNegativeNumber_product() {
        // Test purpose: Verify multiplication of zero and a negative number
        int a = 0;
        int b = -5;
        int expectedResult = 0;
        int result = calculator.multiply(a, b);
        assertEquals(expectedResult, result);
    }

    @Test
    @DisplayName("testMultiply_twoZeros_product")
    void testMultiply_twoZeros_product() {
        // Test purpose: Verify multiplication of two zeros
        int a = 0;
        int b = 0;
        int expectedResult = 0;
        int result = calculator.multiply(a, b);
        assertEquals(expectedResult, result);
    }
}

package com.example.calculator;

import org.junit.jupiter.api.*;
import static org.junit.jupiter.api.Assertions.*;

public class CalculatorTest {

    private Calculator calculator;

    @BeforeEach
    void setup() {
        calculator = new Calculator();
    }

    @Test
    @DisplayName("testDivide_twoPositiveNumbers_resultIsCorrect")
    void testDivide_twoPositiveNumbers_resultIsCorrect() {
        // Test purpose: Verify division of two positive numbers returns the correct result
        int a = 10;
        int b = 2;
        int expectedResult = 5;
        int actualResult = calculator.divide(a, b);
        assertEquals(expectedResult, actualResult);
    }

    @Test
    @DisplayName("testDivide_twoNegativeNumbers_resultIsCorrect")
    void testDivide_twoNegativeNumbers_resultIsCorrect() {
        // Test purpose: Verify division of two negative numbers returns the correct result
        int a = -10;
        int b = -2;
        int expectedResult = 5;
        int actualResult = calculator.divide(a, b);
        assertEquals(expectedResult, actualResult);
    }

    @Test
    @DisplayName("testDivide_positiveAndNegativeNumbers_resultIsCorrect")
    void testDivide_positiveAndNegativeNumbers_resultIsCorrect() {
        // Test purpose: Verify division of a positive and a negative number returns the correct result
        int a = 10;
        int b = -2;
        int expectedResult = -5;
        int actualResult = calculator.divide(a, b);
        assertEquals(expectedResult, actualResult);
    }

    @Test
    @DisplayName("testDivide_divisionByZero_throwsArithmeticException")
    void testDivide_divisionByZero_throwsArithmeticException() {
        // Test purpose: Verify division by zero throws an ArithmeticException
        int a = 10;
        int b = 0;
        assertThrows(ArithmeticException.class, () -> calculator.divide(a, b));
    }
}


