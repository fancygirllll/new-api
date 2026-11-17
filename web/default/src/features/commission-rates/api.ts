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
 * Commission rate API layer (contract 01-CONTRACT.md §4.3 A1/A2/A3 frozen shapes, RATE-01/RATE-03).
 * Endpoint mapping:
 *   fetchCommissionRates      → GET  /api/commission/rates (A1 list, flat embed including the redundant username)
 *   setCommissionRate         → PUT  /api/commission/rates/:distributorId (A2 upsert, body {"rate_bp":N})
 *   fetchCommissionRateHistory → GET /api/commission/rates/:distributorId/history (A3 change history)
 * All business validation is server-side (basis points 0~10000 / exact role=5 match / path parameter parsing);
 * the frontend relies on zero business validation and echoes the failure envelope message verbatim.
 */

/** A1 row (model.CommissionRateDetail flat JSON, json tag aligned verbatim) */
export interface CommissionRateRow {
  id: number
  distributor_id: number
  rate_bp: number
  updated_by: number
  updated_at: number
  username: string
}

/** A3 row (model.CommissionRateHistory json tag verbatim: full old_rate_bp/new_rate_bp names, no abbreviations) */
export interface RateHistoryRow {
  id: number
  distributor_id: number
  old_rate_bp: number
  new_rate_bp: number
  operator_id: number
  created_at: number
}

export type CommissionRatePageInfo<T> = StatementPageInfo<T>

/** A1 paginated rate list (AdminAuth, sorted by distributor_id ascending) */
export async function fetchCommissionRates(
  page = 1
): Promise<ApiEnvelope<CommissionRatePageInfo<CommissionRateRow>>> {
  const res = await api.get(`/api/commission/rates?p=${page}`)
  return res.data
}

/** A2 set rate (upsert semantics: Create when no row / Update when a row exists; the server's exact role=5 check prevents misbinding) */
export async function setCommissionRate(
  distributorId: number,
  rateBp: number
): Promise<ApiEnvelope<null>> {
  const res = await api.put(`/api/commission/rates/${distributorId}`, {
    rate_bp: rateBp,
  })
  return res.data
}

/** A3 paginated rate change history (old_rate_bp→new_rate_bp + operator_id + created_at, RATE-03) */
export async function fetchCommissionRateHistory(
  distributorId: number,
  page = 1
): Promise<ApiEnvelope<CommissionRatePageInfo<RateHistoryRow>>> {
  const res = await api.get(
    `/api/commission/rates/${distributorId}/history?p=${page}`
  )
  return res.data
}

/**
 * Basis points → percentage for display (the project's single bp conversion point, D-07 "same conversion"):
 * 500 → '5%', 5000 → '50%', 0 → '0%', 1250 → '12.5%'.
 * The statement items 'rate' column also imports this function (no inlining, no copy-paste derivation).
 */
export function rateBpToPercent(bp: number): string {
  const percent = bp / 100
  const normalized = Number.isInteger(percent) ? percent : Number(percent.toFixed(2))
  return `${normalized}%`
}

/** Amount/time formatting reuses the single entry from the statement domain (no copy-paste derivation, S9 single-formatter discipline) */
export { formatCents, formatUnixTime }
