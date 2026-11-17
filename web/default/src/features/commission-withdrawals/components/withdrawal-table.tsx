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
 * Shared withdrawal view component (D4 no copy-paste derivation, moved verbatim from the statement-table scope model).
 * admin=review workbench (A17 three filters + redundant username column + gated action column + view statement);
 * self=distributor withdrawal records (A16 server-narrowed, read-only + timeline + rejection reason/voucher display, SC5/D-03).
 * Reuse pattern: pass the scope prop to switch the API base path and narrow columns; do not copy this file to derive a second implementation.
 */
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
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
  fetchWithdrawals,
  formatCents,
  formatUnixTime,
  WITHDRAWAL_STATUS_BADGE_VARIANT,
  WITHDRAWAL_STATUS_LABEL_KEYS,
  WITHDRAWAL_TIMELINE_FIELDS,
  type Withdrawal,
  type WithdrawalScope,
  type WithdrawalStatus,
} from '../api'

const PAGE_SIZE = 10

interface WithdrawalTableProps {
  /** admin=review workbench (A17); self=the distributor themselves (A16, hides admin columns, filter bar, and action column) */
  scope: WithdrawalScope
  /** Accept callback (admin, rendered only on pending rows; the caller assembles the API call) */
  onAccept?: (w: Withdrawal) => void
  /** Reject callback (admin, rendered only on reviewing rows; the caller assembles the dialog) */
  onReject?: (w: Withdrawal) => void
  /** Approve callback (admin, rendered only on reviewing rows; the caller assembles the confirmation dialog) */
  onApprove?: (w: Withdrawal) => void
  /** Mark-paid callback (admin, rendered only on approved rows; the caller assembles the voucher dialog) */
  onMarkPaid?: (w: Withdrawal) => void
  /** View-statement callback (admin, rendered on rows in all states; the caller assembles StatementDetail(scope=admin) statement detail, WDRAW-03) */
  onOpenStatement?: (w: Withdrawal) => void
  /** Refetch the list on change (the caller bumps this after apply/accept/reject/approve/mark-paid) */
  refreshKey?: number
}

/** Withdrawal status badge (labels come from the constant mapping, never pass through server strings, T-05-02-02) */
function WithdrawalStatusBadge({ status }: { status: WithdrawalStatus }) {
  const { t } = useTranslation()
  return (
    <Badge variant={WITHDRAWAL_STATUS_BADGE_VARIANT[status]}>
      {t(WITHDRAWAL_STATUS_LABEL_KEYS[status])}
    </Badge>
  )
}

/** Timeline cell (five timeline columns; a value of 0 is not displayed) */
function TimelineCell({ w }: { w: Withdrawal }) {
  const { t } = useTranslation()
  const marks = WITHDRAWAL_TIMELINE_FIELDS.filter((f) => (w[f.key] as number) > 0)
  return (
    <div className='flex flex-col gap-0.5 text-xs whitespace-nowrap'>
      {marks.map((f) => (
        <div key={f.key}>
          <span className='text-muted-foreground'>{t(f.labelKey)}: </span>
          {formatUnixTime(w[f.key] as number)}
        </div>
      ))}
    </div>
  )
}

/**
 * Withdrawal list (A16/A17): admin distributor_id/status filters + pagination + state-gated action column;
 * self is read-only (timeline + rejection reason + voucher number, SC5/D-03).
 */
