'use client'

import { ThemeProvider } from 'next-themes'
import { SWRConfig } from 'swr'
import { ReactNode } from 'react'
import { apiClient } from './lib/api'

export function Providers({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider attribute="class" defaultTheme="dark" enableSystem={false}>
      <SWRConfig 
        value={{
          fetcher: apiClient,
          refreshInterval: 0, // Manual polling via hooks
          revalidateOnFocus: true,
          shouldRetryOnError: true,
          errorRetryCount: 3,
        }}
      >
        {children}
      </SWRConfig>
    </ThemeProvider>
  )
}
