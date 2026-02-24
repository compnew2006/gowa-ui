import { describe, expect, it } from 'vitest';
import { listContactsArgsSchema, getContactArgsSchema } from '../../src/tools/contacts.js';
import { sendTextMessageArgsSchema } from '../../src/tools/messages.js';
import { createCampaignArgsSchema } from '../../src/tools/campaigns.js';

describe('tool input schemas', () => {
  it('rejects invalid contact ID', () => {
    expect(() => getContactArgsSchema.parse({ contact_id: '' })).toThrow();
  });

  it('rejects oversized message text', () => {
    expect(() => sendTextMessageArgsSchema.parse({
      contact_id: 'abc',
      text: 'x'.repeat(5000)
    })).toThrow();
  });

  it('rejects invalid contact list page', () => {
    expect(() => listContactsArgsSchema.parse({ page: 0, limit: 20 })).toThrow();
  });

  it('requires required campaign fields', () => {
    expect(() => createCampaignArgsSchema.parse({ name: '' })).toThrow();
  });
});
