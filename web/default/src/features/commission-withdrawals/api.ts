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
 * Withdrawal API layer (contract 01-CONTRACT.md appendix E2 frozen shapes A15-A21, v1.4).
 * There is no separate amount field (D-08): settle_amount_cents is taken live from the statement via a backend JOIN,
 * and the display layer reuses formatCents (contract §3.0 prohibits float arithmetic).
 */

/** Five withdrawal states (appendix E1 frozen; Q1=B approved is a separate state, Q2=B reject can only be initiated from reviewing) */
export type WithdrawalStatus =
  | 'pending'
  | 'reviewing'
  | 'approved'
  | 'paid'
  | 'rejected'

/** View: admin=review workbench (A17-A21), self=distributor (A15/A16) */
export type WithdrawalScope = 'admin' | 'self'

/** Withdrawal (A16/A17 WithdrawalDetail flat snake_case, appendix E2 frozen fields aligned verbatim) */
export interface Withdrawal {
  id: number
  withdrawal_no: string
  statement_id: number
  distributor_id: number
  status: WithdrawalStatus
  reason: string
  voucher_no: string
  operator_id: number
  created_at: number
  reviewed_at: number
  approved_at: number
  paid_at: number
  rejected_at: number
  /** admin list JOINs users redundantly (A17); may be absent in the self response */
  username?: string
  /** JOINed statement redundancy (D-08 live value) */
  period: string
  settle_amount_cents: number
}

export type WithdrawalPageInfo<T> = StatementPageInfo<T>

/** Five states → i18n key (dynamic labels render via t(key); the constant mapping never passes through server strings, T-05-02-02) */
export const WITHDRAWAL_STATUS_LABEL_KEYS: Record<WithdrawalStatus, string> = {
  pending: 'Withdrawal Pending',
  reviewing: 'Withdrawal Reviewing',
  approved: 'Withdrawal Approved',
  paid: 'Withdrawal Paid',
  rejected: 'Withdrawal Rejected',
}

/** Five states → badge variant (visual state distinction; rejected stands out in red) */
export const WITHDRAWAL_STATUS_BADGE_VARIANT: Record<
  WithdrawalStatus,
  'default' | 'secondary' | 'outline' | 'destructive'
> = {
  pending: 'secondary',
  reviewing: 'default',
  approved: 'outline',
  paid: 'default',
  rejected: 'destructive',
}

/** Five timeline columns (appendix E1; a value of 0 is not displayed) */
export const WITHDRAWAL_TIMELINE_FIELDS: Array<{
  key: keyof Withdrawal
  labelKey: string
}> = [
  { key: 'created_at', labelKey: 'Applied At' },
  { key: 'reviewed_at', labelKey: 'Reviewed At' },
  { key: 'approved_at', labelKey: 'Approved At' },
  { key: 'paid_at', labelKey: 'Paid At' },
  { key: 'rejected_at', labelKey: 'Rejected At' },
]

/** scope → withdrawal list base path (admin=A17; self=A16, the server forces WHERE distributor_id=self) */
export function withdrawalsBasePath(scope: WithdrawalScope): string {
  return scope === 'self'
    ? '/api/commission/withdrawals/self'
    : '/api/commission/withdrawals'
}

export interface WithdrawalListParams {
  page?: number
  pageSize?: number
  /** Supported only on the admin route (A17); self ignores anything passed by the client */
  distributorId?: number
  status?: string
}

/** A16/A17 withdrawal list (optional distributor_id/status filters, sorted id DESC with the newest application first) */
export async function fetchWithdrawals(
  scope: WithdrawalScope,
  params: WithdrawalListParams = {}
): Promise<ApiEnvelope<WithdrawalPageInfo<Withdrawal>>> {
  const q = new URLSearchParams()
  if (params.page) q.set('p', String(params.page))
  if (params.pageSize) q.set('page_size', String(params.pageSize))
  if (params.distributorId) {
    q.set('distributor_id', String(params.distributorId))
  }
  if (params.status) q.set('status', params.status)
  const res = await api.get(`${withdrawalsBasePath(scope)}?${q.toString()}`)
  return res.data
}

/** A15 start a withdrawal application (full amount per statement, D-08: no amount input in the request; response {success} frozen shape) */
export async function applyWithdraw(
  statementId: number
): Promise<ApiEnvelope<null>> {
  const res = await api.post('/api/commission/withdrawals', {
    statement_id: statementId,
  })
  return res.data
}

/** A18 accept (pending→reviewing, no statement operation) */
export async function acceptWithdrawal(
  id: number
): Promise<ApiEnvelope<null>> {
  const res = await api.post(`/api/commission/withdrawals/${id}/accept`)
  return res.data
}

/** A19 reject (reviewing→rejected, reason required; the same transaction rolls the statement back to payable, D-05) */
export async function rejectWithdrawal(
  id: number,
  reason: string
): Promise<ApiEnvelope<null>> {
  const res = await api.post(`/api/commission/withdrawals/${id}/reject`, {
    reason,
  })
  return res.data
}

/** A20 approve (reviewing→approved, the statement stays withdrawing, Q1=B) */
export async function approveWithdrawal(
  id: number
): Promise<ApiEnvelope<null>> {
  const res = await api.post(`/api/commission/withdrawals/${id}/approve`)
  return res.data
}

/** A21 mark paid (approved→paid, voucher_no required; the same transaction settles the statement, D-05) */
export async function markWithdrawalPaid(
  id: number,
  voucherNo: string
): Promise<ApiEnvelope<null>> {
  const res = await api.post(`/api/commission/withdrawals/${id}/paid`, {
    voucher_no: voucherNo,
  })
  return res.data
}

/** Amount/time formatting reuses the single entry from the statement domain (no copy-paste derivation) */
export { formatCents, formatUnixTime }
