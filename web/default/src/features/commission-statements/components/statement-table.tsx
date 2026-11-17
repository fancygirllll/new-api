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
/**
 * Shared statement view component (contract appendix D4)
 * Phase 4: admin read-only entry (scope="admin" → /api/commission/statements*, all-distributor view)
 * Phase 6: reused by the distributor console (scope="self" → /api/commission/statements/self*, DistributorAuth narrows the data)
 * Reuse pattern: pass the scope prop to switch the API base path and narrow columns; do not copy this file to derive a second implementation
 */
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import {
  fetchStatements,
  formatCents,
  formatUnixTime,
  type CommissionStatement,
  type StatementScope,
} from '../api'

const PAGE_SIZE = 10

interface StatementTableProps {
  /** admin=all-distributor view (A4 three filters); self=the distributor themselves (A10, Phase 6 reuse, hides the distributor column and filter bar) */
  scope: StatementScope
  /** Fixed distributor filter (the self case is narrowed server-side, so no need to pass it) */
  distributorId?: number
  /** Row click callback (admin opens details; Phase 6 self can also wire item navigation) */
  onOpenDetail?: (statement: CommissionStatement) => void
  /** Phase 5 additive: 'apply for withdrawal' callback on payable rows in the self view (rendered only when all three conditions hold; undefined has zero impact on admin, D4 no copy-paste derivation) */
  onApplyWithdraw?: (statement: CommissionStatement) => void
  /** 6.2 additive: 'adjust' callback on payable rows in the admin view (A8 RootAuth; undefined has zero impact) */
  onAdjust?: (statement: CommissionStatement) => void
  /** Phase 5 additive: refetch the list on change (statement status refreshes after apply/reject-rollback/settle, D-05) */
  refreshKey?: number
}

/**
 * Statement list (A4/A10): period/distributor_id/status filter bar + paginated table.
 * Minimal read-only: no export (contract D4 scope discipline); Phase 5 additive inline 'apply for withdrawal' entry.
 */
export function StatementTable({
  scope,
  distributorId,
  onOpenDetail,
  onApplyWithdraw,
  onAdjust,
  refreshKey,
}: StatementTableProps) {
  const { t } = useTranslation()
  const [rows, setRows] = useState<CommissionStatement[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [period, setPeriod] = useState('')
  const [status, setStatus] = useState('')
  const [distributorInput, setDistributorInput] = useState('')
  const [distributorFilter, setDistributorFilter] = useState<number | undefined>(
    distributorId
  )
  const [loading, setLoading] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await fetchStatements(scope, {
        page,
        pageSize: PAGE_SIZE,
        distributorId: distributorFilter,
        period: period || undefined,
        status: status || undefined,
      })
      if (res.success && res.data) {
        setRows(res.data.items ?? [])
        setTotal(res.data.total ?? 0)
      }
    } finally {
      setLoading(false)
    }
  }, [scope, page, distributorFilter, period, status])

  useEffect(() => {
    void load()
  }, [load, refreshKey])

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <div className='flex flex-col gap-4'>
      {/* Filter bar: in the self case the server already narrowed the data, so hide the distributor/period filters (column narrowing is driven by scope) */}
      {scope === 'admin' && (
        <div className='flex flex-wrap items-center gap-2'>
          <Input
            className='w-36'
            placeholder={t('Distributor ID')}
            value={distributorInput}
            onChange={(e) => setDistributorInput(e.target.value)}
            onBlur={() => {
              const v = Number(distributorInput)
              setPage(1)
              setDistributorFilter(
                distributorInput && Number.isFinite(v) && v > 0 ? v : undefined
              )
            }}
          />
          <Input
            className='w-36'
            placeholder={t('Period')}
            value={period}
            onChange={(e) => {
              setPage(1)
              setPeriod(e.target.value)
            }}
          />
          <select
            className='border-input bg-background h-9 rounded-md border px-3 text-sm'
            value={status}
            onChange={(e) => {
              setPage(1)
              setStatus(e.target.value)
            }}
          >
            <option value=''>{t('All Status')}</option>
            <option value='payable'>payable</option>
            <option value='withdrawing'>withdrawing</option>
            <option value='settled'>settled</option>
          </select>
        </div>
      )}

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('Period')}</TableHead>
            {scope === 'admin' && <TableHead>{t('Distributor ID')}</TableHead>}
            <TableHead>{t('Total Commission')}</TableHead>
            <TableHead>{t('Adjusted Amount')}</TableHead>
            <TableHead>{t('Settle Amount')}</TableHead>
            <TableHead>{t('Status')}</TableHead>
            <TableHead>{t('Created At')}</TableHead>
            <TableHead />
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.length === 0 ? (
            <TableRow>
              <TableCell
                colSpan={scope === 'admin' ? 8 : 7}
                className='text-muted-foreground h-24 text-center'
              >
                {t('No statements found')}
              </TableCell>
            </TableRow>
          ) : (
            rows.map((row) => (
              <TableRow key={row.id}>
                <TableCell>{row.period}</TableCell>
                {scope === 'admin' && (
                  <TableCell>{row.distributor_id}</TableCell>
                )}
                <TableCell>{formatCents(row.total_commission_cents)}</TableCell>
                <TableCell>{formatCents(row.adjusted_cents)}</TableCell>
                <TableCell>{formatCents(row.settle_amount_cents)}</TableCell>
                <TableCell>{row.status}</TableCell>
                <TableCell>{formatUnixTime(row.created_at)}</TableCell>
                <TableCell>
                  <div className='flex flex-wrap gap-1'>
                    {onOpenDetail && (
                      <Button
                        variant='outline'
                        size='sm'
                        onClick={() => onOpenDetail(row)}
                      >
                        {t('Detail')}
                      </Button>
                    )}
                    {/* Phase 5 additive: 'apply for withdrawal' only when self + payable + callback are all present (behavior is unchanged when undefined) */}
                    {scope === 'self' &&
                      row.status === 'payable' &&
                      onApplyWithdraw && (
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => onApplyWithdraw(row)}
                        >
                          {t('Apply Withdrawal')}
                        </Button>
                      )}
                    {/* 6.2 additive: 'adjust' only when admin + payable + callback are all present (A8 RootAuth; the callback is passed only by the assembly point for root viewers) */}
                    {scope === 'admin' &&
                      row.status === 'payable' &&
                      onAdjust && (
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => onAdjust(row)}
                        >
                          {t('Adjust')}
                        </Button>
                      )}
                  </div>
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>

      <div className='flex items-center justify-end gap-2 text-sm'>
        <span>
          {t('Page')} {page} / {totalPages} · {total} {t('Records')}
        </span>
        <Button
          variant='outline'
          size='sm'
          disabled={page <= 1 || loading}
          onClick={() => setPage((p) => Math.max(1, p - 1))}
        >
          {t('Previous')}
        </Button>
        <Button
          variant='outline'
          size='sm'
          disabled={page >= totalPages || loading}
          onClick={() => setPage((p) => p + 1)}
        >
          {t('Next')}
        </Button>
      </div>
    </div>
  )
}
