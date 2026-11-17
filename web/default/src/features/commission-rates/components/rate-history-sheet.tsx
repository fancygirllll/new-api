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
import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'

import {
  fetchCommissionRateHistory,
  formatUnixTime,
  rateBpToPercent,
  type RateHistoryRow,
} from '../api'

const PAGE_SIZE = 10

interface RateHistorySheetProps {
  /** Target distributor (null=closed) */
  distributorId: number | null
  onOpenChange: (open: boolean) => void
}

/**
 * Rate change history drawer (A3, RATE-03): renders the change timeline row by row
 * old_rate_bp→new_rate_bp (both sides go through the single conversion function) + operator_id + created_at.
 * Field names follow model/commission_rate.go json tags verbatim (full old_rate_bp/new_rate_bp names, no abbreviations).
 */
export function RateHistorySheet({
  distributorId,
  onOpenChange,
}: RateHistorySheetProps) {
  const { t } = useTranslation()
  const [rows, setRows] = useState<RateHistoryRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  // Request sequence guard: only the latest request may write back state, preventing an in-flight response for a previous target from overwriting new data out of order
  const reqSeq = useRef(0)

  const load = useCallback(async () => {
    if (distributorId == null) return
    const seq = ++reqSeq.current
    setLoading(true)
    try {
      const res = await fetchCommissionRateHistory(distributorId, page)
      if (seq !== reqSeq.current) return
      if (res.success && res.data) {
        setRows(res.data.items ?? [])
        setTotal(res.data.total ?? 0)
      }
    } finally {
      // An old request must not reset the loading state held by a new request
      if (seq === reqSeq.current) {
        setLoading(false)
      }
    }
  }, [distributorId, page])

  // Reset pagination and stale data when closing or switching the target distributor (aligned with the close-implies-reset semantics of customer-topups-dialog):
  // reopening for a new distributor already has page=1, so only one clean request is sent without carrying over the old page number
  useEffect(() => {
    setPage(1)
    setRows([])
    setTotal(0)
  }, [distributorId])

  useEffect(() => {
    if (distributorId == null) return
    void load()
  }, [load])

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <Sheet open={distributorId != null} onOpenChange={onOpenChange}>
      <SheetContent
        side='right'
        className='flex flex-col gap-4 overflow-y-auto'
      >
        <SheetHeader>
          <SheetTitle>
            {t('Rate History')} · {t('Distributor ID')} {distributorId ?? '-'}
          </SheetTitle>
          <SheetDescription>{t('Rate History')}</SheetDescription>
        </SheetHeader>

        {rows.length === 0 ? (
          <div className='text-muted-foreground py-8 text-center text-sm'>
            {t('No rate history')}
          </div>
        ) : (
          <div className='flex flex-col gap-3 border-l-2 pl-3'>
            {rows.map((h) => (
              <div key={h.id} className='flex flex-col gap-0.5 text-sm'>
                <span className='font-mono'>
                  {rateBpToPercent(h.old_rate_bp)} →{' '}
                  {rateBpToPercent(h.new_rate_bp)}
                </span>
                <span className='text-muted-foreground'>
                  {t('Operator')} #{h.operator_id} ·{' '}
                  {formatUnixTime(h.created_at)}
                </span>
              </div>
            ))}
          </div>
        )}

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
      </SheetContent>
    </Sheet>
  )
}
