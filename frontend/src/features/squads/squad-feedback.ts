import { faultCode, isForbiddenError } from '@/lib/api-client'

/** Keeps inaccessible identity indistinguishable from absence, while separating transient failures. */
export function squadFailureMessage(error: unknown) {
  if (isForbiddenError(error)) return 'squad.forbidden'
  if (faultCode(error) === 'not_found') return 'squad.notFound'
  return 'squad.failed'
}
