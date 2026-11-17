/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function sleep(ms: number = 1000) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

/**
 * Sanitize a CSS variable name by replacing special characters
 * Converts a model name (e.g. gpt-3.5-turbo) into a valid CSS variable name (gpt-3-5-turbo)
 * @param name - Original name
 * @returns The sanitized CSS variable name
 */
export function sanitizeCssVariableName(name: string): string {
  // Replace dots, spaces, and slashes with hyphens
  // Remove other characters that are not allowed in CSS variable names
  return name.replace(/[.\s/]/g, '-').replace(/[^\w-]/g, '')
}

/**
 * Generates page numbers for pagination with ellipsis
 * @param currentPage - Current page number (1-based)
 * @param totalPages - Total number of pages
 * @returns Array of page numbers and ellipsis strings
 *
 * Examples:
 * - Small dataset (≤4 pages): [1, 2, 3, 4]
 * - Near beginning: [1, 2, '...', 10]
 * - In middle: [1, '...', 5, '...', 10]
 * - Near end: [1, '...', 9, 10]
 */
export function getPageNumbers(currentPage: number, totalPages: number) {
  const maxVisiblePages = 4
  const rangeWithDots = []

  if (totalPages <= maxVisiblePages) {
    for (let i = 1; i <= totalPages; i++) {
      rangeWithDots.push(i)
    }
  } else {
    rangeWithDots.push(1)

    if (currentPage <= 2) {
      rangeWithDots.push(2)
      rangeWithDots.push('...', totalPages)
    } else if (currentPage >= totalPages - 1) {
      rangeWithDots.push('...')
      rangeWithDots.push(totalPages - 1, totalPages)
    } else {
      rangeWithDots.push('...')
      rangeWithDots.push(currentPage)
      rangeWithDots.push('...', totalPages)
    }
  }

  return rangeWithDots
}

/**
 * Truncate text to a maximum length with ellipsis
 */
export function truncateText(text: string, maxLength: number): string {
  if (!text || text.length <= maxLength) return text
  return text.slice(0, maxLength) + '...'
}

/**
 * Try to parse and pretty-print JSON, fallback to original text if invalid
 * @param text - Text that might be JSON
 * @returns Pretty-printed JSON or original text
 */
export function tryPrettyJson(text: string): string {
  const raw = (text ?? '').toString().trim()
  if (!raw) return ''
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}
