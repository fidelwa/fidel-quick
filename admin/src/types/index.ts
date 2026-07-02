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
  // FID-34: días para que venzan los puntos. null = sin vencimiento.
  expiry_days: number | null
  // FID-36: monto mínimo de compra para acreditar. null = sin mínimo.
  min_ticket_amount: number | null
}

export interface Reward {
  id: string
  customer_id: string
  customer_sisfi_id: string
  name: string
  description: string
  points_cost: number
  active: boolean
  // FID-38: stock/disponibilidad limitada. null = ilimitado.
  stock_total: number | null
  redeemed_count: number
  limit_per_client: number | null
}

export interface CashbackProgram {
  id: string
  customer_id: string
  name: string
  cashback_rate: number
  active: boolean
  // FID-34: días para que venza el saldo. null = sin vencimiento.
  expiry_days: number | null
  // FID-36: monto mínimo de compra para acreditar. null = sin mínimo.
  min_ticket_amount: number | null
  // FID-37: techo de cashback por transacción. null = sin cap.
  max_cashback_per_tx: number | null
  // FID-37: techo de cashback acumulado por periodo. null = sin cap.
  max_cashback_per_period: number | null
}

export interface CashbackReward {
  id: string
  customer_id: string
  customer_sisfi_id: string
  name: string
  description: string
  cost: number
  active: boolean
  // FID-38: stock/disponibilidad limitada. null = ilimitado.
  stock_total: number | null
  redeemed_count: number
  limit_per_client: number | null
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

export interface CustomerMetrics {
  registered_clients: number
  active_clients: number
  reactivated_clients: number
  participation_rate: number
  purchase_frequency: number
  average_ticket: number
  spend_per_client: number
  total_spend: number
  benefits_generated: number
  benefits_redeemed: number
  benefits_cost: number
  potential_gain: number
  redemption_rate: number
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
  /** Días de vida de una tarjeta desde su creación; null = sin expiración. */
  card_expiry_days: number | null
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

// MeResponse mirrors GET /api/v1/auth/me: the admin summary plus the feature
// flags resolved for the caller's customer (map of flag key → enabled) for UI
// gating. `flags` may be absent when no flag resolver is wired server-side.
export interface MeResponse extends AdminSummary {
  flags?: Record<string, boolean>
}

// FeatureFlag mirrors the admin feature-flag definition
// (GET/PUT /api/v1/admin/flags).
export interface FeatureFlag {
  key: string
  enabled_globally: boolean
  customer_overrides: Record<string, boolean>
  default_value: boolean
  description?: string
  created_at: string
  updated_at: string
}

// FeatureFlagUpdate is the PUT /api/v1/admin/flags/:key body. Omitted fields
// are left unchanged (upsert semantics).
export interface FeatureFlagUpdate {
  enabled_globally?: boolean
  customer_overrides?: Record<string, boolean>
  default_value?: boolean
  description?: string
}

export interface AuthResponse {
  token: string
  admin: AdminSummary
}
