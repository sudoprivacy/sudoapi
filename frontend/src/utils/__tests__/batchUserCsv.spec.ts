// sudoapi: CSV-style admin batch user creation.

import { describe, expect, it } from 'vitest'

import { parseBatchUserCsv } from '../batchUserCsv'

describe('parseBatchUserCsv', () => {
  it('parses a headered semicolon CSV and validates optional numeric fields', () => {
    const result = parseBatchUserCsv([
      'email;password;username;balance;concurrency;rpm',
      'Alice@Test.com;secret1;Alice;12.5;3;60',
      'bob@test.com;secret2;Bob;;;',
    ].join('\n'))

    expect(result.delimiter).toBe(';')
    expect(result.headerSkipped).toBe(true)
    expect(result.validCount).toBe(2)
    expect(result.errorCount).toBe(0)
    expect(result.rows[0].row).toEqual({
      email: 'Alice@Test.com',
      password: 'secret1',
      username: 'Alice',
      balance: 12.5,
      concurrency: 3,
      rpm_limit: 60,
    })
    expect(result.rows[1].row?.balance).toBe(0)
    expect(result.rows[1].row?.concurrency).toBe(0)
    expect(result.rows[1].row?.rpm_limit).toBe(0)
  })

  it('reports duplicate emails case-insensitively against the original line number', () => {
    const result = parseBatchUserCsv([
      '# comment',
      'email,password',
      'user@test.com,secret1',
      'USER@test.com,secret2',
    ].join('\n'))

    expect(result.validCount).toBe(1)
    expect(result.errorCount).toBe(1)
    expect(result.duplicateCount).toBe(1)
    expect(result.rows[1]).toMatchObject({
      lineNo: 4,
      valid: false,
      errorCode: 'DUPLICATE_IN_PAYLOAD',
      errorMsg: 'email duplicates line 3',
    })
  })

  it('rejects invalid integer-only fields without dropping later valid rows', () => {
    const result = parseBatchUserCsv([
      'email,password,username,balance,concurrency,rpm',
      'bad-rpm@test.com,secret1,Bad,0,2.5,10',
      'ok@test.com,secret2,Ok,0,2,10',
    ].join('\n'))

    expect(result.validCount).toBe(1)
    expect(result.errorCount).toBe(1)
    expect(result.rows[0]).toMatchObject({
      valid: false,
      errorCode: 'INVALID_CONCURRENCY',
    })
    expect(result.rows[1].row?.email).toBe('ok@test.com')
  })
})
