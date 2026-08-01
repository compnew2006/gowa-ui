import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
  site: 'https://compnew2006.github.io',
  base: '/Gowa-UI',
  integrations: [
    starlight({
      title: 'Gowa-UI',
      description: 'A modern WhatsApp Business Platform',
      social: [
        { icon: 'github', label: 'GitHub', href: 'https://github.com/compnew2006/gowa-ui' },
      ],
      sidebar: [
        {
          label: 'Getting Started',
          items: [
            { label: 'Introduction', slug: 'getting-started/introduction' },
            { label: 'Quickstart', slug: 'getting-started/quickstart' },
            { label: 'Configuration', slug: 'getting-started/configuration' },
          ],
        },
        {
          label: 'Features',
          items: [
            { label: 'Dashboard', slug: 'features/dashboard' },
            { label: 'WhatsApp Accounts', slug: 'features/accounts' },
            { label: 'Roles & Permissions', slug: 'features/roles-permissions' },
            { label: 'SSO (Single Sign-On)', slug: 'features/sso' },
            { label: 'Audit Logs', slug: 'features/audit-logs' },
            { label: 'Canned Responses', slug: 'features/canned-responses' },
            { label: 'Custom Actions', slug: 'features/custom-actions' },
            { label: 'Templates', slug: 'features/templates' },
            { label: 'Campaigns', slug: 'features/campaigns' },
            { label: 'WhatsApp Status', slug: 'features/status' },
          ],
        },
        {
          label: 'API Reference',
          items: [
            { label: 'Overview', slug: 'api-reference/overview' },
            { label: 'Authentication', slug: 'api-reference/authentication' },
            { label: 'API Keys', slug: 'api-reference/api-keys' },
            { label: 'Users', slug: 'api-reference/users' },
            { label: 'Organizations', slug: 'api-reference/organizations' },
            { label: 'Roles', slug: 'api-reference/roles' },
            { label: 'Accounts', slug: 'api-reference/accounts' },
            { label: 'Contacts', slug: 'api-reference/contacts' },
            { label: 'Messages', slug: 'api-reference/messages' },
            { label: 'Status', slug: 'api-reference/status' },
            { label: 'Templates', slug: 'api-reference/templates' },
            { label: 'Campaigns', slug: 'api-reference/campaigns' },
            { label: 'Canned Responses', slug: 'api-reference/canned-responses' },
            { label: 'Custom Actions', slug: 'api-reference/custom-actions' },
            { label: 'Webhooks', slug: 'api-reference/webhooks' },
            { label: 'Analytics', slug: 'api-reference/analytics' },
          ],
        },
      ],
    }),
  ],
});
