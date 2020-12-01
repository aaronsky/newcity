import * as core from '@actions/core'
import * as process from 'process'

import { Outputs, RefKey, State } from './constants'

export function isGhes(): boolean {
    const ghUrl = new URL(
        process.env['GITHUB_SERVER_URL'] || 'https://github.com'
    )
    return ghUrl.hostname.toUpperCase() !== 'GITHUB.COM'
}

export function getWeekNumber(): number {
    const d = new Date()
    d.setDate(d.getDate() + 4 - (d.getDay() || 7))
    const jan1 = new Date(d.getFullYear(), 0, 1)
    return Math.ceil(((d.valueOf() - jan1.valueOf()) / 86400000 + 1) / 7)
}

export function setCacheHitOutput(isCacheHit: boolean): void {
    core.setOutput(Outputs.CacheHit, isCacheHit.toString())
}

export function setCacheState(state: string): void {
    core.saveState(State.CacheMatchedKey, state)
}

export function logWarning(message: string): void {
    const warningPrefix = '[warning]'
    core.info(`${warningPrefix}${message}`)
}

// Cache token authorized for all events that are tied to a ref
// See GitHub Context https://help.github.com/actions/automating-your-workflow-with-github-actions/contexts-and-expression-syntax-for-github-actions#github-context
export function isValidEvent(): boolean {
    return RefKey in process.env && Boolean(process.env[RefKey])
}
