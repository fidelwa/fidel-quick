export interface Customer {
  id: string
  name: string
  slug: string
  phone: string
  address: string
  logo_url: string
  description: string
  welcome_message: string
  active: boolean
  onboarding_completed: boolean
}

export interface Program {
  id: string
  customer_id: string
  type: string
  name: string
  points_ratio: number
  active: boolean
}

export interface Reward {
  id: string
  customer_id: string
  program_id: string
  name: string
  description: string
  points_cost: number
  active: boolean
}

export interface CashbackProgram {
  id: string
  customer_id: string
  type: string
  name: string
  cashback_rate: number
  active: boolean
}

export interface CashbackReward {
  id: string
  customer_id: string
  program_id: string
  name: string
  description: string
  cost: number
  active: boolean
}

export interface Client {
  id: string
  customer_id: string
  name: string
  phone: string
  created_at: string
}

export interface Collaborator {
  id: string
  customer_id: string
  name: string
  phone: string
  hash_id: string
  active: boolean
}

export interface FeedbackEntry {
  id: string
  message: string
  client_name: string
  created_at: string
}

export interface Transaction {
  id: string
  client_id: string
  program_id: string
  collaborator_id: string
  type: "earn" | "burn" | "adjustment"
  amount: number
  balance_after: number
  invoice_url: string
  description: string
  manual_entry: boolean
  correction_reason: string
  correction_evidence_url: string
  correctable_until: string | null
  created_at: string
}

export interface CashbackTransaction {
  id: string
  client_id: string
  program_id: string
  collaborator_id: string
  type: "earn" | "burn" | "adjustment"
  amount: number
  purchase_amount: number
  balance_after: number
  invoice_url: string
  description: string
  manual_entry: boolean
  correction_reason: string
  correction_evidence_url: string
  correctable_until: string | null
  created_at: string
}

export interface Balance {
  client_id: string
  program_id: string
  balance: number
}

export interface OnboardingRegisterRequest {
  name: string
  slug: string
  phone: string
  description?: string
  logo_url?: string
  admin_email: string
  admin_password: string
}

export interface AuthResponse {
  token: string
  admin: {
    id: string
    email: string
    customer_id: string
  }
}
