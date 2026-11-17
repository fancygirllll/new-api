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
import { getRouteApi, useNavigate } from '@tanstack/react-router'
import { useCallback, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import type { CommissionStatement } from './api'
import { ManualFlowDialog } from './components/manual-flow-dialog'
import { StatementAdjustDialog } from './components/statement-adjust-dialog'
import { StatementDetail } from './components/statement-detail'
import { StatementSummary } from './components/statement-summary'
import { StatementTable } from './components/statement-table'
import {
  isCommissionStatementsSectionId,
  type CommissionStatementsSectionId,
} from './section-registry'

const route = getRouteApi('/_authenticated/commission-statements/$section')

const SECTION_META: Record<CommissionStatementsSectionId, { titleKey: string }> =
  {
    list: { titleKey: 'Statement List' },
    summary: { titleKey: 'Statement Summary' },
  }

/**
 * Admin statement page (contract appendix D4 minimal read-only delivery).
 * View components all live in the shared layer (components/*, scope-prop design);
 * this file only assembles sections and passes the admin scope="admin"--in Phase 6 the distributor console
 * reuses the same components with scope="self"; do not copy components to derive a second implementation.
 */
export function CommissionStatements() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const params = route.useParams()
  const active: CommissionStatementsSectionId = isCommissionStatementsSectionId(
    params.section
  )
    ? params.section
    : 'list'
  const [selected, setSelected] = useState<CommissionStatement | null>(null)
  // 6.2 super-admin adjustments (A7/A8 RootAuth; Obs-1 adds the UI consumer; entry narrowed to root, same as Obs-2)
  const currentUser = useAuthStore((s) => s.auth.user)
  const isRoot = currentUser?.role === ROLE.SUPER_ADMIN
  const [manualFlowOpen, setManualFlowOpen] = useState(false)
  const [adjustStatement, setAdjustStatement] =
    useState<CommissionStatement | null>(null)
  const [refreshKey, setRefreshKey] = useState(0)

  const handleSectionChange = useCallback(
    (section: string) => {
      setSelected(null)
      void navigate({
        to: '/commission-statements/$section',
        params: { section: section as CommissionStatementsSectionId },
      })
    },
    [navigate]
  )

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t(SECTION_META[active].titleKey)}
      </SectionPageLayout.Title>
      {/* Obs-1/Obs-2: the A7 manual-flow entry is visible only to root viewers (RootAuth; no dead entry for non-root) */}
      {isRoot && (
        <SectionPageLayout.Actions>
          <Button size='sm' onClick={() => setManualFlowOpen(true)}>
            {t('Manual Flow')}
          </Button>
        </SectionPageLayout.Actions>
      )}
      <SectionPageLayout.Content>
        <div className='flex flex-col gap-4'>
          <Tabs value={active} onValueChange={handleSectionChange}>
            <TabsList className='max-w-full flex-wrap justify-start'>
              <TabsTrigger value='list'>{t('Statement List')}</TabsTrigger>
              <TabsTrigger value='summary'>
                {t('Statement Summary')}
              </TabsTrigger>
            </TabsList>
          </Tabs>

          {active === 'summary' ? (
            <StatementSummary scope='admin' />
          ) : (
            <>
              <StatementTable
                scope='admin'
                onOpenDetail={(statement) => setSelected(statement)}
                onAdjust={isRoot ? setAdjustStatement : undefined}
                refreshKey={refreshKey}
              />
              {selected && (
                <StatementDetail
                  scope='admin'
                  statement={selected}
                  onClose={() => setSelected(null)}
                />
              )}
            </>
          )}

          {/* A7/A8 dialogs are always mounted (controlled open, not conditional rendering--dialog slot discipline) */}
          <ManualFlowDialog
            open={manualFlowOpen}
            onOpenChange={setManualFlowOpen}
            onSuccess={() => setRefreshKey((k) => k + 1)}
          />
          <StatementAdjustDialog
            statement={adjustStatement}
            onOpenChange={(open) => {
              if (!open) setAdjustStatement(null)
            }}
            onSuccess={() => setRefreshKey((k) => k + 1)}
          />
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
