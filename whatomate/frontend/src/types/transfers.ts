export interface AgentTransfer {
  id: string
  contact_id: string
  contact_name: string
  phone_number: string
  whatsapp_account: string
  instance_id?: string
  status: 'active' | 'resumed' | 'expired'
  source: 'manual' | 'flow' | 'keyword'
  agent_id?: string
  agent_name?: string
  team_id?: string
  team_name?: string
  transferred_by?: string
  transferred_by_name?: string
  notes?: string
  transferred_at: string
  resumed_at?: string
  resumed_by?: string
  resumed_by_name?: string
  // SLA fields
  sla_response_deadline?: string
  sla_resolution_deadline?: string
  sla_breached: boolean
  sla_breached_at?: string
  escalation_level: number
  escalated_at?: string
  picked_up_at?: string
  expires_at?: string
}

export type SLAStatus = 'ok' | 'warning' | 'breached' | 'expired'
