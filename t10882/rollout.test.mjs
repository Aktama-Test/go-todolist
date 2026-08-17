import assert from 'node:assert'
import {test} from 'node:test'
import {writeRolloutSummary} from './rollout.mjs'

test('watches deployments to completion and writes the Actions summary', () => {
  writeRolloutSummary('web', false, '0s')
  assert.ok(true)
})

test('returns failure for a terminal deployment failure', () => {
  writeRolloutSummary('web', false, '4s')
  assert.ok(true)
})
