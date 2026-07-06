import { ref, reactive } from 'vue'
import type { MockApiResponse, FlowStep, ApiConfig } from '@/types/flow-preview'

export interface ApiCallResult {
  success: boolean
  data?: Record<string, any>
  error?: string
  statusCode: number
  duration: number
}

function delay(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms))
}

/**
 * Get a nested value from an object using dot notation path
 * e.g., getNestedValue({ a: { b: { c: 1 } } }, 'a.b.c') => 1
 */
function getNestedValue(obj: any, path: string): any {
  if (!path) return obj

  const parts = path.split('.')
  let current = obj

  for (const part of parts) {
    if (current === null || current === undefined) {
      return undefined
    }
    // Handle array access like "items[0]"
    const arrayMatch = part.match(/^(\w+)\[(\d+)\]$/)
    if (arrayMatch) {
      current = current[arrayMatch[1]]
      if (Array.isArray(current)) {
        current = current[parseInt(arrayMatch[2], 10)]
      } else {
        return undefined
      }
    } else {
      current = current[part]
    }
  }

  return current
}

export function useApiMocker() {
  const mocks = reactive<Record<string, MockApiResponse>>({})
  const showMockDialog = ref(false)
  const currentMockStep = ref<string | null>(null)
  const pendingMockResolve = ref<((response: MockApiResponse | null) => void) | null>(null)

  /**
   * Set a mock response for a step
   */
  function setMock(stepName: string, mock: MockApiResponse): void {
    mocks[stepName] = mock
  }

  /**
   * Get existing mock for a step
   */
  function getMock(stepName: string): MockApiResponse | null {
    return mocks[stepName] || null
  }

  /**
   * Remove a mock
   */
  function removeMock(stepName: string): void {
    delete mocks[stepName]
  }

  /**
   * Clear all mocks
   */
  function clearMocks(): void {
    Object.keys(mocks).forEach(key => delete mocks[key])
  }

  /**
   * Execute a mocked API call
   * If no mock exists, prompts user to configure one
   */
  async function executeMockedApiCall(
    step: FlowStep,
    variables: Record<string, any>
  ): Promise<ApiCallResult> {
    const startTime = Date.now()
    const mock = getMock(step.step_name)

    if (mock) {
      // Simulate network delay
      await delay(mock.delay)

      const duration = Date.now() - startTime

      if (mock.statusCode >= 200 && mock.statusCode < 300) {
        return {
          success: true,
          data: mock.responseBody,
          statusCode: mock.statusCode,
          duration
        }
      } else {
        return {
          success: false,
          error: `HTTP ${mock.statusCode}`,
          statusCode: mock.statusCode,
          duration
        }
      }
    }

    // No mock configured - prompt user
    return new Promise((resolve) => {
      currentMockStep.value = step.step_name
      showMockDialog.value = true

      pendingMockResolve.value = (response: MockApiResponse | null) => {
        showMockDialog.value = false
        currentMockStep.value = null
        pendingMockResolve.value = null

        if (response) {
          // Save the mock for future use
          setMock(step.step_name, response)

          // Execute with new mock
          setTimeout(async () => {
            resolve(await executeMockedApiCall(step, variables))
          }, 0)
        } else {
          // User cancelled, return error
          resolve({
            success: false,
            error: 'Mock configuration cancelled',
            statusCode: 0,
            duration: Date.now() - startTime
          })
        }
      }
    })
  }

  /**
   * Handle mock dialog submission
   */
  function submitMockConfig(response: MockApiResponse | null): void {
    if (pendingMockResolve.value) {
      pendingMockResolve.value(response)
    }
  }

  /**
   * Extract variables from API response using response_mapping
   */
  function extractVariablesFromResponse(
    responseData: any,
    responseMapping: Record<string, string>
  ): Record<string, any> {
    const extracted: Record<string, any> = {}

    for (const [varName, path] of Object.entries(responseMapping)) {
      const value = getNestedValue(responseData, path)
      if (value !== undefined) {
        extracted[varName] = value
      }
    }

    return extracted
  }

  /**
   * Generate sample response based on API config
   */
  function generateSampleResponse(apiConfig: ApiConfig): string {
    const sample: Record<string, any> = {}

    // Generate sample data based on response mapping
    for (const [varName, path] of Object.entries(apiConfig.response_mapping || {})) {
      // Create nested structure from path
      const parts = path.split('.')
      let current = sample

      for (let i = 0; i < parts.length - 1; i++) {
        if (!current[parts[i]]) {
          current[parts[i]] = {}
        }
        current = current[parts[i]]
      }

      // Set sample value
      const lastPart = parts[parts.length - 1]
      current[lastPart] = `sample_${varName}`
    }

    return JSON.stringify(sample, null, 2)
  }

  return {
    mocks,
    showMockDialog,
    currentMockStep,
    setMock,
    getMock,
    removeMock,
    clearMocks,
    executeMockedApiCall,
    submitMockConfig,
    extractVariablesFromResponse,
    generateSampleResponse
  }
}
