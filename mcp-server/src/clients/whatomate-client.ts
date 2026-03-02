import axios, { type AxiosError, type AxiosInstance, type AxiosRequestConfig } from 'axios';
import { URL } from 'node:url';
import type { AppConfig } from '../config.js';
import { AppError } from '../errors.js';

interface Envelope<T> {
  status?: string;
  data: T;
  message?: string;
}

export interface PaginatedResult<T> {
  items: T[];
  total: number;
  page: number;
  limit: number;
  hasMore?: boolean;
}

export interface ContactRecord {
  id: string;
  phone_number?: string;
  name?: string;
  profile_name?: string;
  status?: string;
  [key: string]: unknown;
}

export interface MessageRecord {
  id: string;
  message_type?: string;
  direction?: string;
  status?: string;
  content?: unknown;
  created_at?: string;
  [key: string]: unknown;
}

export interface CampaignRecord {
  id: string;
  name?: string;
  status?: string;
  total_recipients?: number;
  sent_count?: number;
  delivered_count?: number;
  failed_count?: number;
  [key: string]: unknown;
}

export interface DashboardAnalytics {
  [key: string]: unknown;
}

function isRetryableGetError(error: unknown): boolean {
  if (!axios.isAxiosError(error)) {
    return false;
  }

  if (!error.response) {
    return true;
  }

  return error.response.status >= 500;
}

function toNumber(value: unknown, fallback = 0): number {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value;
  }
  return fallback;
}

function sanitizePath(path: string): string {
  if (!path.startsWith('/')) {
    return `/${path}`;
  }
  return path;
}

export class WhatomateClient {
  private readonly http: AxiosInstance;
  private readonly getRetries: number;

  constructor(private readonly config: AppConfig) {
    const host = new URL(config.whatomateApiBaseUrl).hostname;
    if (!config.outboundAllowedHosts.includes(host)) {
      throw new Error(`WHATOMATE_BASE_URL host is not in outbound allowlist: ${host}`);
    }

    this.http = axios.create({
      baseURL: config.whatomateApiBaseUrl,
      timeout: config.requestTimeoutMs,
      headers: {
        'Content-Type': 'application/json',
        'X-API-Key': config.whatomateApiKey,
        ...(config.whatomateOrganizationId ? { 'X-Organization-ID': config.whatomateOrganizationId } : {})
      }
    });

    this.getRetries = config.getRetries;
  }

  async getCurrentOrganization(): Promise<Record<string, unknown>> {
    return this.requestEnvelope<Record<string, unknown>>({ method: 'GET', url: '/organizations/current' }, true);
  }

  async listContacts(params: {
    page?: number;
    limit?: number;
    search?: string;
    account_id?: string;
  }): Promise<PaginatedResult<ContactRecord>> {
    const data = await this.requestEnvelope<Record<string, unknown>>({
      method: 'GET',
      url: '/contacts',
      params
    }, true);

    const items = (data.contacts ?? data.items ?? []) as ContactRecord[];
    return {
      items,
      total: toNumber(data.total, items.length),
      page: toNumber(data.page, params.page ?? 1),
      limit: toNumber(data.limit, params.limit ?? 20)
    };
  }

  async getContact(contactId: string): Promise<ContactRecord> {
    return this.requestEnvelope<ContactRecord>({ method: 'GET', url: `/contacts/${contactId}` }, true);
  }

  async listMessages(contactId: string, params: {
    page?: number;
    limit?: number;
    before_id?: string;
    account?: string;
  }): Promise<PaginatedResult<MessageRecord>> {
    const data = await this.requestEnvelope<Record<string, unknown>>({
      method: 'GET',
      url: `/contacts/${contactId}/messages`,
      params
    }, true);

    const items = (data.messages ?? data.items ?? []) as MessageRecord[];
    return {
      items,
      total: toNumber(data.total, items.length),
      page: toNumber(data.page, params.page ?? 1),
      limit: toNumber(data.limit, params.limit ?? 50),
      hasMore: Boolean(data.has_more)
    };
  }

