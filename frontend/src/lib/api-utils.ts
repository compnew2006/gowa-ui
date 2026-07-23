/**
 * API utility functions for handling errors consistently
 */

import type { AxiosError } from 'axios'

/**
 * Extracts error message from various error formats.
 * Handles Axios errors, standard errors, and string errors.
 */
export function getErrorMessage(error: unknown, defaultMessage = 'An error occurred'): string {
  if (!error) {
    return defaultMessage
  }

  // Handle Axios error with response
  if (isAxiosError(error)) {
    const responseMessage = error.response?.data?.message
    if (responseMessage && typeof responseMessage === 'string') {
      return responseMessage
    }

    // Handle error array in response
    const errors = error.response?.data?.errors
    if (Array.isArray(errors) && errors.length > 0) {
      return errors[0].message || errors[0] || defaultMessage
    }
  }

  // Handle standard Error object
  if (error instanceof Error) {
    return error.message || defaultMessage
  }

  // Handle string error
  if (typeof error === 'string') {
    return error
  }

  return defaultMessage
}

/**
 * Type guard for Axios errors
 */
function isAxiosError(error: unknown): error is AxiosError<{ message?: string; errors?: any[] }> {
  return (
    typeof error === 'object' &&
    error !== null &&
    'isAxiosError' in error &&
    (error as any).isAxiosError === true
  )
}
