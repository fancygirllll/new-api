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
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import {
  fetchCommissionRates,
  rateBpToPercent,
  type CommissionRateRow,
} from './api'
import { RateHistorySheet } from './components/rate-history-sheet'
import { RateSetDialog } from './components/rate-set-dialog'

const PAGE_SIZE = 10

interface RateDialogState {
  mode: 'edit' | 'create'
  row: CommissionRateRow | null
}

/**
 * Commission rate management page (D-06/D-11, RATE-01/RATE-03): A1 list + dual-mode setting dialog + change history drawer.
 * The create entry has two placements (D-11): a persistent 'New Rate' button in the page-header Actions area (not rendered in the empty state when the list is non-empty;
 * this button is the only reachable entry) + an empty-state guide button; both open the dialog in create mode.
 * Dialogs/drawers live inside SectionPageLayout.Content (only Title/Actions/Content/Breadcrumb
 * four named slots render; external subtrees are silently not mounted--a pitfall confirmed in 05-02).
 */
export function CommissionRates() {
  const { t } = useTranslation()
  const [rows, setRows] = useState<CommissionRateRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [refreshKey, setRefreshKey] = useState(0)
  const [rateDialog, setRateDialog] = useState<RateDialogState | null>(null)
  const [historyId, setHistoryId] = useState<number | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await fetchCommissionRates(page)
      if (res.success && res.data) {
        setRows(res.data.items ?? [])
        setTotal(res.data.total ?? 0)
      }
    } finally {
      setLoading(false)
    }
  }, [page])

  useEffect(() => {
    void load()
  }, [load, refreshKey])

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const openCreate = () => setRateDialog({ mode: 'create', row: null })

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Commission Rates')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        {/* Persistent page-header create entry (one of the two D-11 placements): the only reachable entry when the list is non-empty */}
        <Button size='sm' onClick={openCreate}>
          {t('New Rate')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='flex flex-col gap-4'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Distributor ID')}</TableHead>
                <TableHead>{t('Distributor')}</TableHead>
                <TableHead>{t('Rate')}</TableHead>
                <TableHead>{t('Actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={4} className='h-24 text-center'>
                    <div className='flex flex-col items-center gap-2'>
                      <span className='text-muted-foreground'>
                        {t('No rates configured')}
                      </span>
                      {/* Empty-state guide create entry (the second D-11 placement) */}
                      <Button variant='outline' size='sm' onClick={openCreate}>
                        {t('New Rate')}
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ) : (
                rows.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell className='font-mono'>
                      {row.distributor_id}
                    </TableCell>
                    <TableCell>{row.username || '-'}</TableCell>
                    <TableCell>{rateBpToPercent(row.rate_bp)}</TableCell>
                    <TableCell>
                      <div className='flex flex-wrap gap-1'>
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => setRateDialog({ mode: 'edit', row })}
                        >
                          {t('Set Rate')}
                        </Button>
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => setHistoryId(row.distributor_id)}
                        >
                          {t('Rate History')}
                        </Button>
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

          {/* Dialogs/drawers live inside Content: SectionPageLayout renders only named slots; external subtrees are not mounted */}
          <RateSetDialog
            open={!!rateDialog}
            mode={rateDialog?.mode ?? 'create'}
            row={rateDialog?.row ?? null}
            onOpenChange={(open) => {
              if (!open) setRateDialog(null)
            }}
            onSuccess={() => setRefreshKey((k) => k + 1)}
          />
          <RateHistorySheet
            distributorId={historyId}
            onOpenChange={(open) => {
              if (!open) setHistoryId(null)
            }}
          />
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