export function WithdrawalTable({
  scope,
  onAccept,
  onReject,
  onApprove,
  onMarkPaid,
  onOpenStatement,
  refreshKey,
}: WithdrawalTableProps) {
  const { t } = useTranslation()
  const [rows, setRows] = useState<Withdrawal[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [distributorInput, setDistributorInput] = useState('')
  const [distributorFilter, setDistributorFilter] = useState<number | undefined>(
    undefined
  )
  const [status, setStatus] = useState('')
  const [loading, setLoading] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await fetchWithdrawals(scope, {
        page,
        pageSize: PAGE_SIZE,
        distributorId: distributorFilter,
        status: status || undefined,
      })
      if (res.success && res.data) {
        setRows(res.data.items ?? [])
        setTotal(res.data.total ?? 0)
      }
    } finally {
      setLoading(false)
    }
  }, [scope, page, distributorFilter, status])

  useEffect(() => {
    void load()
  }, [load, refreshKey])

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <div className='flex flex-col gap-4'>
      {/* Filter bar: in the self case the server already narrowed the data, so hide the filters (column narrowing is driven by scope) */}
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
          <select
            className='border-input bg-background h-9 rounded-md border px-3 text-sm'
            value={status}
            onChange={(e) => {
              setPage(1)
              setStatus(e.target.value)
            }}
          >
            <option value=''>{t('All Status')}</option>
            <option value='pending'>pending</option>
            <option value='reviewing'>reviewing</option>
            <option value='approved'>approved</option>
            <option value='paid'>paid</option>
            <option value='rejected'>rejected</option>
          </select>
        </div>
      )}

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('Withdrawal No')}</TableHead>
            {scope === 'admin' && <TableHead>{t('Distributor')}</TableHead>}
            <TableHead>{t('Period')}</TableHead>
            <TableHead>{t('Settle Amount')}</TableHead>
            <TableHead>{t('Status')}</TableHead>
            <TableHead>{t('Reject Reason')}</TableHead>
            <TableHead>{t('Voucher No')}</TableHead>
            {scope === 'self' && <TableHead>{t('Timeline')}</TableHead>}
            {scope === 'admin' && <TableHead>{t('Actions')}</TableHead>}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.length === 0 ? (
            <TableRow>
              <TableCell
                colSpan={scope === 'admin' ? 8 : 7}
                className='text-muted-foreground h-24 text-center'
              >
                {t('No withdrawals found')}
              </TableCell>
            </TableRow>
          ) : (
            rows.map((row) => (
              <TableRow key={row.id}>
                <TableCell className='font-mono'>{row.withdrawal_no}</TableCell>
                {scope === 'admin' && (
                  <TableCell>{row.username || row.distributor_id}</TableCell>
                )}
                <TableCell>{row.period}</TableCell>
                <TableCell>{formatCents(row.settle_amount_cents)}</TableCell>
                <TableCell>
                  <WithdrawalStatusBadge status={row.status} />
                </TableCell>
                <TableCell className='max-w-48 break-words'>
                  {row.status === 'rejected' && row.reason ? row.reason : '-'}
                </TableCell>
                <TableCell className='font-mono'>
                  {row.status === 'paid' && row.voucher_no
                    ? row.voucher_no
                    : '-'}
                </TableCell>
                {scope === 'self' && (
                  <TableCell>
                    <TimelineCell w={row} />
                  </TableCell>
                )}
                {scope === 'admin' && (
                  <TableCell>
                    <div className='flex flex-wrap gap-1'>
                      {row.status === 'pending' && onAccept && (
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => onAccept(row)}
                        >
                          {t('Accept')}
                        </Button>
                      )}
                      {row.status === 'reviewing' && onReject && (
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => onReject(row)}
                        >
                          {t('Reject')}
                        </Button>
                      )}
                      {row.status === 'reviewing' && onApprove && (
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => onApprove(row)}
                        >
                          {t('Approve')}
                        </Button>
                      )}
                      {row.status === 'approved' && onMarkPaid && (
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => onMarkPaid(row)}
                        >
                          {t('Mark Paid')}
                        </Button>
                      )}
                      {onOpenStatement && (
                        <Button
                          variant='ghost'
                          size='sm'
                          onClick={() => onOpenStatement(row)}
                        >
                          {t('View Statement')}
                        </Button>
                      )}
                    </div>
                  </TableCell>
                )}
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
