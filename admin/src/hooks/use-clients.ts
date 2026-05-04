import { useQuery } from "@tanstack/react-query"
import {
  getClients,
  getClientBalance,
  getClientTransactions,
  getCashbackClientBalance,
  getCashbackClientTransactions,
} from "@/lib/api-client"

export function useClients(customerId: string) {
  return useQuery({
    queryKey: ["clients", customerId],
    queryFn: () => getClients(customerId),
    enabled: !!customerId,
  })
}

export function useClientBalance(programId: string, clientId: string) {
  return useQuery({
    queryKey: ["client-balance", programId, clientId],
    queryFn: () => getClientBalance(programId, clientId),
    enabled: !!programId && !!clientId,
  })
}

export function useClientTransactions(programId: string, clientId: string) {
  return useQuery({
    queryKey: ["client-transactions", programId, clientId],
    queryFn: () => getClientTransactions(programId, clientId),
    enabled: !!programId && !!clientId,
  })
}

export function useCashbackClientBalance(programId: string, clientId: string) {
  return useQuery({
    queryKey: ["cashback-client-balance", programId, clientId],
    queryFn: () => getCashbackClientBalance(programId, clientId),
    enabled: !!programId && !!clientId,
  })
}

export function useCashbackClientTransactions(programId: string, clientId: string) {
  return useQuery({
    queryKey: ["cashback-client-transactions", programId, clientId],
    queryFn: () => getCashbackClientTransactions(programId, clientId),
    enabled: !!programId && !!clientId,
  })
}
