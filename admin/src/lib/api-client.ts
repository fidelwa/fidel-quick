import type {
  Customer,
  Sisfi,
  CustomerSisfi,
  Program,
  Reward,
  CashbackProgram,
  CashbackReward,
  Client,
  Collaborator,
  FeedbackEntry,
  Transaction,
  CashbackTransaction,
  Balance,
  PushcardConfig,
  PushcardCard,
  AdminSummary,
  MeResponse,
  FeatureFlag,
  FeatureFlagUpdate,
  AuthResponse,
  OnboardingRegisterRequest,
  GoogleOnboardingRequest,
  OnboardingStatus,
} from "@/types"

const BASE_URL = import.meta.env.VITE_API_URL || "/api/v1"

let authToken = ""

export function setToken(token: string) {
  authToken = token
}

export function getToken() {
  return authToken
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(authToken ? { Authorization: `Bearer ${authToken}` } : {}),
      ...options.headers,
    },
  })

  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error || `Request failed: ${res.status}`)
  }

  if (res.status === 204) return undefined as T
  return res.json()
}

// Customers
export const getCustomer = (id: string) =>
  request<Customer>(`/customers/${id}`)

export const updateCustomer = (id: string, data: Partial<Customer>) =>
  request<Customer>(`/customers/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  })

// SISFI catalog & customer-sisfi
export const getSisfi = () =>
  request<Sisfi[]>(`/sisfi`)

export const getCustomerSisfi = (customerId: string) =>
  request<CustomerSisfi[]>(`/customer-sisfi?customer_id=${customerId}`)

export const createCustomerSisfi = (data: { customer_id: string; sisfi_id: string; name: string }) =>
  request<CustomerSisfi>(`/customer-sisfi`, {
    method: "POST",
    body: JSON.stringify(data),
  })

export const updateCustomerSisfi = (id: string, data: Partial<Pick<CustomerSisfi, "name" | "active">>) =>
  request<void>(`/customer-sisfi/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  })

// Programs (earn-burn)
export const getPrograms = (customerId: string) =>
  request<Program[]>(`/programs?customer_id=${customerId}`)

export const createProgram = (data: { customer_id: string; name: string; points_ratio: number }) =>
  request<Program>(`/programs`, {
    method: "POST",
    body: JSON.stringify(data),
  })

export const updateProgram = (
  id: string,
  data: Partial<Pick<Program, "name" | "points_ratio" | "active" | "expiry_days" | "min_ticket_amount">>,
) =>
  request<Program>(`/programs/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  })

// Rewards (earn-burn)
export const getRewards = (programId: string) =>
  request<Reward[]>(`/programs/${programId}/rewards`)

export const createReward = (programId: string, data: { name: string; description: string; points_cost: number }) =>
  request<Reward>(`/programs/${programId}/rewards`, {
    method: "POST",
    body: JSON.stringify(data),
  })

export const updateReward = (programId: string, rewardId: string, data: Partial<Pick<Reward, "name" | "description" | "points_cost" | "active">>) =>
  request<Reward>(`/programs/${programId}/rewards/${rewardId}`, {
    method: "PUT",
    body: JSON.stringify(data),
  })

// Cashback Programs
export const getCashbackPrograms = (customerId: string) =>
  request<CashbackProgram[]>(`/cashback-programs?customer_id=${customerId}`)

export const createCashbackProgram = (data: { customer_id: string; name: string; cashback_rate: number }) =>
  request<CashbackProgram>(`/cashback-programs`, {
    method: "POST",
    body: JSON.stringify(data),
  })

export const updateCashbackProgram = (
  id: string,
  data: Partial<
    Pick<
      CashbackProgram,
      | "name"
      | "cashback_rate"
      | "active"
      | "expiry_days"
      | "min_ticket_amount"
      | "max_cashback_per_tx"
      | "max_cashback_per_period"
    >
  >,
) =>
  request<CashbackProgram>(`/cashback-programs/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  })

// Cashback Rewards
export const getCashbackRewards = (programId: string) =>
  request<CashbackReward[]>(`/cashback-programs/${programId}/rewards`)

export const createCashbackReward = (programId: string, data: { name: string; description: string; cost: number }) =>
  request<CashbackReward>(`/cashback-programs/${programId}/rewards`, {
    method: "POST",
    body: JSON.stringify(data),
  })

export const updateCashbackReward = (programId: string, rewardId: string, data: Partial<Pick<CashbackReward, "name" | "description" | "cost" | "active">>) =>
  request<CashbackReward>(`/cashback-programs/${programId}/rewards/${rewardId}`, {
    method: "PUT",
    body: JSON.stringify(data),
  })

// Clients
export const getClients = (customerId: string) =>
  request<Client[]>(`/customers/${customerId}/clients`)

// Collaborators
export const getCollaborators = (customerId: string) =>
  request<Collaborator[]>(`/customers/${customerId}/collaborators`)

