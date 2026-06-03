import './global.css'
import type { Metadata } from 'next'
import type { ReactNode } from 'react'

import { RootProvider } from 'fumadocs-ui/provider/next'
import { Geist, Geist_Mono, Orbitron } from 'next/font/google'

const geist = Geist({ subsets: ['latin'], variable: '--font-geist' })
const geistMono = Geist_Mono({ subsets: ['latin'], variable: '--font-geist-mono' })
const orbitron = Orbitron({
  subsets: ['latin'],
  variable: '--font-orbitron',
  weight: ['400', '500', '600', '700', '800', '900'],
})

export const metadata: Metadata = {
  title: {
    template: '%s | yori',
    default: 'yori — the home for everything you tell your AI',
  },
  description:
    'A local, file-based library of reusable AI building blocks — prompts, agents, slash-commands, skills — that you compose with Liquid and render into ready-to-pipe text.',
}

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body
        className={`${geist.variable} ${geistMono.variable} ${orbitron.variable} font-sans antialiased`}
      >
        <RootProvider
          search={{
            options: {
              type: 'static',
              api: `${process.env.NEXT_PUBLIC_BASE_PATH ?? ''}/api/search`,
            },
          }}
        >
          {children}
        </RootProvider>
      </body>
    </html>
  )
}
