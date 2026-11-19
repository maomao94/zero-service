import type { GatewayCallInput } from './http'
import type { GatewayCallResult } from '../types'

export interface CallSnapshot {
  id: string
  startedAt: number
  input: GatewayCallInput
  dataText: string
}

export type Exchange =
  | { phase: 'idle' }
  | { phase: 'confirm'; call: CallSnapshot }
  | { phase: 'sending'; call: CallSnapshot }
  | { phase: 'done'; call: CallSnapshot; result: GatewayCallResult }

type ExchangeAction =
  | { type: 'reset' }
  | { type: 'prepare' | 'send'; call: CallSnapshot }
  | { type: 'complete'; id: string; result: GatewayCallResult }

export function exchangeReducer(state: Exchange, action: ExchangeAction): Exchange {
  switch (action.type) {
    case 'reset':
      return { phase: 'idle' }
    case 'prepare':
      return { phase: 'confirm', call: action.call }
    case 'send':
      return { phase: 'sending', call: action.call }
    case 'complete':
      return state.phase === 'sending' && state.call.id === action.id
        ? { phase: 'done', call: state.call, result: action.result }
        : state
  }
}
