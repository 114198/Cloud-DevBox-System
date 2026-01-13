//! Shared types and utilities for Cloud DevBox
//!
//! This crate contains common types, error definitions, and utilities
//! shared across all Rust services and applications.

pub mod error;
pub mod types;

pub use error::{Error, Result};
pub use types::*;
