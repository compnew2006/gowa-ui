export interface InstanceChatCloseRatingSettings {
  enabled: boolean;
  followup_window_minutes: number;
  templates: Record<string, string>;
  use_poll: boolean;
  poll_options: string[];
}

export const DEFAULT_INSTANCE_CHAT_CLOSE_RATING_FOLLOWUP_WINDOW_MINUTES = 15;
export const MIN_INSTANCE_CHAT_CLOSE_RATING_FOLLOWUP_WINDOW_MINUTES = 1;
export const MAX_INSTANCE_CHAT_CLOSE_RATING_FOLLOWUP_WINDOW_MINUTES = 1440;

export const DEFAULT_INSTANCE_CHAT_CLOSE_RATING_TEMPLATES: Record<
  string,
  string
> = {
  en: "Hi {customer_name}, your chat {chat_id} with {agent_name} at {organization_name} is now closed. Please reply with a number from 1 to 10 to rate your experience.",
  ar: "مرحبًا {customer_name}، تم إغلاق المحادثة {chat_id} مع {agent_name} في {organization_name}. الرجاء الرد برقم من 1 إلى 10 لتقييم تجربتك.",
  es: "Hola {customer_name}, tu chat {chat_id} con {agent_name} en {organization_name} se ha cerrado. Responde con un numero del 1 al 10 para calificar tu experiencia.",
};

export function normalizeInstanceChatCloseRatingFollowupWindowMinutes(
  value: unknown,
): number {
  const parsed =
    typeof value === "number"
      ? value
      : typeof value === "string"
        ? Number(value)
        : Number.NaN;

  if (!Number.isFinite(parsed)) {
    return DEFAULT_INSTANCE_CHAT_CLOSE_RATING_FOLLOWUP_WINDOW_MINUTES;
  }

  const rounded = Math.trunc(parsed);
  return Math.min(
    MAX_INSTANCE_CHAT_CLOSE_RATING_FOLLOWUP_WINDOW_MINUTES,
    Math.max(MIN_INSTANCE_CHAT_CLOSE_RATING_FOLLOWUP_WINDOW_MINUTES, rounded),
  );
}

export function normalizeInstanceChatCloseRatingTemplates(
  value: unknown,
): Record<string, string> {
  const templates: Record<string, string> = {
    ...DEFAULT_INSTANCE_CHAT_CLOSE_RATING_TEMPLATES,
  };

  if (!value || typeof value !== "object") {
    return templates;
  }

  for (const [language, rawTemplate] of Object.entries(
    value as Record<string, unknown>,
  )) {
    const key = language.trim().toLowerCase();
    if (!key || typeof rawTemplate !== "string") {
      continue;
    }

    const template = rawTemplate.trim();
    if (!template) {
      continue;
    }

    templates[key] = template;
  }

  return templates;
}

export function normalizeInstanceChatCloseRatingSettings(
  raw: any,
): InstanceChatCloseRatingSettings {
  return {
    enabled:
      typeof raw?.chat_close_rating_enabled === "boolean"
        ? raw.chat_close_rating_enabled
        : true,
    followup_window_minutes:
      normalizeInstanceChatCloseRatingFollowupWindowMinutes(
        raw?.chat_close_rating_followup_window_minutes,
      ),
    templates: normalizeInstanceChatCloseRatingTemplates(
      raw?.chat_close_rating_templates,
    ),
    use_poll:
      typeof raw?.use_poll === "boolean"
        ? raw.use_poll
        : false,
    poll_options:
      Array.isArray(raw?.poll_options)
        ? raw.poll_options.filter((v: any) => typeof v === "string")
        : [],
  };
}

export function cloneInstanceChatCloseRatingSettings(
  current: InstanceChatCloseRatingSettings,
): InstanceChatCloseRatingSettings {
  return {
    enabled: current.enabled,
    followup_window_minutes: current.followup_window_minutes,
    templates: { ...current.templates },
    use_poll: current.use_poll,
    poll_options: Array.isArray(current.poll_options) ? [...current.poll_options] : [],
  };
}

export function sanitizeInstanceChatCloseRatingSettings(
  current: InstanceChatCloseRatingSettings,
): InstanceChatCloseRatingSettings {
  return {
    enabled: current.enabled !== false,
    followup_window_minutes:
      normalizeInstanceChatCloseRatingFollowupWindowMinutes(
        current.followup_window_minutes,
      ),
    templates: normalizeInstanceChatCloseRatingTemplates(current.templates),
    use_poll: current.use_poll === true,
    poll_options: Array.isArray(current.poll_options)
      ? current.poll_options.map((v) => v.trim()).filter((v) => v !== "")
      : [],
  };
}
