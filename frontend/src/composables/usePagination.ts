/**
 * Generate an array of page numbers for pagination UI.
 * Shows first, last, current, and surrounding pages with ellipses.
 *
 * @param currentPage - Current page number
 * @param totalPages - Total number of pages
 * @param siblingCount - Number of pages to show on each side of current (default: 1)
 *
 * @example
 * // For page 5 of 10: [1, '...', 4, 5, 6, '...', 10]
 * const pages = getPageNumbers(5, 10)
 */
export function getPageNumbers(
  currentPage: number,
  totalPages: number,
  siblingCount = 1
): (number | '...')[] {
  const pages: (number | '...')[] = []

  if (totalPages <= 5 + siblingCount * 2) {
    // Show all pages if total is small
    for (let i = 1; i <= totalPages; i++) {
      pages.push(i)
    }
    return pages
  }

  // Always show first page
  pages.push(1)

  // Calculate range around current page
  const leftSibling = Math.max(2, currentPage - siblingCount)
  const rightSibling = Math.min(totalPages - 1, currentPage + siblingCount)

  // Add ellipsis after first page if needed
  if (leftSibling > 2) {
    pages.push('...')
  }

  // Add pages around current
  for (let i = leftSibling; i <= rightSibling; i++) {
    if (i !== 1 && i !== totalPages) {
      pages.push(i)
    }
  }

  // Add ellipsis before last page if needed
  if (rightSibling < totalPages - 1) {
    pages.push('...')
  }

  // Always show last page
  if (totalPages > 1) {
    pages.push(totalPages)
  }

  return pages
}
