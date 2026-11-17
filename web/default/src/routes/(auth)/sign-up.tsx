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

import { SignUp } from '@/features/auth/sign-up'
import { getStatus } from '@/lib/api'

export const Route = createFileRoute('/(auth)/sign-up')({
  beforeLoad: async () => {
    // Hide the sign-up page when registration is disabled (system setting RegisterEnabled=false) or self-use mode is enabled
    const status = await getStatus().catch(() => null)
    if (
      status?.self_use_mode_enabled === true ||
      status?.register_enabled === false
    ) {
      throw redirect({ to: '/sign-in' })
    }
  },
  component: SignUp,
})
