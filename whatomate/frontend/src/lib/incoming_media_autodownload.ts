import { useConfigStore } from '@/stores/config'
import { useInstancesStore } from '@/stores/instances'
import { isMediaMessageType, prefetchMediaBlob } from '@/lib/media_prefetch_cache'

type IncomingMessagePayload = {
  id?: unknown
  direction?: unknown
  message_type?: unknown
  instance_id?: unknown
}

function normalizeString(value: unknown): string {
  return typeof value === 'string' ? value.trim() : ''
}

export interface IncomingMediaAutoDownloadDecisionInput {
  provider: unknown
  instanceAutoDownloadEnabled: boolean
  payload: IncomingMessagePayload
}

export function shouldAutoDownloadIncomingMedia(input: IncomingMediaAutoDownloadDecisionInput): boolean {
  const provider = normalizeString(input.provider).toLowerCase()
  if (provider !== 'whatsmeow') return false
  if (!input.instanceAutoDownloadEnabled) return false

  const messageID = normalizeString(input.payload.id)
  if (!messageID) return false

  const direction = normalizeString(input.payload.direction).toLowerCase()
  if (direction !== 'incoming') return false

  return isMediaMessageType(input.payload.message_type)
}

function isInstanceAutoDownloadEnabled(instanceID: string): boolean {
  if (!instanceID) return false
  const instancesStore = useInstancesStore()
  const instance = instancesStore.instances.find(item => item.id === instanceID)
  return instance?.settings?.auto_download_incoming_media === true
}

export function maybeAutoDownloadIncomingMedia(payload: unknown): void {
  const parsedPayload = (payload && typeof payload === 'object')
    ? (payload as IncomingMessagePayload)
    : {}

  const instanceID = normalizeString(parsedPayload.instance_id)
  const configStore = useConfigStore()

  const shouldDownload = shouldAutoDownloadIncomingMedia({
    provider: configStore.provider,
    instanceAutoDownloadEnabled: isInstanceAutoDownloadEnabled(instanceID),
    payload: parsedPayload,
  })
  if (!shouldDownload) return

  const messageID = normalizeString(parsedPayload.id)
  void prefetchMediaBlob(messageID).catch(() => {
    // Intentionally ignore prefetch failures to keep realtime flow non-blocking.
  })
}
