import {
  LayoutDashboard,
  MessageSquare,
  Megaphone,
  Settings,
  Users,
  Contact,
  Key,
  MessageSquareText,
  Webhook,
  BarChart3,
  ShieldCheck,
  Zap,
  Shield,
  Tags,
  ScrollText,
  Server,
  LayoutTemplate
} from 'lucide-vue-next'
import type { Component } from 'vue'

interface NavItem {
  name: string
  path: string
  icon: Component
  permission?: string
  childPermissions?: string[]
  children?: NavItem[]
}

export interface NavSection {
  label: string
  items: NavItem[]
  /** Permissions needed to show section — at least one must pass */
  permissions: string[]
  /** Pin to bottom of sidebar */
  pinBottom?: boolean
}

export const navigationSections: NavSection[] = [
  {
    label: 'nav.sectionMain',
    permissions: ['analytics', 'chat'],
    items: [
      {
        name: 'nav.dashboard',
        path: '/',
        icon: LayoutDashboard,
        permission: 'analytics'
      },
      {
        name: 'nav.chat',
        path: '/chat',
        icon: MessageSquare,
        permission: 'chat'
      },
    ]
  },
  {
    label: 'nav.sectionMessaging',
    permissions: ['campaigns'],
    items: [
      {
        name: 'nav.campaigns',
        path: '/campaigns',
        icon: Megaphone,
        permission: 'campaigns'
      },
    ]
  },
  {
    label: 'nav.sectionAnalytics',
    permissions: ['analytics.agents', 'analytics'],
    items: [
      {
        name: 'nav.agentAnalytics',
        path: '/analytics/agents',
        icon: BarChart3,
        permission: 'analytics.agents'
      },
    ]
  },
  {
    label: '',
    permissions: ['settings.general', 'accounts', 'contacts.manage', 'canned_responses', 'tags', 'templates', 'teams', 'users', 'roles', 'api_keys', 'webhooks', 'custom_actions', 'settings.sso', 'audit_logs', 'gowa_instances'],
    pinBottom: true,
    items: [
      {
        name: 'nav.settings',
        path: '/settings',
        icon: Settings,
        permission: 'settings.general',
        childPermissions: ['settings.general', 'accounts', 'contacts.manage', 'canned_responses', 'tags', 'templates', 'teams', 'users', 'roles', 'api_keys', 'webhooks', 'custom_actions', 'settings.sso', 'audit_logs', 'gowa_instances'],
        children: [
          { name: 'nav.general', path: '/settings', icon: Settings, permission: 'settings.general' },
          { name: 'nav.accounts', path: '/settings/accounts', icon: Users, permission: 'accounts' },
          { name: 'nav.contacts', path: '/settings/contacts', icon: Contact, permission: 'contacts.manage' },
          { name: 'nav.cannedResponses', path: '/settings/canned-responses', icon: MessageSquareText, permission: 'canned_responses' },
          { name: 'nav.tags', path: '/settings/tags', icon: Tags, permission: 'tags' },
          { name: 'nav.templates', path: '/settings/templates', icon: LayoutTemplate, permission: 'templates' },
          { name: 'nav.teams', path: '/settings/teams', icon: Users, permission: 'teams' },
          { name: 'nav.users', path: '/settings/users', icon: Users, permission: 'users' },
          { name: 'nav.roles', path: '/settings/roles', icon: Shield, permission: 'roles' },
          { name: 'nav.apiKeys', path: '/settings/api-keys', icon: Key, permission: 'api_keys' },
          { name: 'nav.webhooks', path: '/settings/webhooks', icon: Webhook, permission: 'webhooks' },
          { name: 'nav.customActions', path: '/settings/custom-actions', icon: Zap, permission: 'custom_actions' },
          { name: 'nav.sso', path: '/settings/sso', icon: ShieldCheck, permission: 'settings.sso' },
          { name: 'nav.auditLogs', path: '/settings/audit-logs', icon: ScrollText, permission: 'audit_logs' },
          { name: 'nav.gowaServers', path: '/settings/gowa-servers', icon: Server, permission: 'gowa_instances' }
        ]
      }
    ]
  }
]
