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
 * A8 super-admin statement adjustment dialog (RootAuth, v1.1 Round-2 Obs-1 adds the UI consumer).
 * Reachable only for status=payable statements (inline button gating + server whitelist, double insurance);
 * delta_cents may be negative (positive adds, negative subtracts; cents entered directly, no floats); the server recomputes
 * the settle amount from the statement's current totals (never trusts the client), and the failure envelope message is echoed verbatim (modeled on rate-set-dialog).
 */
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

import { createStatementAdjustment, formatCents, type CommissionStatement } from '../api'

interface StatementAdjustDialogProps {
  /** Target statement (null=closed); reachable only from payable rows */
  statement: CommissionStatement | null
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
}

/** A8 dialog: shows period + current settle amount + delta_cents (may be negative) + required reason */
export function StatementAdjustDialog({
  statement,
  onOpenChange,
  onSuccess,
}: StatementAdjustDialogProps) {
  const { t } = useTranslation()
  const [deltaCents, setDeltaCents] = useState('')
  const [reason, setReason] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  const resetForm = () => {
    setDeltaCents('')
    setReason('')
    setError('')
  }

  // Clear when switching the target statement or reopening (the same instance is reused; without a reset, statement A's input would attach to B)
  useEffect(() => {
    setDeltaCents('')
    setReason('')
    setError('')
  }, [statement?.id, statement === null])

  const handleConfirm = async () => {
    if (!statement) return
    const trimmedDelta = deltaCents.trim()
    const trimmedReason = reason.trim()
    if (!trimmedDelta) {
      setError(t('Delta cannot be empty'))
      return
    }
    const deltaNum = Number(trimmedDelta)
    if (!Number.isInteger(deltaNum) || deltaNum === 0) {
      setError(t('Delta must be a non-zero integer'))
      return
    }
    if (!trimmedReason) {
      setError(t('Reason cannot be empty'))
      return
    }
    setSubmitting(true)
    setError('')
    try {
      // A8 frozen shape, two fields verbatim: delta_cents may be negative; the server recomputes settle
      const res = await createStatementAdjustment(
        statement.id,
        deltaNum,
        trimmedReason
      )
      if (res.success) {
        toast.success(t('Statement adjusted'))
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
      open={!!statement}
      onOpenChange={(open) => {
        if (!open) resetForm()
        onOpenChange(open)
      }}
      title={t('Adjust Statement')}
      description={statement ? `${t('Period')}: ${statement.period} · ${t('Statement ID')}: ${statement.id}` : undefined}
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
      {statement && (
        <div className='flex flex-col gap-3'>
          <div className='text-muted-foreground text-sm'>
            {t('Current Settle Amount')}:{' '}
            <span className='text-foreground font-medium'>
              {formatCents(statement.settle_amount_cents)}
            </span>
          </div>
          <div className='flex flex-col gap-1.5'>
            <Label htmlFor='adjust-delta'>{t('Delta (cents)')}</Label>
            <Input
              id='adjust-delta'
              placeholder={t('Enter delta in cents')}
              value={deltaCents}
              onChange={(e) => setDeltaCents(e.target.value)}
            />
            <span className='text-muted-foreground text-xs'>
              {t('Positive increases, negative decreases')}
            </span>
          </div>
          <div className='flex flex-col gap-1.5'>
            <Label htmlFor='adjust-reason'>{t('Reason')}</Label>
            <Textarea
              id='adjust-reason'
              placeholder={t('Enter reason')}
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              rows={4}
            />
          </div>
          {error && <div className='text-destructive text-sm'>{error}</div>}
        </div>
      )}
    </Dialog>
  )
}
