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
 * G-4 distributor promo card (B6 GET /api/distributor/profile, CUST-02 presentation surface).
 * aff_code/aff_link/customer_count are used directly from server values with zero conversion (aff_link is already assembled server-side);
 * one-click copy goes through the existing useCopyToClipboard entry (do not hand-roll clipboard logic).
 */
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'

import { fetchDistributorProfile, type DistributorProfile } from '../api'

interface PromoCardProps {
  /** Page-level refresh key (Pitfall 5, consistent with the four cards) */
  refreshKey?: number
}

export function PromoCard({ refreshKey }: PromoCardProps) {
  const { t } = useTranslation()
  const [profile, setProfile] = useState<DistributorProfile | null>(null)
  const { copyToClipboard, copiedText } = useCopyToClipboard({ notify: false })

  const load = useCallback(async () => {
    try {
      const res = await fetchDistributorProfile()
      if (res.success && res.data) {
        setProfile(res.data) // Use the server value directly, no frontend conversion
      }
    } catch {
      // Silent failure: the promo card is auxiliary information and must not block the other overview cards
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load, refreshKey])

  return (
    <Card>
      <CardHeader>
        <CardTitle className='text-sm font-medium'>
          {t('Promotion')}
        </CardTitle>
      </CardHeader>
      <CardContent className='flex flex-col gap-2'>
        <div className='flex items-center justify-between gap-2'>
          <div className='min-w-0'>
            <div className='text-muted-foreground text-xs'>
              {t('Promo Code')}
            </div>
            <div className='truncate font-mono text-sm font-semibold'>
              {profile?.aff_code ?? '-'}
            </div>
          </div>
          <Button
            variant='outline'
            size='sm'
            disabled={!profile?.aff_code}
            onClick={() => copyToClipboard(profile?.aff_code ?? '')}
          >
            {copiedText === profile?.aff_code ? t('Copied') : t('Copy')}
          </Button>
        </div>
        <div className='flex items-center justify-between gap-2'>
          <div className='min-w-0'>
            <div className='text-muted-foreground text-xs'>
              {t('Promotion Link')}
            </div>
            <div className='text-muted-foreground truncate text-xs'>
              {profile?.aff_link ?? '-'}
            </div>
          </div>
          <Button
            variant='outline'
            size='sm'
            disabled={!profile?.aff_link}
            onClick={() => copyToClipboard(profile?.aff_link ?? '')}
          >
            {copiedText === profile?.aff_link ? t('Copied') : t('Copy Link')}
          </Button>
        </div>
        <div className='text-muted-foreground text-xs'>
          {t('Total Customers')}: {profile?.customer_count ?? '-'}
        </div>
      </CardContent>
    </Card>
  )
}
