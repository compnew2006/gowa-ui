import { contactsService } from "@/services/api";
import type { Contact } from "@/stores/contacts";

const MENTION_PHONE_REGEX = /@(\+?\d{6,20})/g;

interface ContactCandidate {
  phone_number?: unknown;
  profile_name?: unknown;
  name?: unknown;
}

function normalizeText(value: unknown): string {
  if (typeof value !== "string") return "";
  return value.replace(/\s+/g, " ").trim();
}

function normalizePhoneForLookup(value: unknown): string {
  if (typeof value !== "string") return "";
  return value.replace(/[^\d]/g, "");
}

function getContactDisplayName(candidate: ContactCandidate): string {
  return (
    normalizeText(candidate.profile_name) ||
    normalizeText(candidate.name) ||
    normalizeText(candidate.phone_number)
  );
}

function listEnvelopeToContacts(payload: unknown): ContactCandidate[] {
  if (!payload || typeof payload !== "object") {
    return [];
  }

  const root = payload as Record<string, unknown>;
  const envelope =
    root.data && typeof root.data === "object"
      ? (root.data as Record<string, unknown>)
      : root;
  const contacts = envelope.contacts;
  if (!Array.isArray(contacts)) {
    return [];
  }

  return contacts as ContactCandidate[];
}

export function extractMentionPhones(text: string): string[] {
  if (!text || !text.includes("@")) {
    return [];
  }

  const phones = new Set<string>();
  for (const match of text.matchAll(MENTION_PHONE_REGEX)) {
    const normalized = normalizePhoneForLookup(match[1]);
    if (normalized) {
      phones.add(normalized);
    }
  }
  return Array.from(phones);
}

export class MentionContactResolver {
  private readonly nameByPhone = new Map<string, string>();
  private readonly attemptedLookups = new Set<string>();
  private readonly pendingLookups = new Map<string, Promise<boolean>>();

  preloadContacts(contacts: Contact[]): boolean {
    let changed = false;

    for (const contact of contacts) {
      const normalizedPhone = normalizePhoneForLookup(contact.phone_number);
      if (!normalizedPhone) continue;

      const displayName = getContactDisplayName(contact);
      if (!displayName) continue;

      const currentName = this.nameByPhone.get(normalizedPhone);
      if (currentName === displayName) continue;

      this.nameByPhone.set(normalizedPhone, displayName);
      changed = true;
    }

    return changed;
  }

  replaceMentions(text: string): string {
    if (!text || !text.includes("@")) {
      return text;
    }

    return text.replace(MENTION_PHONE_REGEX, (fullMatch, rawPhone: string) => {
      const normalizedPhone = normalizePhoneForLookup(rawPhone);
      if (!normalizedPhone) {
        return fullMatch;
      }

      const displayName = this.nameByPhone.get(normalizedPhone);
      if (!displayName) {
        return fullMatch;
      }

      return `@${displayName}`;
    });
  }

  async resolveMentionsInTexts(texts: string[]): Promise<boolean> {
    const phones = new Set<string>();

    for (const text of texts) {
      for (const phone of extractMentionPhones(text)) {
        phones.add(phone);
      }
    }

    if (phones.size === 0) {
      return false;
    }

    const results = await Promise.all(
      Array.from(phones, (phone) => this.resolvePhone(phone)),
    );
    return results.some(Boolean);
  }

  private async resolvePhone(normalizedPhone: string): Promise<boolean> {
    if (!normalizedPhone) {
      return false;
    }

    if (this.nameByPhone.has(normalizedPhone)) {
      return false;
    }

    const existingLookup = this.pendingLookups.get(normalizedPhone);
    if (existingLookup) {
      return existingLookup;
    }

    if (this.attemptedLookups.has(normalizedPhone)) {
      return false;
    }

    const lookup = this.resolvePhoneFromApi(normalizedPhone)
      .catch(() => false)
      .finally(() => {
        this.pendingLookups.delete(normalizedPhone);
      });

    this.pendingLookups.set(normalizedPhone, lookup);
    return lookup;
  }

  private async resolvePhoneFromApi(normalizedPhone: string): Promise<boolean> {
    const response = await contactsService.list({
      search: normalizedPhone,
      page: 1,
      limit: 50,
    });

    const contacts = listEnvelopeToContacts(response.data);
    if (contacts.length === 0) {
      this.attemptedLookups.add(normalizedPhone);
      return false;
    }

    let bestMatch: ContactCandidate | null = null;
    for (const contact of contacts) {
      const candidatePhone = normalizePhoneForLookup(contact.phone_number);
      if (!candidatePhone) continue;
      if (candidatePhone === normalizedPhone) {
        bestMatch = contact;
        break;
      }
      if (
        !bestMatch &&
        (candidatePhone.endsWith(normalizedPhone) ||
          normalizedPhone.endsWith(candidatePhone))
      ) {
        bestMatch = contact;
      }
    }

    if (!bestMatch) {
      this.attemptedLookups.add(normalizedPhone);
      return false;
    }

    const displayName = getContactDisplayName(bestMatch);
    if (!displayName) {
      this.attemptedLookups.add(normalizedPhone);
      return false;
    }

    this.nameByPhone.set(normalizedPhone, displayName);
    this.attemptedLookups.add(normalizedPhone);
    return true;
  }
}
