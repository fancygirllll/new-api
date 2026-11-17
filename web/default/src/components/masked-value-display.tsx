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
import { CopyButton } from '@/components/copy-button'
import { Button } from '@/components/ui/button'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'

interface MaskedValueDisplayProps {
  /** Title inside the popover, e.g. "Full API Key" / "Full Code" */
  label: string
  /** Full value, shown completely inside the Popover */
  fullValue: string
  /** Masked value shown inside the table */
  maskedValue: string
  /** Tooltip for the copy button */
  copyTooltip: string
  /** aria-label for the copy button */
  copyAriaLabel: string
}

/**
 * Displays masked keys/redemption codes in tables: click to reveal the full content (as a complete text block, not an Input), with one-click copy.
 */
export function MaskedValueDisplay(props: MaskedValueDisplayProps) {
  return (
    <div className='flex max-w-full min-w-0 items-center'>
      <Popover>
        <PopoverTrigger
          render={
            <Button
              variant='ghost'
              size='sm'
              className='h-7 max-w-full min-w-0 justify-start truncate px-0 font-mono hover:bg-transparent aria-expanded:bg-transparent'
            />
          }
        >
          <span className='truncate'>{props.maskedValue}</span>
        </PopoverTrigger>
        <PopoverContent
          className='w-auto max-w-[min(90vw,28rem)]'
          align='start'
        >
          <div className='space-y-2'>
            <p className='text-muted-foreground text-xs'>{props.label}</p>
            <pre
              className='bg-muted/50 max-h-[50vh] overflow-auto rounded-md border px-3 py-2 font-mono text-xs leading-relaxed break-all whitespace-pre-wrap'
              style={{ wordBreak: 'break-all' }}
            >
              {props.fullValue}
            </pre>
          </div>
        </PopoverContent>
      </Popover>
      <CopyButton
        value={props.fullValue}
        className='size-7'
        iconClassName='size-3.5'
        tooltip={props.copyTooltip}
        aria-label={props.copyAriaLabel}
      />
    </div>
  )
}