export const createCollaborator = (customerId: string, data: { name: string; phone: string }) =>
  request<Collaborator>(`/customers/${customerId}/collaborators`, {
    method: "POST",
    body: JSON.stringify(data),
  })

// Client data
export const getClientBalance = (programId: string, clientId: string) =>
  request<Balance>(`/programs/${programId}/clients/${clientId}/balance`)

export const getClientTransactions = (programId: string, clientId: string) =>
  request<Transaction[]>(`/programs/${programId}/clients/${clientId}/transactions`)

export const getCashbackClientBalance = (programId: string, clientId: string) =>
  request<Balance>(`/cashback-programs/${programId}/clients/${clientId}/balance`)

export const getCashbackClientTransactions = (programId: string, clientId: string) =>
  request<CashbackTransaction[]>(`/cashback-programs/${programId}/clients/${clientId}/transactions`)

// Feedback
export const getFeedback = (customerId: string) =>
  request<FeedbackEntry[]>(`/customers/${customerId}/feedback`)

// Auth
export const loginAdmin = (email: string, password: string) =>
  request<AuthResponse>(`/auth/login`, {
    method: "POST",
    body: JSON.stringify({ email, password }),
  })

export const registerAdmin = (email: string, password: string, customer_id: string) =>
  request<AuthResponse>(`/auth/register`, {
    method: "POST",
    body: JSON.stringify({ email, password, customer_id }),
  })

// Password reset (FID-16)
// forgotPassword responde 200 siempre (no revela si el email existe).
export const forgotPassword = (email: string) =>
  request<{ message: string }>(`/auth/forgot-password`, {
    method: "POST",
    body: JSON.stringify({ email }),
  })

export const resetPassword = (token: string, new_password: string) =>
  request<{ message: string }>(`/auth/reset-password`, {
    method: "POST",
    body: JSON.stringify({ token, new_password }),
  })

// Onboarding
export const onboardingRegister = (data: OnboardingRegisterRequest) =>
  request<AuthResponse>(`/onboarding/register`, {
    method: "POST",
    body: JSON.stringify(data),
  })

export const onboardingGoogle = (data: GoogleOnboardingRequest) =>
  request<AuthResponse>(`/onboarding/register/google`, {
    method: "POST",
    body: JSON.stringify(data),
  })

// Verifica si el teléfono ya está en uso por algún customer activo.
// Endpoint público (sin auth), no expone qué negocio es.
export const checkPhoneExists = (phone: string) =>
  request<{ exists: boolean }>(`/onboarding/phone-check?phone=${encodeURIComponent(phone)}`)

// Google Auth
export const loginGoogle = (googleToken: string) =>
  request<AuthResponse>(`/auth/login/google`, {
    method: "POST",
    body: JSON.stringify({ google_token: googleToken }),
  })

export const linkGoogle = (googleToken: string) =>
  request<AdminSummary>(`/auth/link/google`, {
    method: "POST",
    body: JSON.stringify({ google_token: googleToken }),
  })

export const unlinkGoogle = () =>
  request<AdminSummary>(`/auth/link/google`, {
    method: "DELETE",
  })

export const getMe = () => request<MeResponse>(`/auth/me`)

// Feature flags (admin — JWT authenticated)
export const getFeatureFlags = () =>
  request<FeatureFlag[]>(`/admin/flags`)

export const updateFeatureFlag = (key: string, data: FeatureFlagUpdate) =>
  request<FeatureFlag>(`/admin/flags/${encodeURIComponent(key)}`, {
    method: "PUT",
    body: JSON.stringify(data),
  })

// Onboarding
export const getOnboarding = () =>
  request<OnboardingStatus>(`/onboarding`)

export const updateOnboardingStep = (step: number) =>
  request<OnboardingStatus>(`/onboarding/step`, {
    method: "PUT",
    body: JSON.stringify({ step }),
  })

export const completeOnboarding = () =>
  request<OnboardingStatus>(`/onboarding/complete`, {
    method: "POST",
  })


// Pushcard
export const getPushcardConfig = (customerId: string) =>
  request<PushcardConfig>(`/pushcard/config?customer_id=${customerId}`)

export const upsertPushcardConfig = (
  customerSisfiID: string,
  data: {
    card_slots: number
    reward_on_complete?: string
    card_expiry_days?: number | null
  }
) =>
  request<PushcardConfig>(`/pushcard/programs/${customerSisfiID}/config`, {
    method: "PUT",
    body: JSON.stringify(data),
  })

export const getPushcardCards = (
  customerSisfiID: string,
  status?: "open" | "completed" | "redeemed" | "cancelled",
  limit = 50
) => {
  const qs = new URLSearchParams()
  if (status) qs.set("status", status)
  qs.set("limit", String(limit))
  return request<PushcardCard[]>(`/pushcard/programs/${customerSisfiID}/cards?${qs.toString()}`)
}
