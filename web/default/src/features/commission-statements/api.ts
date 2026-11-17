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

/**
 * Statement API layer (contract 01-CONTRACT.md §4.3 A4/A5/A6 + appendix D3 A14 frozen shapes).
 * Amounts are int64 cents throughout the chain (contract §3.0); only the display layer divides by 100 via formatCents, with no float arithmetic.
 */

/** View: admin=all distributors (A4/A5/A6/A14), self=distributor-scoped data (A10/A11, reused in Phase 6) */
export type StatementScope = 'admin' | 'self'

export interface CommissionStatement {
  id: number
  distributor_id: number
  period: string
  total_topup_cents: number
  total_commission_cents: number
  adjusted_cents: number
  settle_amount_cents: number
  status: string
  created_at: number
  locked_at: number
}

export interface StatementAdjustment {
  id: number
  statement_id: number
  delta_cents: number
  reason: string
  operator_id: number
  created_at: number
}

export interface StatementItemWithUsername {
  id: number
  statement_id: number
  flow_id: number
  billing_no: string
  payment_method: string
  payment_provider: string
  topup_money_cents: number
  commission_cents: number
  customer_id: number
  complete_time: number
  username: string
  /** Contract v1.5 appendix F1: rate snapshot (basis points) joined redundantly via the flow table; 0 when there is no linked flow (flow_id=0/manual/reversal) */
  rate_bp: number
}

export interface StatementSummaryItem {
  distributor_id: number
  username: string
  total_commission_cents: number
  adjusted_cents: number
  statement_status: string
  statement_id: number
}

export interface StatementSummaryTotals {
  total_commission_cents: number
  adjusted_cents: number
  distributor_count: number
}

export interface StatementSummaryData {
  period: string
  items: StatementSummaryItem[]
  totals: StatementSummaryTotals
}

export interface StatementPageInfo<T> {
  page: number
  page_size: number
  total: number
  items: T[]
}

export interface ApiEnvelope<T> {
  success: boolean
  message: string
  data?: T
}

/** scope → statement list base path (admin=A4; self=A10 scoped to the distributor, a Phase 6 reuse point) */
export function statementsBasePath(scope: StatementScope): string {
  return scope === 'self'
    ? '/api/commission/statements/self'
    : '/api/commission/statements'
}

/** scope → statement items path (admin=A6; self=A11, contract §4.3 frozen path) */
export function statementItemsPath(
  scope: StatementScope,
  statementId: number
): string {
  return scope === 'self'
    ? `/api/commission/statements/self/${statementId}/items`
    : `/api/commission/statements/${statementId}/items`
}

export interface StatementListParams {
  page?: number
  pageSize?: number
  distributorId?: number
  period?: string
  status?: string
}

/** A4/A10 statement list (optional distributor_id/period/status filters, supported only on the admin route) */
export async function fetchStatements(
  scope: StatementScope,
  params: StatementListParams = {}
): Promise<ApiEnvelope<StatementPageInfo<CommissionStatement>>> {
  const q = new URLSearchParams()
  if (params.page) q.set('p', String(params.page))
  if (params.pageSize) q.set('page_size', String(params.pageSize))
  if (params.distributorId) {
    q.set('distributor_id', String(params.distributorId))
  }
  if (params.period) q.set('period', params.period)
  if (params.status) q.set('status', params.status)
  const res = await api.get(
    `${statementsBasePath(scope)}?${q.toString()}`
  )
  return res.data
}

/** A5 statement detail (body + adjustments array). Contract has an admin route only; for self the row data from A10 is assembled (Phase 6 reuse point) */
export async function fetchStatementDetail(
  statementId: number
): Promise<
  ApiEnvelope<{ statement: CommissionStatement; adjustments: StatementAdjustment[] }>
> {
  const res = await api.get(`/api/commission/statements/${statementId}`)
  return res.data
}

/** A6/A11 paginated statement items (items include the redundant customer username) */
export async function fetchStatementItems(
  scope: StatementScope,
  statementId: number,
  page = 1,
  pageSize = 10
): Promise<ApiEnvelope<StatementPageInfo<StatementItemWithUsername>>> {
  const res = await api.get(
    `${statementItemsPath(scope, statementId)}?p=${page}&page_size=${pageSize}`
  )
  return res.data
}

/** A14 cross-distributor summary (contract appendix D3 frozen shape, AdminAuth) */
export async function fetchStatementSummary(
  period: string
): Promise<ApiEnvelope<StatementSummaryData>> {
  const res = await api.get(
    `/api/commission/statements/summary?period=${encodeURIComponent(period)}`
  )
  return res.data
}

/** A7 flow types (manual_credit top-up / manual_debit reversal, model-layer whitelist) */
export type ManualFlowType = 'manual_credit' | 'manual_debit'

/**
 * A7 super-admin manual credit/reversal flow (RootAuth, contract §4.3 frozen shape, do not add or remove fields).
 * amount_cents is always positive (the server stores debit as negative); reason is required for the audit trail.
 * Failure envelope message is echoed verbatim (reason cannot be empty / flow_type whitelist / target role validation text).
 */
export async function createManualFlow(
  distributorId: number,
  flowType: ManualFlowType,
  amountCents: number,
  reason: string
): Promise<ApiEnvelope<null>> {
  const res = await api.post('/api/commission/flows/manual', {
    distributor_id: distributorId,
    flow_type: flowType,
    amount_cents: amountCents,
    reason,
  })
  return res.data
}

/**
 * A8 super-admin statement adjustment (RootAuth, contract §4.3 frozen shape, do not add or remove fields).
 * delta_cents may be negative (positive adds, negative subtracts); only status=payable is allowed (the server whitelist rejects settled/withdrawing);
 * the server recomputes the settle amount from the statement's current totals and does not trust the client.
 */
export async function createStatementAdjustment(
  statementId: number,
  deltaCents: number,
  reason: string
): Promise<ApiEnvelope<null>> {
  const res = await api.post(
    `/api/commission/statements/${statementId}/adjustments`,
    {
      delta_cents: deltaCents,
      reason,
    }
  )
  return res.data
}

/** Cents → yuan for display (display-layer conversion only; transport/computation stays int64 cents, no float accumulation) */
export function formatCents(cents: number): string {
  return (cents / 100).toLocaleString('zh-CN', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
}

/** Unix seconds → local time for display */
export function formatUnixTime(seconds: number): string {
  if (!seconds) return '-'
  return new Date(seconds * 1000).toLocaleString('zh-CN', { hour12: false })
}
