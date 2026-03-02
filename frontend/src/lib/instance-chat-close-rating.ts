export interface InstanceChatCloseRatingSettings {
  enabled: boolean;
  override_org_settings: boolean; // Virtual flag for frontend
  followup_window_minutes: number;
  templates: Record<string, string>;
}

export function normalizeInstanceChatCloseRatingSettings(
  raw: any
): InstanceChatCloseRatingSettings {
  // We use the presence of `chat_close_rating_templates` or similar in instance settings
  // as an indicator that the org settings are overridden.
  const hasOverride = raw && typeof raw === "object" && 'chat_close_rating_templates' in raw;

  return {
    enabled: typeof raw?.chat_close_rating_enabled === 'boolean' ? raw.chat_close_rating_enabled : true,
    override_org_settings: hasOverride,
    followup_window_minutes: typeof raw?.chat_close_rating_followup_window_minutes === 'number' ? raw.chat_close_rating_followup_window_minutes : 15,
    templates: {
      en: 'Please rate your experience with us out of 5:\n1: 😡 Very Poor\n2: 🙁 Poor\n3: 😐 Average\n4: 🙂 Good\n5: 😍 Excellent',
      ar: 'يرجى تقييم تجربتك معنا من 5:\n1: 😡 سيء جداً\n2: 🙁 سيء\n3: 😐 متوسط\n4: 🙂 جيد\n5: 😍 ممتاز',
      es: 'Por favor, califique su experiencia con nosotros de 5:\n1: 😡 Muy mala\n2: 🙁 Mala\n3: 😐 Regular\n4: 🙂 Buena\n5: 😍 Excelente',
      ...(raw?.chat_close_rating_templates || {})
    }
  };
}

export function cloneInstanceChatCloseRatingSettings(
  current: InstanceChatCloseRatingSettings
): InstanceChatCloseRatingSettings {
  return {
    enabled: current.enabled,
    override_org_settings: current.override_org_settings,
    followup_window_minutes: current.followup_window_minutes,
    templates: { ...current.templates },
  };
}
