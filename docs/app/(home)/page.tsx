import Link from 'next/link'

import { HeroTerminal } from './hero-terminal'

const learningPaths = [
  {
    title: 'Getting Started',
    description:
      'Install yori, init a store, scaffold your first artifact, and render it into a Unix pipeline.',
    href: '/docs/getting-started',
    icon: '▶',
    accent: 'oklch(0.78 0.16 195)',
    links: [
      { label: 'Install', href: '/docs/getting-started/install' },
      { label: 'Quickstart', href: '/docs/getting-started/quickstart' },
      { label: 'Concepts', href: '/docs/concepts' },
    ],
  },
  {
    title: 'Templating',
    description:
      'Compose with Liquid: variables, partial includes, conditionals, loops, and template inheritance via slots.',
    href: '/docs/templating',
    icon: '◆',
    accent: 'oklch(0.72 0.18 285)',
    links: [
      { label: 'Variables', href: '/docs/templating#variables' },
      { label: 'Partials & includes', href: '/docs/templating#partials' },
      { label: 'Inheritance (slots)', href: '/docs/templating#inheritance' },
    ],
  },
  {
    title: 'Registry',
    description:
      'Git-as-registry: install a team’s shelf, vendor individual items with their deps, and publish your own.',
    href: '/docs/registry',
    icon: '●',
    accent: 'oklch(0.78 0.14 60)',
    links: [
      { label: 'Packages', href: '/docs/registry#packages' },
      { label: 'Items & manifests', href: '/docs/registry#items' },
      { label: 'Context-aware install', href: '/docs/registry#context-aware' },
    ],
  },
]

const categoryCards = [
  {
    title: 'Commands',
    description: 'Every yori subcommand, its flags, and what it does.',
    href: '/docs/commands',
    icon: '⌨',
  },
  {
    title: 'Concepts',
    description: 'Artifacts, the four types, and the layered store.',
    href: '/docs/concepts',
    icon: '🧠',
  },
  {
    title: 'Deploy to agents',
    description: '`yori sync` renders skills, commands & subagents into Claude, Codex, Cursor.',
    href: '/docs/sync',
    icon: '🔌',
  },
  {
    title: 'Evaluate',
    description: '`yori export promptfoo` hands a composed artifact + its cases to promptfoo.',
    href: '/docs/eval',
    icon: '🧪',
  },
  {
    title: 'Design notes',
    description: 'Sticky decisions: pure renderer, files as truth, git as the network.',
    href: '/docs/design',
    icon: '◇',
  },
  {
    title: 'FAQ',
    description: 'What yori is not, and why.',
    href: '/docs/faq',
    icon: '?',
  },
]

