'use client'

import { Fragment, useEffect, useRef, useState } from 'react'

type Line = { kind: 'cmd' | 'out' | 'dim'; text: string }
type PanelPhase = 'empty' | 'template' | 'rendering' | 'rendered'

type Step =
  | { t: 'cmd'; text: string }
  | { t: 'out'; lines: string[] }
  | { t: 'dim'; lines: string[] }
  | { t: 'panel'; phase: 'empty' | 'template' }
  | { t: 'render' } // morph the panel template → rendered, line by line
  | { t: 'wait'; ms: number }

// The terminal box has a fixed height; cap how many lines render so the
// newest output is always fully visible (older lines drop off cleanly
// rather than the top line getting clipped mid-glyph).
const MAX_VISIBLE_LINES = 9

// The artifact shown in the panel. Each row carries its raw template form and
// its rendered form; `hl` marks the substring that the render *filled in*, so
// we can highlight exactly what changed. Unchanged rows render muted.
type PRow = { tmpl: string; rendered: string; hl?: string }

const ARTIFACT: PRow[] = [
  { tmpl: "{% include 'style' %}", rendered: 'Be concise.', hl: 'Be concise.' },
  { tmpl: '', rendered: '' },
  { tmpl: 'Analyze this log as a', rendered: 'Analyze this log as a' },
  { tmpl: '{{ tone }} engineer:', rendered: 'blunt engineer:', hl: 'blunt' },
  { tmpl: '', rendered: '' },
  { tmpl: '{{ input }}', rendered: 'NPE at Auth.java:42', hl: 'NPE at Auth.java:42' },
]

const CYAN = 'oklch(0.78 0.16 195)'
const VIOLET = 'oklch(0.72 0.18 285)'
const GREEN = 'oklch(0.72 0.17 150)'

// The looping demo: type a command, print its output, then drive the render
// panel so the two stay in sync — yori's compose → render loop, the I/O Tower
// turning a template into finished text.
const SCRIPT: Step[] = [
  { t: 'cmd', text: 'yori init' },
  { t: 'out', lines: ['✓ created ./.yori/store'] },
  { t: 'wait', ms: 450 },

  { t: 'cmd', text: 'yori add triage' },
  { t: 'out', lines: ['✓ wrote store/triage.md'] },
  { t: 'panel', phase: 'template' },
  { t: 'wait', ms: 1100 },

  { t: 'cmd', text: 'echo "NPE at Auth.java:42" | yori run triage --tone=blunt' },
  { t: 'render' },
  { t: 'dim', lines: ['→ piped to stdout'] },
  { t: 'wait', ms: 2600 },
]

// Static final frame for reduced-motion / no-JS.
const STATIC_LINES: Line[] = SCRIPT.flatMap((s) =>
  s.t === 'cmd'
    ? [{ kind: 'cmd', text: s.text } as Line]
    : s.t === 'out'
      ? s.lines.map((l) => ({ kind: 'out', text: l }) as Line)
      : s.t === 'dim'
        ? s.lines.map((l) => ({ kind: 'dim', text: l }) as Line)
        : [],
)

// Split a template line into runs, colouring Liquid tokens ({{ … }} cyan,
// {% … %} violet) so the placeholders read as placeholders.
function renderTemplateLine(text: string) {
  if (!text) return <>&nbsp;</>
  const parts = text.split(/(\{\{[^}]*\}\}|\{%[^%]*%\})/g)
  return (
    <>
      {parts.map((p, i) => {
        if (p.startsWith('{{'))
          return (
            <span key={i} style={{ color: CYAN }}>
              {p}
            </span>
          )
        if (p.startsWith('{%'))
          return (
            <span key={i} style={{ color: VIOLET }}>
              {p}
            </span>
          )
        return <Fragment key={i}>{p}</Fragment>
      })}
    </>
  )
}

// Render the resolved line, highlighting the filled-in substring green.
function renderResolvedLine(row: PRow) {
  if (!row.rendered) return <>&nbsp;</>
  if (!row.hl) return <span className="text-fd-muted-foreground">{row.rendered}</span>
  const idx = row.rendered.indexOf(row.hl)
  if (idx < 0) return <span className="text-fd-muted-foreground">{row.rendered}</span>
  return (
    <span className="text-fd-muted-foreground">
      {row.rendered.slice(0, idx)}
      <span style={{ color: GREEN }}>{row.hl}</span>
      {row.rendered.slice(idx + row.hl.length)}
    </span>
  )
}

