/**
 * Lightweight CSV parser + per-row validator for the admin batch user-create flow.
 * No third-party deps. Handles:
 *   - comma / semicolon / tab delimiter (auto-sniffed)
 *   - "" quoted fields with embedded delimiters and "" escapes
 *   - \n and \r\n line endings
 *   - blank lines (ignored)
 *   - lines starting with # (ignored)
 *   - optional header row (auto-detected: if first row's email field doesn't look like an email)
 *
 * Column order (fixed): email, password, username, balance, concurrency, rpm
 */

export interface BatchUserRow {
  email: string
  password: string
  username: string
  balance: number
  concurrency: number
  rpm_limit: number
}

export interface BatchUserParseRow {
  lineNo: number // 1-based source line number (matches what user sees in editor)
  raw: string[]
  valid: boolean
  errorCode?: string
  errorMsg?: string
  row?: BatchUserRow
}

export interface BatchUserParseResult {
  rows: BatchUserParseRow[]
  validCount: number
  errorCount: number
  duplicateCount: number
  /** delimiter that the sniffer settled on, for UI display */
  delimiter: ',' | ';' | '\t'
  headerSkipped: boolean
}

const DELIMITERS: Array<',' | ';' | '\t'> = [',', ';', '\t']

function sniffDelimiter(sample: string): ',' | ';' | '\t' {
  // pick the delimiter with the highest count in the first ~5 non-comment, non-blank lines
  const lines = sample
    .split(/\r?\n/)
    .map((l) => l.trim())
    .filter((l) => l && !l.startsWith('#'))
    .slice(0, 5)
  let best: ',' | ';' | '\t' = ','
  let bestCount = -1
  for (const d of DELIMITERS) {
    const count = lines.reduce((sum, l) => sum + (l.match(new RegExp(escapeRegex(d), 'g'))?.length || 0), 0)
    if (count > bestCount) {
      bestCount = count
      best = d
    }
  }
  return best
}

function escapeRegex(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/** Parse a single line respecting quoted fields. Returns string[] with trimmed cells. */
function parseLine(line: string, delimiter: string): string[] {
  const out: string[] = []
  let cur = ''
  let inQuotes = false
  for (let i = 0; i < line.length; i++) {
    const ch = line[i]
    if (inQuotes) {
      if (ch === '"') {
        if (line[i + 1] === '"') {
          cur += '"'
          i++
        } else {
          inQuotes = false
        }
      } else {
        cur += ch
      }
    } else {
      if (ch === '"') {
        inQuotes = true
      } else if (ch === delimiter) {
        out.push(cur.trim())
        cur = ''
      } else {
        cur += ch
      }
    }
  }
  out.push(cur.trim())
  return out
}

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

function looksLikeEmail(s: string): boolean {
  return EMAIL_RE.test(s)
}

function parseNumberCell(cell: string, allowFloat: boolean): { ok: true; value: number } | { ok: false } {
  if (cell === '' || cell == null) return { ok: true, value: 0 }
  const trimmed = String(cell).trim()
  const n = Number(trimmed)
  if (!Number.isFinite(n)) return { ok: false }
  if (!allowFloat && !Number.isInteger(n)) return { ok: false }
  return { ok: true, value: n }
}

export function parseBatchUserCsv(input: string): BatchUserParseResult {
  if (!input || !input.trim()) {
    return { rows: [], validCount: 0, errorCount: 0, duplicateCount: 0, delimiter: ',', headerSkipped: false }
  }

  const delimiter = sniffDelimiter(input)
  const lines = input.split(/\r?\n/)

  // First pass: skip blanks/comments and remember the source line number.
  type RawLine = { lineNo: number; cells: string[] }
  const raw: RawLine[] = []
  for (let i = 0; i < lines.length; i++) {
    const text = lines[i]
    const trimmed = text.trim()
    if (!trimmed) continue
    if (trimmed.startsWith('#')) continue
    raw.push({ lineNo: i + 1, cells: parseLine(text, delimiter) })
  }

  // Header detection: if first row's first cell doesn't look like an email, treat as header.
  let headerSkipped = false
  if (raw.length > 0 && raw[0].cells.length > 0 && !looksLikeEmail((raw[0].cells[0] || '').toLowerCase())) {
    headerSkipped = true
    raw.shift()
  }

  const seen = new Map<string, number>() // normalized email → first lineNo
  const rows: BatchUserParseRow[] = []
  let validCount = 0
  let errorCount = 0
  let duplicateCount = 0

  for (const r of raw) {
    const out: BatchUserParseRow = { lineNo: r.lineNo, raw: r.cells, valid: false }
    // Pad cells to 6 columns
    const cells = [...r.cells]
    while (cells.length < 6) cells.push('')

    const [emailCell, passwordCell, usernameCell, balanceCell, concurrencyCell, rpmCell] = cells
    const email = (emailCell || '').trim()
    const normalized = email.toLowerCase()
    const password = passwordCell || ''
    const username = (usernameCell || '').trim()

    if (!normalized) {
      out.errorCode = 'INVALID_EMAIL'
      out.errorMsg = 'email is required'
    } else if (!looksLikeEmail(normalized)) {
      out.errorCode = 'INVALID_EMAIL'
      out.errorMsg = 'email format is invalid'
    } else if (password.length < 6) {
      out.errorCode = 'WEAK_PASSWORD'
      out.errorMsg = 'password must be at least 6 characters'
    } else {
      const bal = parseNumberCell(balanceCell, true)
      const conc = parseNumberCell(concurrencyCell, false)
      const rpm = parseNumberCell(rpmCell, false)
      if (!bal.ok || bal.value < 0) {
        out.errorCode = 'INVALID_BALANCE'
        out.errorMsg = 'balance must be a non-negative number'
      } else if (!conc.ok || conc.value < 0) {
        out.errorCode = 'INVALID_CONCURRENCY'
        out.errorMsg = 'concurrency must be a non-negative integer'
      } else if (!rpm.ok || rpm.value < 0) {
        out.errorCode = 'INVALID_RPM'
        out.errorMsg = 'rpm must be a non-negative integer'
      } else if (seen.has(normalized)) {
        out.errorCode = 'DUPLICATE_IN_PAYLOAD'
        out.errorMsg = `email duplicates line ${seen.get(normalized)}`
        duplicateCount++
      } else {
        out.valid = true
        out.row = {
          email,
          password,
          username,
          balance: bal.value,
          concurrency: conc.value,
          rpm_limit: rpm.value,
        }
        seen.set(normalized, r.lineNo)
      }
    }

    if (out.valid) {
      validCount++
    } else {
      errorCount++
    }
    rows.push(out)
  }

  return { rows, validCount, errorCount, duplicateCount, delimiter, headerSkipped }
}
