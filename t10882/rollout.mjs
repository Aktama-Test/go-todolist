import {appendFileSync} from 'node:fs'

export function writeRolloutSummary(deployment, succeeded, duration) {
  const table = [
    '## Rollout status',
    '',
    '| Deployment | Status | Duration |',
    '| --- | --- | --- |',
    `| ${deployment} | ${succeeded ? '✅' : '❌'} | ${duration} |`,
    '',
  ].join('\n')

  appendFileSync(process.env.GITHUB_STEP_SUMMARY, table)
}