  async sendTextMessage(contactId: string, text: string, options?: {
    reply_to_message_id?: string;
    instance_id?: string;
    whatsapp_account?: string;
  }): Promise<Record<string, unknown>> {
    return this.requestEnvelope<Record<string, unknown>>({
      method: 'POST',
      url: `/contacts/${contactId}/messages`,
      data: {
        type: 'text',
        content: {
          body: text
        },
        ...(options?.reply_to_message_id ? { reply_to_message_id: options.reply_to_message_id } : {}),
        ...(options?.instance_id ? { instance_id: options.instance_id } : {}),
        ...(options?.whatsapp_account ? { whatsapp_account: options.whatsapp_account } : {})
      }
    });
  }

  async createCampaign(payload: {
    name: string;
    whatsapp_account: string;
    template_id?: string;
    body_content?: string;
    header_media_id?: string;
    min_delay_seconds?: number;
    max_delay_seconds?: number;
    scheduled_at?: string;
  }): Promise<CampaignRecord> {
    return this.requestEnvelope<CampaignRecord>({
      method: 'POST',
      url: '/campaigns',
      data: payload
    });
  }

  async startCampaign(campaignId: string): Promise<Record<string, unknown>> {
    return this.requestEnvelope<Record<string, unknown>>({
      method: 'POST',
      url: `/campaigns/${campaignId}/start`
    });
  }

  async getCampaign(campaignId: string): Promise<CampaignRecord> {
    return this.requestEnvelope<CampaignRecord>({
      method: 'GET',
      url: `/campaigns/${campaignId}`
    }, true);
  }

  async getDashboardAnalytics(params: {
    account_id?: string;
    period?: 'today' | 'week' | 'month' | 'year';
  }): Promise<DashboardAnalytics> {
    return this.requestEnvelope<DashboardAnalytics>({
      method: 'GET',
      url: '/analytics/dashboard',
      params
    }, true);
  }

  private async requestEnvelope<T>(request: AxiosRequestConfig, retryableGet = false, attempt = 0): Promise<T> {
    try {
      const response = await this.http.request<Envelope<T>>({
        ...request,
        url: sanitizePath(request.url ?? '/')
      });

      const body = response.data;
      if (body && typeof body === 'object' && 'status' in body && body.status === 'error') {
        throw new AppError({
          code: 'WHATOMATE_API_ERROR',
          message: typeof body.message === 'string' ? body.message : 'Whatomate API request failed',
          httpStatus: response.status,
          exposeMessage: true,
          details: body
        });
      }

      if (!body || typeof body !== 'object' || !('data' in body)) {
        throw new AppError({
          code: 'WHATOMATE_INVALID_RESPONSE',
          message: 'Whatomate API returned an unexpected response',
          exposeMessage: false,
          details: body
        });
      }

      return body.data;
    } catch (error) {
      if (retryableGet && attempt < this.getRetries && isRetryableGetError(error)) {
        return this.requestEnvelope<T>(request, retryableGet, attempt + 1);
      }

      if (error instanceof AppError) {
        throw error;
      }

      if (axios.isAxiosError(error)) {
        throw this.toAppError(error);
      }

      throw new AppError({
        code: 'WHATOMATE_REQUEST_FAILED',
        message: 'Failed to complete Whatomate API request',
        exposeMessage: false,
        details: error
      });
    }
  }

  private toAppError(error: AxiosError): AppError {
    const status = error.response?.status ?? 500;
    const message = this.extractErrorMessage(error);

    return new AppError({
      code: 'WHATOMATE_REQUEST_FAILED',
      message,
      httpStatus: status,
      exposeMessage: status >= 400 && status < 500,
      details: {
        status,
        path: error.config?.url,
        method: error.config?.method,
        response: error.response?.data
      }
    });
  }

  private extractErrorMessage(error: AxiosError): string {
    const data = error.response?.data;
    if (data && typeof data === 'object' && 'message' in data && typeof data.message === 'string') {
      return data.message;
    }
    if (typeof data === 'string' && data.trim() !== '') {
      return data.slice(0, 200);
    }
    return error.message || 'Whatomate request failed';
  }
}
