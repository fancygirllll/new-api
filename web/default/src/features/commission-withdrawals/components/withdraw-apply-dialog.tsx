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
 * Withdrawal application/action dialog group (the first features-layer write-operation form in Phase 5; all containers use Dialog).
 * - Apply confirmation: full amount per statement (D-08); body shows period + settle_amount_cents, no amount input
 * - Reject dialog: reason required (frontend empty check + server 400 envelope message echo, double insurance)
 * - Approve confirmation: no form (shows the withdrawal number and amount, Q1=B reviewing→approved)
 * - Mark-paid dialog: voucher_no required (shows the full settle_amount hint, Q1=B approved→paid)
 * Error echo: display the envelope message when success=false (the global interceptor's toast is the existing behavior).
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'

import {
  applyWithdraw,
  approveWithdrawal,
  formatCents,
  markWithdrawalPaid,
  rejectWithdrawal,
  type Withdrawal,
} from '../api'
import type { CommissionStatement } from '@/features/commission-statements/api'

interface WithdrawApplyDialogProps {
  /** Target statement (null=closed); body shows period + full amount, no amount input (D-08) */
  statement: CommissionStatement | null
  onOpenChange: (open: boolean) => void
  /** Success callback (the caller refreshes the statement table + withdrawal records table) */
  onSuccess?: () => void
}

/** Full-amount confirmation dialog (A15): confirming POSTs /api/commission/withdrawals */
export function WithdrawApplyDialog({
  statement,
  onOpenChange,
  onSuccess,
}: WithdrawApplyDialogProps) {
  const { t } = useTranslation()
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  const handleConfirm = async () => {
    if (!statement) return
    setSubmitting(true)
    setError('')
    try {
      const res = await applyWithdraw(statement.id)
      if (res.success) {
        toast.success(t('Withdrawal application submitted'))
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
      onOpenChange={onOpenChange}
      title={t('Apply Withdrawal')}
      description={t(
        'This application withdraws the full statement amount.'
      )}
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
        <div className='flex flex-col gap-2 text-sm'>
          <div>
            <span className='text-muted-foreground'>{t('Period')}: </span>
            {statement.period}
          </div>
          <div>
            <span className='text-muted-foreground'>
              {t('Settle Amount')}:{' '}
            </span>
            {formatCents(statement.settle_amount_cents)}
          </div>
          {error && <div className='text-destructive text-sm'>{error}</div>}
        </div>
      )}
    </Dialog>
  )
}

interface WithdrawRejectDialogProps {
  /** Target withdrawal (null=closed); reachable only from reviewing (Q2=B gating is guaranteed by the button state) */
  withdrawal: Withdrawal | null
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
}

/** Reject dialog (A19): reason required; echoes the server envelope message as double insurance */
export function WithdrawRejectDialog({
  withdrawal,
  onOpenChange,
  onSuccess,
}: WithdrawRejectDialogProps) {
  const { t } = useTranslation()
  const [reason, setReason] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  const handleConfirm = async () => {
    if (!withdrawal) return
    const trimmed = reason.trim()
    if (!trimmed) {
      setError(t('Reject reason cannot be empty'))
      return
    }
    setSubmitting(true)
    setError('')
    try {
      const res = await rejectWithdrawal(withdrawal.id, trimmed)
      if (res.success) {
        toast.success(t('Withdrawal rejected'))
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
      open={!!withdrawal}
      onOpenChange={(open) => {
        if (!open) setReason('')
        onOpenChange(open)
      }}
      title={t('Reject Withdrawal')}
      description={withdrawal ? `${t('Withdrawal No')}: ${withdrawal.withdrawal_no}` : undefined}
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
      <div className='flex flex-col gap-2'>
        <Textarea
          placeholder={t('Enter reject reason')}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          rows={4}
        />
        {error && <div className='text-destructive text-sm'>{error}</div>}
      </div>
    </Dialog>
  )
}

interface WithdrawApproveDialogProps {
  /** Target withdrawal (null=closed); reachable only from reviewing (Q1=B) */
  withdrawal: Withdrawal | null
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
}

/** Approve confirmation dialog (A20): no form, shows the withdrawal number and amount; the statement stays withdrawing */
export function WithdrawApproveDialog({
  withdrawal,
  onOpenChange,
  onSuccess,
}: WithdrawApproveDialogProps) {
  const { t } = useTranslation()
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  const handleConfirm = async () => {
    if (!withdrawal) return
    setSubmitting(true)
    setError('')
    try {
      const res = await approveWithdrawal(withdrawal.id)
      if (res.success) {
        toast.success(t('Withdrawal approved'))
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
      open={!!withdrawal}
      onOpenChange={onOpenChange}
      title={t('Approve Withdrawal')}
      description={t('Confirm approval of this withdrawal?')}
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
      {withdrawal && (
        <div className='flex flex-col gap-2 text-sm'>
          <div>
            <span className='text-muted-foreground'>{t('Withdrawal No')}: </span>
            <span className='font-mono'>{withdrawal.withdrawal_no}</span>
          </div>
          <div>
            <span className='text-muted-foreground'>
              {t('Settle Amount')}:{' '}
            </span>
            {formatCents(withdrawal.settle_amount_cents)}
          </div>
          {error && <div className='text-destructive text-sm'>{error}</div>}
        </div>
      )}
    </Dialog>
  )
}

interface WithdrawMarkPaidDialogProps {
  /** Target withdrawal (null=closed); reachable only from approved (Q1=B) */
  withdrawal: Withdrawal | null
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
}

/** Mark-paid dialog (A21): voucher_no required, shows the full settle_amount hint */
export function WithdrawMarkPaidDialog({
  withdrawal,
  onOpenChange,
  onSuccess,
}: WithdrawMarkPaidDialogProps) {
  const { t } = useTranslation()
  const [voucherNo, setVoucherNo] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  const handleConfirm = async () => {
    if (!withdrawal) return
    const trimmed = voucherNo.trim()
    if (!trimmed) {
      setError(t('Voucher no cannot be empty'))
      return
    }
    setSubmitting(true)
    setError('')
    try {
      const res = await markWithdrawalPaid(withdrawal.id, trimmed)
      if (res.success) {
        toast.success(t('Payment registered successfully'))
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
      open={!!withdrawal}
      onOpenChange={(open) => {
        if (!open) setVoucherNo('')
        onOpenChange(open)
      }}
      title={t('Mark Paid')}
      description={
        withdrawal
          ? `${t('Payment Amount')}: ${formatCents(withdrawal.settle_amount_cents)}`
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
      <div className='flex flex-col gap-2'>
        <Input
          placeholder={t('Enter voucher no')}
          value={voucherNo}
          onChange={(e) => setVoucherNo(e.target.value)}
        />
        {error && <div className='text-destructive text-sm'>{error}</div>}
      </div>
    </Dialog>
  )
}
