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
}

export interface OnboardingStatus {
  id?: string
  customer_id?: string
  current_step: number
  completed: boolean
  completed_at?: string
}

export interface Sisfi {
  id: string
  name: string
  description: string
  active: boolean
}

export interface CustomerSisfi {
  id: string
  customer_id: string
  sisfi_id: string
  name: string
  active: boolean
}

export interface Program {
  id: string
  customer_id: string
  name: string
  points_ratio: number
  active: boolean
}

export interface Reward {
  id: string
  customer_id: string
  customer_sisfi_id: string
  name: string
  description: string
  points_cost: number
  active: boolean
}

export interface CashbackProgram {
  id: string
  customer_id: string
  name: string
  cashback_rate: number
  active: boolean
}

export interface CashbackReward {
  id: string
  customer_id: string
  customer_sisfi_id: string
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
  customer_sisfi_id: string
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
  customer_sisfi_id: string
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

export interface PushcardConfig {
  customer_sisfi_id: string
  customer_id: string
  name: string
  card_slots: number
  reward_on_complete: string
  active: boolean
}

export interface PushcardCard {
  id: string
  customer_sisfi_id: string
  client_id: string
  status: "open" | "completed" | "redeemed" | "cancelled"
  stamps_count: number
  completed_at: string | null
  created_at: string
  updated_at: string
}

export interface Balance {
  client_id: string
  customer_sisfi_id: string
  balance: number
}

export interface OnboardingRegisterRequest {
  name: string
  phone: string
  country_code?: string
  description?: string
  logo_url?: string
  admin_email: string
  admin_password: string
}

export interface GoogleOnboardingRequest {
  google_token: string
  name: string
  phone: string
  description?: string
}

export interface AdminSummary {
  id: string
  email: string
  customer_id: string
  google_email?: string | null
}

export interface AuthResponse {
  token: string
  admin: AdminSummary
}
