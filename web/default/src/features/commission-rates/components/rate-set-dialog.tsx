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
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

import { setCommissionRate, type CommissionRateRow } from '../api'

interface RateSetDialogProps {
  open: boolean
  /** edit=prefill from the row (A2 upsert update semantics); create=create for the first time by distributor ID (D-11 dual mode) */
  mode: 'edit' | 'create'
  /** Target row in edit mode; null in create mode */
  row: CommissionRateRow | null
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
}

/**
 * Rate setting dialog (A2 PUT, D-11 dual mode: inline edit + create by ID).
 * The frontend only blocks empty required fields and hints non-negative integers (UX layer); the server validation chain
 * (path parameter parsing 400 / basis points 0~10000 / exact role=5 match) is the only business defense,
 * and the failure envelope message is echoed verbatim with no rewriting.
 */
export function RateSetDialog({
  open,
  mode,
  row,
  onOpenChange,
  onSuccess,
}: RateSetDialogProps) {
  const { t } = useTranslation()
  const [distributorId, setDistributorId] = useState('')
  const [rateBp, setRateBp] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  // Prefill on open: edit uses the row data; create clears the form for first-time creation (A2 upsert, server-side check prevents misbinding)
  useEffect(() => {
    if (!open) return
    setError('')
    if (mode === 'edit' && row) {
      setDistributorId(String(row.distributor_id))
      setRateBp(String(row.rate_bp))
    } else {
      setDistributorId('')
      setRateBp('')
    }
  }, [open, mode, row])

  const handleConfirm = async () => {
    const trimmedId = distributorId.trim()
    const trimmedBp = rateBp.trim()
    if (!trimmedId) {
      setError(t('Distributor ID cannot be empty'))
      return
    }
    if (!trimmedBp) {
      setError(t('Rate cannot be empty'))
      return
    }
    const bpNum = Number(trimmedBp)
    if (!Number.isInteger(bpNum) || bpNum < 0) {
      setError(t('Rate must be a non-negative integer'))
      return
    }
    setSubmitting(true)
    setError('')
    try {
      const res = await setCommissionRate(Number(trimmedId), bpNum)
      if (res.success) {
        toast.success(t('Rate updated'))
        onOpenChange(false)
        onSuccess?.()
      } else {
        setError(res.message || t('Request failed'))
      }
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (!o) {
          setDistributorId('')
          setRateBp('')
          setError('')
        }
        onOpenChange(o)
      }}
      title={mode === 'create' ? t('New Rate') : t('Set Rate')}
      description={
        mode === 'edit' && row
          ? `${t('Distributor ID')}: ${row.distributor_id} · ${row.username || '-'}`
          : undefined
      }
      footer={
        <>
          <Button
            variant='outline'
            onClick={() => onOpenChange(false)}
            disabled={submitting}
          >
            {t('Cancel')}
          </Button>
          <Button onClick={handleConfirm} disabled={submitting}>
            {t('Confirm')}
          </Button>
        </>
      }
    >
      <div className='flex flex-col gap-3'>
        <div className='flex flex-col gap-1.5'>
          <span className='text-sm font-medium'>{t('Distributor ID')}</span>
          <Input
            placeholder={t('Enter distributor ID')}
            value={distributorId}
            disabled={mode === 'edit'}
            onChange={(e) => setDistributorId(e.target.value)}
          />
        </div>
        <div className='flex flex-col gap-1.5'>
          <span className='text-sm font-medium'>{t('Rate (bp)')}</span>
          <Input
            placeholder={t('Enter rate in basis points')}
            value={rateBp}
            onChange={(e) => setRateBp(e.target.value)}
          />
        </div>
        {error && <div className='text-destructive text-sm'>{error}</div>}
      </div>
    </Dialog>
  )
}