export function HeroTerminal() {
  const [lines, setLines] = useState<Line[]>([])
  const [typing, setTyping] = useState('')
  const [done, setDone] = useState(false)
  const [phase, setPhase] = useState<PanelPhase>('empty')
  // How many rows have flipped from template → rendered during the morph.
  const [revealed, setRevealed] = useState(0)
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const reduce =
      typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
    if (reduce) {
      setLines(STATIC_LINES)
      setPhase('rendered')
      setRevealed(ARTIFACT.length)
      setDone(true)
      return
    }

    let cancelled = false
    const timers: ReturnType<typeof setTimeout>[] = []
    const sleep = (ms: number) =>
      new Promise<void>((res) => {
        timers.push(setTimeout(res, ms))
      })

    async function run() {
      while (!cancelled) {
        setLines([])
        setTyping('')
        setPhase('empty')
        setRevealed(0)
        for (const step of SCRIPT) {
          if (cancelled) return
          if (step.t === 'cmd') {
            for (let i = 1; i <= step.text.length; i++) {
              if (cancelled) return
              setTyping(step.text.slice(0, i))
              await sleep(18)
            }
            setLines((l) => [...l, { kind: 'cmd', text: step.text }])
            setTyping('')
            await sleep(140)
          } else if (step.t === 'out') {
            setLines((l) => [...l, ...step.lines.map((t) => ({ kind: 'out', text: t }) as Line)])
            await sleep(140)
          } else if (step.t === 'dim') {
            setLines((l) => [...l, ...step.lines.map((t) => ({ kind: 'dim', text: t }) as Line)])
            await sleep(140)
          } else if (step.t === 'panel') {
            setPhase(step.phase)
          } else if (step.t === 'render') {
            setPhase('rendering')
            await sleep(260)
            for (let i = 1; i <= ARTIFACT.length; i++) {
              if (cancelled) return
              setRevealed(i)
              await sleep(190)
            }
            setPhase('rendered')
          } else if (step.t === 'wait') {
            await sleep(step.ms)
          }
        }
      }
    }
    run()
    return () => {
      cancelled = true
      timers.forEach(clearTimeout)
    }
  }, [])

  const morphing = phase === 'rendering' || phase === 'rendered'
  const headerLabel =
    phase === 'empty'
      ? 'no artifact'
      : phase === 'template'
        ? 'store/triage.md'
        : phase === 'rendering'
          ? 'rendering…'
          : 'rendered → stdout'
  const headerColor = phase === 'rendered' ? GREEN : phase === 'empty' ? undefined : CYAN

  return (
    <div className="grid gap-4 md:grid-cols-5">
      {/* Terminal */}
      <div
        className="overflow-hidden rounded-xl border md:col-span-3"
        style={{ borderColor: 'oklch(1 0 0 / 10%)', background: 'oklch(0.08 0.01 240)' }}
      >
        <div
          className="flex items-center gap-2 border-b px-4 py-2.5"
          style={{ borderColor: 'oklch(1 0 0 / 5%)' }}
        >
          <div className="flex gap-1.5">
            <span className="block h-2.5 w-2.5 rounded-full bg-red-500/60" />
            <span className="block h-2.5 w-2.5 rounded-full bg-yellow-500/60" />
            <span className="block h-2.5 w-2.5 rounded-full bg-green-500/60" />
          </div>
          <span className="text-fd-muted-foreground ml-2 font-mono text-[10px]">
            ~/project — zsh
          </span>
        </div>
        <div
          ref={scrollRef}
          className="flex h-[208px] flex-col justify-end overflow-hidden p-4 font-mono text-[11px] leading-relaxed"
        >
          {/* Keep only the last few lines so newest output is always fully
              visible — the box has a fixed height, so rendering everything
              would clip the top line mid-glyph instead of scrolling. */}
          {lines.slice(done ? -MAX_VISIBLE_LINES : -(MAX_VISIBLE_LINES - 1)).map((line, i) => (
            <div
              key={i}
              className={
                line.kind === 'cmd'
                  ? 'text-fd-foreground truncate'
                  : line.kind === 'dim'
                    ? 'text-fd-muted-foreground/60 italic'
                    : 'text-fd-muted-foreground'
              }
            >
              {line.kind === 'cmd' ? (
                <>
                  <span style={{ color: CYAN }}>$ </span>
                  {line.text}
                </>
              ) : (
                line.text || ' '
              )}
            </div>
          ))}
          {!done && (
            <div className="text-fd-foreground truncate">
              <span style={{ color: CYAN }}>$ </span>
              {typing}
              <span className="terminal-cursor" style={{ color: CYAN }}>
                ▋
              </span>
            </div>
          )}
        </div>
      </div>

      {/* The render panel: triage.md template morphing into rendered output */}
      <div
        className="flex flex-col rounded-xl border p-3 md:col-span-2"
        style={{ borderColor: 'oklch(1 0 0 / 10%)', background: 'oklch(0.1 0.01 240 / 60%)' }}
      >
        <div className="mb-2 flex items-center gap-1.5">
          <span
            className="font-mono text-[9px] tracking-[0.2em] uppercase transition-colors duration-300"
            style={{ color: headerColor ?? 'oklch(0.65 0.01 240)' }}
          >
            {headerLabel}
          </span>
          {phase === 'template' && (
            <span className="text-fd-muted-foreground/50 font-mono text-[9px]">· liquid</span>
          )}
        </div>

        <div className="min-h-[176px] flex-1 font-mono text-[10.5px] leading-relaxed">
          {phase === 'empty' ? (
            <div className="text-fd-muted-foreground/50 py-12 text-center text-[10px]">
              empty store
            </div>
          ) : (
            ARTIFACT.map((row, i) => {
              const showRendered = morphing && i < revealed
              return (
                <div
                  key={i}
                  className={showRendered ? 'ticket-in' : undefined}
                  style={{ minHeight: '1.45em' }}
                >
                  {showRendered ? (
                    renderResolvedLine(row)
                  ) : (
                    <span className="text-fd-foreground/90">{renderTemplateLine(row.tmpl)}</span>
                  )}
                </div>
              )
            })
          )}
        </div>
      </div>
    </div>
  )
}
