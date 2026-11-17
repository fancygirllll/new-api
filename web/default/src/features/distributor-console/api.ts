/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { api } from '@/lib/api'

import {
  formatCents,
  formatUnixTime,
  type ApiEnvelope,
  type StatementPageInfo,
} from '@/features/commission-statements/api'

/**
 * Distributor console API layer (contract 01-CONTRACT.md §4.3 A9/A12 + §4.2 B4/B5, v1.5 appendix F).
 * Response interfaces match the backend json tags verbatim (06-01 appendix F frozen shapes);
 * Amount-unit discipline: A9/A12/B4 are int64 cents; conversion happens only in the display layer via formatCents;
 * B5 topup rows use money as float yuan (model/topup.go); the display must not use the cents conversion entry.
 */

/** A9 commission dashboard (four frozen cent fields verbatim + appendix F2 four-count extension) */
export interface CommissionDashboard {
  current_pending_cents: number
  total_statement_cents: number
  total_withdrawn_cents: number
  pending_withdraw_cents: number
  current_pending_count: number
  total_statement_count: number
  total_withdrawn_count: number
  pending_withdraw_count: number
}

/** A12 unbilled real-time preview (three frozen fields verbatim, §4.3) */
export interface CurrentCommission {
  period: string
  pending_flows: number
  pending_cents: number
}

/** A9 commission dashboard (DistributorAuth session-derived, zero client parameters) */
export async function fetchDashboard(): Promise<
  ApiEnvelope<CommissionDashboard>
> {
  const res = await api.get('/api/commission/dashboard')
  return res.data
}

/** A12 unbilled real-time preview (same predicate as A9 current unbilled; D-05 keeps them separate) */
export async function fetchCurrent(): Promise<ApiEnvelope<CurrentCommission>> {
  const res = await api.get('/api/commission/current')
  return res.data
}

/** B4 customer list row (controller/distributor.go distributorCustomerItem json tag verbatim) */
export interface DistributorCustomerRow {
  id: number
  username: string
  display_name: string
  created_at: number
  total_topup_cents: number
  total_commission_cents: number
}

export type DistributorCustomerPageInfo<T> = StatementPageInfo<T>

/** B4 customers under this distributor (server-enforced attribution filter, paginated envelope) */
export async function fetchDistributorCustomers(
  page?: number
): Promise<ApiEnvelope<DistributorCustomerPageInfo<DistributorCustomerRow>>> {
  const q = new URLSearchParams()
  if (page) q.set('p', String(page))
  const res = await api.get(`/api/distributor/customers?${q.toString()}`)
  return res.data
}

/** B5 customer topup row (model/topup.go TopUp json tag verbatim; money is float yuan, not cents) */
export interface CustomerTopupRow {
  id: number
  user_id: number
  amount: number
  money: number
  trade_no: string
  payment_method: string
  payment_provider: string
  create_time: number
  complete_time: number
  status: string
}

export type CustomerTopupPageInfo<T> = StatementPageInfo<T>

/** B5 single-customer topup detail (IDOR: server returns 403 for customers not under this distributor) */
export async function fetchDistributorCustomerTopups(
  customerId: number,
  page?: number
): Promise<ApiEnvelope<CustomerTopupPageInfo<CustomerTopupRow>>> {
  const q = new URLSearchParams()
  if (page) q.set('p', String(page))
  const res = await api.get(
    `/api/distributor/customers/${customerId}/topups?${q.toString()}`
  )
  return res.data
}

/** B6 distributor profile (§4.2 B6 json tag verbatim; aff_link is the complete link assembled server-side, used as-is) */
export interface DistributorProfile {
  aff_code: string
  aff_link: string
  customer_count: number
}

/** B6 this distributor's own profile (DistributorAuth session-derived, zero client parameters) */
export async function fetchDistributorProfile(): Promise<
  ApiEnvelope<DistributorProfile>
> {
  const res = await api.get('/api/distributor/profile')
  return res.data
}

/** Five topup statuses (common/constants.go TopUpStatus* verbatim) */
export type TopupStatus =
  | 'pending'
  | 'success'
  | 'failed'
  | 'expired'
  | 'refunded'

/** Five statuses → i18n key (constant mapping, never pass through server strings; same discipline as T-06-02-03) */
export const TOPUP_STATUS_LABEL_KEYS: Record<TopupStatus, string> = {
  pending: 'Topup Pending',
  success: 'Topup Success',
  failed: 'Topup Failed',
  expired: 'Topup Expired',
  refunded: 'Topup Refunded',
}

/** Five statuses → badge variant (success stands out; failure/refund are warnings) */
export const TOPUP_STATUS_BADGE_VARIANT: Record<
  TopupStatus,
  'default' | 'secondary' | 'outline' | 'destructive'
> = {
  pending: 'secondary',
  success: 'default',
  failed: 'destructive',
  expired: 'outline',
  refunded: 'destructive',
}

/** Amount/time formatting reuses the single entry from the statement domain (no copy-paste derivation, S9) */
export { formatCents, formatUnixTime }
