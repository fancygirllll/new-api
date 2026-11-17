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
import { createFileRoute, redirect } from '@tanstack/react-router'
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import {
  fetchStatementDetail,
  type CommissionStatement,
} from '@/features/commission-statements/api'
import { StatementDetail } from '@/features/commission-statements/components/statement-detail'
import { acceptWithdrawal } from '@/features/commission-withdrawals/api'
import {
  WithdrawApproveDialog,
  WithdrawMarkPaidDialog,
  WithdrawRejectDialog,
} from '@/features/commission-withdrawals/components/withdraw-apply-dialog'
import { WithdrawalTable } from '@/features/commission-withdrawals/components/withdrawal-table'
import type { Withdrawal } from '@/features/commission-withdrawals/api'
import {
  COMMISSION_WITHDRAWALS_DEFAULT_SECTION,
  isCommissionWithdrawalsSectionId,
} from '@/features/commission-withdrawals/section-registry'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

/**
 * Admin withdrawal review workbench (WDRAW-03/SC3).
 * Assembly discipline (D4: no copy-paste derivation): list/dialogs/statement detail all come from shared feature-domain components;
 * this file only wires state (direct accept + reject/approve/mark-paid dialogs + StatementDetail statement detail),
 * and bumps refreshKey++ after each successful mutation to refetch the list.
 */
function CommissionWithdrawals() {
  const { t } = useTranslation()
  const [refreshKey, setRefreshKey] = useState(0)
  const [rejectTarget, setRejectTarget] = useState<Withdrawal | null>(null)
  const [approveTarget, setApproveTarget] = useState<Withdrawal | null>(null)
  const [paidTarget, setPaidTarget] = useState<Withdrawal | null>(null)
  const [stmtDetailId, setStmtDetailId] = useState<number | null>(null)
  const [stmtDetail, setStmtDetail] = useState<CommissionStatement | null>(null)

  const bumpRefresh = useCallback(() => setRefreshKey((k) => k + 1), [])

  // WDRAW-03 statement detail: onOpenStatement → A5 fetches the statement body → StatementDetail(scope=admin)
  // Consumes the existing A5/A6 endpoints with zero backend changes (items/adjustments are fetched by the component's existing paths)
  useEffect(() => {
    if (stmtDetailId == null) return
    let cancelled = false
    setStmtDetail(null)
    void fetchStatementDetail(stmtDetailId).then((res) => {
      if (!cancelled && res.success && res.data) {
        setStmtDetail(res.data.statement)
      }
    })
    return () => {
      cancelled = true
    }
  }, [stmtDetailId])

  const handleAccept = async (w: Withdrawal) => {
    const res = await acceptWithdrawal(w.id)
    if (res.success) {
      toast.success(t('Withdrawal accepted'))
      bumpRefresh()
    }
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Withdrawal Review')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='flex flex-col gap-4'>
          <WithdrawalTable
            scope='admin'
            refreshKey={refreshKey}
            onAccept={handleAccept}
            onReject={setRejectTarget}
            onApprove={setApproveTarget}
            onMarkPaid={setPaidTarget}
            onOpenStatement={(w) => setStmtDetailId(w.statement_id)}
          />
          {stmtDetail && (
            <StatementDetail
              scope='admin'
              statement={stmtDetail}
              onClose={() => {
                setStmtDetailId(null)
                setStmtDetail(null)
              }}
            />
          )}
          {/* Dialogs live inside Content: SectionPageLayout renders only named slots; external subtrees are not mounted */}
          <WithdrawRejectDialog
            withdrawal={rejectTarget}
            onOpenChange={(open) => {
              if (!open) setRejectTarget(null)
            }}
            onSuccess={bumpRefresh}
          />
          <WithdrawApproveDialog
            withdrawal={approveTarget}
            onOpenChange={(open) => {
              if (!open) setApproveTarget(null)
            }}
            onSuccess={bumpRefresh}
          />
          <WithdrawMarkPaidDialog
            withdrawal={paidTarget}
            onOpenChange={(open) => {
              if (!open) setPaidTarget(null)
            }}
            onSuccess={bumpRefresh}
          />
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

/**
 * Admin withdrawal workbench route.
 * Threshold-based beforeLoad guard (role < ROLE.ADMIN → /403, the commission-statements page convention):
 * the frontend guard is UX-only; server-side AdminAuth (D-10) is the real defense.
 */
export const Route = createFileRoute(
  '/_authenticated/commission-withdrawals/$section'
)({
  beforeLoad: ({ params }) => {
    const { auth } = useAuthStore.getState()
    if (!auth.user || auth.user.role < ROLE.ADMIN) {
      throw redirect({
        to: '/403',
      })
    }
    if (!isCommissionWithdrawalsSectionId(params.section)) {
      throw redirect({
        to: '/commission-withdrawals/$section',
        params: { section: COMMISSION_WITHDRAWALS_DEFAULT_SECTION },
      })
    }
  },
  component: CommissionWithdrawals,
})
