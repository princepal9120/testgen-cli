//! Calculator module providing basic arithmetic operations.

/// Returns the sum of two integers.
pub fn add(a: i32, b: i32) -> i32 {
    a + b
}

/// Returns the difference of two integers.
pub fn subtract(a: i32, b: i32) -> i32 {
    a - b
}

/// Returns the product of two integers.
pub fn multiply(a: i32, b: i32) -> i32 {
    a * b
}

/// Returns the quotient of two integers.
/// 
/// # Arguments
/// * `a` - The dividend
/// * `b` - The divisor
/// 
/// # Returns
/// * `Ok(i32)` - The quotient
/// * `Err(String)` - Error if b is zero
pub fn divide(a: i32, b: i32) -> Result<i32, String> {
    if b == 0 {
        return Err("Cannot divide by zero".to_string());
    }
    Ok(a / b)
}