export default function HomePage() {
  return (
    <main className="relative flex min-h-screen flex-col overflow-hidden px-6 py-16 md:py-24">
      {/* Ambient drifting TRON grid */}
      <div className="grid-backdrop pointer-events-none absolute inset-0 -z-10" />
      <div
        className="pointer-events-none absolute inset-0 -z-10"
        style={{
          background:
            'radial-gradient(ellipse 60% 50% at 50% 20%, oklch(0.78 0.16 195 / 10%) 0%, transparent 70%), radial-gradient(ellipse 40% 40% at 75% 70%, oklch(0.72 0.18 285 / 6%) 0%, transparent 70%)',
        }}
      />

      {/* Hero */}
      <div className="animate-fade-in-up mx-auto w-full max-w-5xl">
        <div
          className="relative overflow-hidden rounded-2xl border p-8 md:p-10"
          style={{
            borderColor: 'oklch(1 0 0 / 10%)',
            background: 'linear-gradient(135deg, oklch(1 0 0 / 3%) 0%, transparent 100%)',
          }}
        >
          <div
            className="absolute top-0 left-0 h-28 w-28 rounded-tl-2xl"
            style={{
              borderLeft: '2px solid oklch(0.78 0.16 195 / 25%)',
              borderTop: '2px solid oklch(0.78 0.16 195 / 25%)',
            }}
          />
          <div
            className="absolute right-0 bottom-0 h-28 w-28 rounded-br-2xl"
            style={{
              borderRight: '2px solid oklch(0.72 0.18 285 / 25%)',
              borderBottom: '2px solid oklch(0.72 0.18 285 / 25%)',
            }}
          />

          <div className="relative space-y-5">
            <p
              className="text-xs font-medium tracking-[0.25em] uppercase"
              style={{ color: 'oklch(0.78 0.16 195)' }}
            >
              Documentation
            </p>
            <h1 className="text-4xl font-bold tracking-tight md:text-5xl">yori</h1>
            <p className="text-fd-muted-foreground max-w-xl text-base md:text-lg">
              The home for everything you tell your AI. A local, file-based library of reusable
              prompts, agents, commands &amp; skills — composed with Liquid, rendered to stdout.
              Pure text in, text out; yori never calls a model.
            </p>

            <div className="flex flex-wrap items-center gap-3 pt-2">
              <Link
                href="/docs/getting-started"
                className="text-fd-primary-foreground inline-flex items-center gap-2 rounded-lg px-6 py-2.5 text-sm font-semibold transition-all hover:brightness-110"
                style={{
                  background:
                    'linear-gradient(135deg, oklch(0.78 0.16 195) 0%, oklch(0.62 0.18 210) 100%)',
                  boxShadow: '0 0 24px oklch(0.78 0.16 195 / 25%)',
                }}
              >
                Get started <span aria-hidden>→</span>
              </Link>
              <Link
                href="/docs/commands"
                className="border-fd-border text-fd-foreground hover:bg-fd-accent inline-flex items-center gap-2 rounded-lg border px-6 py-2.5 text-sm font-medium transition-colors"
              >
                Command reference
              </Link>
            </div>
          </div>
        </div>
      </div>

      {/* The core loop — live terminal + reacting shelf */}
      <div className="animate-fade-in-up mx-auto mt-8 w-full max-w-5xl">
        <p
          className="mb-4 text-xs font-medium tracking-[0.25em] uppercase"
          style={{ color: 'oklch(0.78 0.16 195)' }}
        >
          The core loop
        </p>
        <HeroTerminal />
      </div>

      {/* Learning Paths */}
      <div className="mx-auto mt-12 w-full max-w-5xl">
        <p
          className="mb-6 text-xs font-medium tracking-[0.25em] uppercase"
          style={{ color: 'oklch(0.78 0.16 195)' }}
        >
          Start here
        </p>

        <div className="grid gap-4 md:grid-cols-3">
          {learningPaths.map((path, i) => (
            <div
              key={path.title}
              className="animate-fade-in-up group border-fd-border bg-fd-card hover:border-fd-primary/30 relative overflow-hidden rounded-xl border transition-colors"
              style={{ animationDelay: `${i * 80}ms` }}
            >
              <div
                className="h-[2px]"
                style={{ background: `linear-gradient(90deg, ${path.accent}, transparent)` }}
              />
              <div className="p-5">
                <Link href={path.href} className="mb-3 flex items-center gap-2">
                  <span className="text-2xl">{path.icon}</span>
                  <span
                    className="text-fd-foreground group-hover:text-fd-primary text-lg font-semibold transition-colors"
                    style={{
                      fontFamily: 'var(--font-orbitron), sans-serif',
                      letterSpacing: '0.08em',
                    }}
                  >
                    {path.title}
                  </span>
                </Link>
                <p className="text-fd-muted-foreground mb-4 text-sm leading-relaxed">
                  {path.description}
                </p>
                <div
                  className="space-y-1 border-t pt-3"
                  style={{ borderColor: 'oklch(1 0 0 / 5%)' }}
                >
                  {path.links.map((link) => (
                    <Link
                      key={link.href}
                      href={link.href}
                      className="group/link text-fd-muted-foreground hover:bg-fd-accent hover:text-fd-accent-foreground flex items-center justify-between rounded-md px-2 py-1.5 text-sm transition-colors"
                    >
                      <span>{link.label}</span>
                      <span className="-translate-x-2 text-xs opacity-0 transition-all group-hover/link:translate-x-0 group-hover/link:opacity-100">
                        →
                      </span>
                    </Link>
                  ))}
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Categories */}
      <div className="mx-auto mt-16 w-full max-w-5xl">
        <p
          className="mb-6 text-xs font-medium tracking-[0.25em] uppercase"
          style={{ color: 'oklch(0.72 0.18 285)' }}
        >
          Browse
        </p>

        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {categoryCards.map((card, i) => (
            <Link
              key={card.title}
              href={card.href}
              className="animate-fade-in-up group border-fd-border bg-fd-card hover:border-fd-primary/30 block rounded-xl border p-5 transition-colors"
              style={{ animationDelay: `${(i + 3) * 60}ms` }}
            >
              <div className="flex items-start gap-3">
                <span className="text-xl">{card.icon}</span>
                <div>
                  <h3 className="text-fd-foreground group-hover:text-fd-primary text-base font-semibold transition-colors">
                    {card.title}
                  </h3>
                  <p className="text-fd-muted-foreground mt-0.5 text-xs">{card.description}</p>
                </div>
              </div>
            </Link>
          ))}
        </div>
      </div>

      {/* CTA */}
      <div className="mx-auto mt-16 w-full max-w-5xl">
        <div
          className="animate-fade-in-up relative overflow-hidden rounded-2xl"
          style={{ animationDelay: '400ms' }}
        >
          <div
            className="absolute inset-0 rounded-2xl opacity-25"
            style={{
              background:
                'linear-gradient(90deg, oklch(0.78 0.16 195), oklch(0.72 0.18 285), oklch(0.78 0.16 195))',
            }}
          />
          <div className="bg-fd-background absolute inset-[1px] rounded-2xl" />

          <div className="relative flex flex-col items-center justify-between gap-6 p-8 md:flex-row md:p-10">
            <div className="space-y-1">
              <h3 className="text-fd-foreground text-xl font-bold">
                Compose once, deploy everywhere.
              </h3>
              <p className="text-fd-muted-foreground max-w-md text-sm">
                Give every prompt a single place to live. Render it into any pipeline, share it over
                git, and sync it into the dirs Claude Code, Codex &amp; Cursor read from.
              </p>
            </div>
            <div className="flex shrink-0 items-center gap-3">
              <Link
                href="/docs/getting-started/quickstart"
                className="text-fd-primary-foreground inline-flex items-center gap-2 rounded-lg px-6 py-2.5 text-sm font-semibold transition-all hover:brightness-110"
                style={{
                  background:
                    'linear-gradient(135deg, oklch(0.78 0.16 195) 0%, oklch(0.62 0.18 210) 100%)',
                  boxShadow: '0 0 20px oklch(0.78 0.16 195 / 20%)',
                }}
              >
                Quickstart
              </Link>
              <a
                href="https://github.com/arjia-labs/yori"
                target="_blank"
                rel="noopener noreferrer"
                className="border-fd-border text-fd-foreground hover:bg-fd-accent inline-flex items-center gap-2 rounded-lg border px-6 py-2.5 text-sm font-medium transition-colors"
              >
                GitHub
              </a>
            </div>
          </div>
        </div>
      </div>
    </main>
  )
}
