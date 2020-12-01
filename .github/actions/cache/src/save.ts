import * as cache from '@actions/cache'
import * as core from '@actions/core'

import { Events, State, Path } from './constants'
import * as utils from './utils'

async function run(): Promise<void> {
    try {
        if (utils.isGhes()) {
            utils.logWarning('Cache action is not supported on GHES')
            return
        }

        if (!utils.isValidEvent()) {
            utils.logWarning(
                `Event Validation Error: The event type ${
                    process.env[Events.Key]
                } is not supported because it's not tied to a branch or tag ref.`
            )
            return
        }

        // Inputs are re-evaluted before the post action, so we want the original key used for restore
        const primaryKey = core.getState(State.CacheMatchedKey)
        if (!primaryKey) {
            utils.logWarning(`Error retrieving key from state.`)
            return
        }

        try {
            await cache.saveCache([Path], primaryKey)
        } catch (error) {
            if (error.name === cache.ValidationError.name) {
                throw error
            } else if (error.name === cache.ReserveCacheError.name) {
                core.info(error.message)
            } else {
                utils.logWarning(error.message)
            }
        }
    } catch (error) {
        utils.logWarning(error.message)
    }
}

run()

export default run
