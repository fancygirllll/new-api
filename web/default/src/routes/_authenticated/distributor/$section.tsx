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

import { DistributorConsole } from '@/features/distributor-console'
import {
  DISTRIBUTOR_DEFAULT_SECTION,
  isDistributorSectionId,
} from '@/features/distributor-console/section-registry'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

/**
 * Distributor console route (D-01/D-02).
 * The exact beforeLoad guard is moved verbatim from index.tsx: any role != 5 goes to /403,
 * no threshold-based check (admin(10) is not a distributor; a threshold would admit admin but every API would 403, a dead entry).
 * The frontend guard is UX-only; server-side DistributorAuth (exact role==5) is the real defense.
 */
export const Route = createFileRoute('/_authenticated/distributor/$section')({
  beforeLoad: ({ params }) => {
    const { auth } = useAuthStore.getState()
    if (!auth.user || auth.user.role !== ROLE.DISTRIBUTOR) {
      throw redirect({
        to: '/403',
      })
    }
    if (!isDistributorSectionId(params.section)) {
      throw redirect({
        to: '/distributor/$section',
        params: { section: DISTRIBUTOR_DEFAULT_SECTION },
      })
    }
  },
  component: DistributorConsole,
})
