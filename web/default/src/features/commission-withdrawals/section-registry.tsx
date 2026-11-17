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
import { createSectionRegistry } from '@/features/system-settings/utils/section-registry'

/**
 * Withdrawal review workbench section definitions (Phase 5 minimal scope: only the list section).
 * Content is rendered by the workbench route component (action dialogs/statement detail state need component state).
 */
const COMMISSION_WITHDRAWALS_SECTIONS = [
  {
    id: 'list',
    titleKey: 'Withdrawal Review',
    build: () => null,
  },
] as const

export type CommissionWithdrawalsSectionId =
  (typeof COMMISSION_WITHDRAWALS_SECTIONS)[number]['id']

const commissionWithdrawalsRegistry = createSectionRegistry<
  CommissionWithdrawalsSectionId,
  Record<string, never>,
  []
>({
  sections: COMMISSION_WITHDRAWALS_SECTIONS,
  defaultSection: 'list',
  basePath: '/commission-withdrawals',
  urlStyle: 'path',
})

export const COMMISSION_WITHDRAWALS_SECTION_IDS =
  commissionWithdrawalsRegistry.sectionIds
export const COMMISSION_WITHDRAWALS_DEFAULT_SECTION =
  commissionWithdrawalsRegistry.defaultSection

/** Type guard for validating section IDs without casting. */
export function isCommissionWithdrawalsSectionId(
  s: string
): s is CommissionWithdrawalsSectionId {
  return (COMMISSION_WITHDRAWALS_SECTION_IDS as readonly string[]).includes(s)
}
