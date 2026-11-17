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
 * Phase 4: admin read-only entry (scope="admin" → A5 detail + A6 items, all-distributor view)
 * Phase 6: reused by the distributor console (scope="self" → A11 items /api/commission/statements/self/:id/items,
 *          the statement body is passed directly from A10 list rows--the contract has no self detail endpoint, so the adjustments block is admin-only)
 * Reuse pattern: pass the scope prop to switch the items API base path; do not copy this file to derive a second implementation
 */
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import { rateBpToPercent } from '@/features/commission-rates/api'

import {
  fetchStatementDetail,
  fetchStatementItems,
  formatCents,
  formatUnixTime,
  type CommissionStatement,
  type StatementAdjustment,
  type StatementItemWithUsername,
  type StatementScope,
} from '../api'

const ITEM_PAGE_SIZE = 10

interface StatementDetailProps {
  scope: StatementScope
  /** Statement body (from list row data; also applies to the self case, a Phase 6 reuse point) */
  statement: CommissionStatement
  onClose?: () => void
}

/**
 * Statement detail (A5/A6): body fields + adjustments timeline (admin only) + paginated topup items
 * (order no./payment method/topup amount/commission amount/customer/rate/completed time). Minimal read-only, no adjustment actions (A7/A8 UI is not in this phase).
 */
export function StatementDetail({ scope, statement, onClose }: StatementDetailProps) {
  const { t } = useTranslation()
  const [adjustments, setAdjustments] = useState<StatementAdjustment[]>([])
  const [items, setItems] = useState<StatementItemWithUsername[]>([])
  const [itemsTotal, setItemsTotal] = useState(0)
  const [itemsPage, setItemsPage] = useState(1)
  const [loading, setLoading] = useState(false)

  // Reset pagination and stale data when switching the target statement (statement.id changes): this component is a non-modal inline panel,
  // and the parent swaps statement while reusing the same instance; without a reset, statement A's old page/items/adjustments would be attached under statement B's title
  useEffect(() => {
    setItemsPage(1)
    setItems([])
    setItemsTotal(0)
    setAdjustments([])
  }, [statement.id])

  // adjustments are fetchable by admin only (A5 AdminAuth); the self contract has no corresponding endpoint, so keep an empty array
  useEffect(() => {
    if (scope !== 'admin') return
    let cancelled = false
    void fetchStatementDetail(statement.id).then((res) => {
      if (!cancelled && res.success && res.data) {
        setAdjustments(res.data.adjustments ?? [])
      }
    })
    return () => {
      cancelled = true
    }
  }, [scope, statement.id])

  const loadItems = useCallback(async () => {
    setLoading(true)
    try {
      const res = await fetchStatementItems(
        scope,
        statement.id,
        itemsPage,
        ITEM_PAGE_SIZE
      )
      if (res.success && res.data) {
        setItems(res.data.items ?? [])
        setItemsTotal(res.data.total ?? 0)
      }
    } finally {
      setLoading(false)
    }
  }, [scope, statement.id, itemsPage])

  useEffect(() => {
    void loadItems()
  }, [loadItems])

  const itemTotalPages = Math.max(1, Math.ceil(itemsTotal / ITEM_PAGE_SIZE))

  return (
    <div className='flex flex-col gap-4 rounded-lg border p-4'>
      <div className='flex items-center justify-between'>
        <h3 className='text-base font-semibold'>
          {t('Statement Detail')} · {statement.period}
        </h3>
        {onClose && (
          <Button variant='ghost' size='sm' onClick={onClose}>
            {t('Close')}
          </Button>
        )}
      </div>

      <div className='grid grid-cols-2 gap-2 text-sm md:grid-cols-4'>
        <div>
          <span className='text-muted-foreground'>{t('Period')}: </span>
          {statement.period}
        </div>
        {scope === 'admin' && (
          <div>
            <span className='text-muted-foreground'>{t('Distributor ID')}: </span>
            {statement.distributor_id}
          </div>
        )}
        <div>
          <span className='text-muted-foreground'>{t('Total Commission')}: </span>
          {formatCents(statement.total_commission_cents)}
        </div>
        <div>
          <span className='text-muted-foreground'>{t('Adjusted Amount')}: </span>
          {formatCents(statement.adjusted_cents)}
        </div>
        <div>
          <span className='text-muted-foreground'>{t('Settle Amount')}: </span>
          {formatCents(statement.settle_amount_cents)}
        </div>
        <div>
          <span className='text-muted-foreground'>{t('Status')}: </span>
          {statement.status}
        </div>
        <div>
          <span className='text-muted-foreground'>{t('Created At')}: </span>
          {formatUnixTime(statement.created_at)}
        </div>
      </div>

      {/* adjustments timeline (admin view; A8 adjustment entries produced by the 04-03 API) */}
      {scope === 'admin' && (
        <div className='flex flex-col gap-1'>
          <span className='text-sm font-medium'>{t('Adjustments')}</span>
          {adjustments.length === 0 ? (
            <span className='text-muted-foreground text-sm'>
              {t('No adjustments')}
            </span>
          ) : (
            <div className='flex flex-col gap-1 border-l-2 pl-3'>
              {adjustments.map((adj) => (
                <div key={adj.id} className='text-sm'>
                  <span className='font-mono'>
                    {adj.delta_cents >= 0 ? '+' : ''}
                    {formatCents(adj.delta_cents)}
                  </span>
                  <span className='text-muted-foreground'>
                    {' '}
                    · {adj.reason} · {formatUnixTime(adj.created_at)}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Paginated topup items table (A6/A11, includes the redundant customer username) */}
      <div className='flex flex-col gap-2'>
        <span className='text-sm font-medium'>{t('Statement Items')}</span>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('Billing No')}</TableHead>
              <TableHead>{t('Payment Method')}</TableHead>
              <TableHead>{t('Topup Amount')}</TableHead>
              <TableHead>{t('Commission Amount')}</TableHead>
              <TableHead>{t('Customer')}</TableHead>
              <TableHead>{t('Rate')}</TableHead>
              <TableHead>{t('Complete Time')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} className='h-16 text-center'>
                  {t('No statements found')}
                </TableCell>
              </TableRow>
            ) : (
              items.map((item) => (
                <TableRow key={item.id}>
                  <TableCell className='font-mono'>{item.billing_no}</TableCell>
                  <TableCell>{item.payment_method}</TableCell>
                  <TableCell>{formatCents(item.topup_money_cents)}</TableCell>
                  <TableCell>{formatCents(item.commission_cents)}</TableCell>
                  <TableCell>{item.username || item.customer_id}</TableCell>
                  {/* Rate column (SC-7): both scopes display it with zero branching; show a placeholder when there is no linked flow (value 0); conversion goes through the single function, never inline */}
                  <TableCell>
                    {item.rate_bp ? rateBpToPercent(item.rate_bp) : '-'}
                  </TableCell>
                  <TableCell>{formatUnixTime(item.complete_time)}</TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
        <div className='flex items-center justify-end gap-2 text-sm'>
          <span>
            {t('Page')} {itemsPage} / {itemTotalPages} · {itemsTotal}{' '}
            {t('Records')}
          </span>
          <Button
            variant='outline'
            size='sm'
            disabled={itemsPage <= 1 || loading}
            onClick={() => setItemsPage((p) => Math.max(1, p - 1))}
          >
            {t('Previous')}
          </Button>
          <Button
            variant='outline'
            size='sm'
            disabled={itemsPage >= itemTotalPages || loading}
            onClick={() => setItemsPage((p) => p + 1)}
          >
            {t('Next')}
          </Button>
        </div>
      </div>
    </div>
  )
}
