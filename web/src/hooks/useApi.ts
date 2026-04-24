import { useState, useEffect } from 'react'
import { useAuth } from '../contexts/AuthContext'

interface UseApiOptions {
  skip?: boolean
}

export const useApi = <T,>(
  url: string,
  options?: UseApiOptions,
) => {
  const [data, setData] = useState<T | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const { accessToken } = useAuth()

  useEffect(() => {
    if (options?.skip || !accessToken) return

    const fetchData = async () => {
      setIsLoading(true)
      setError(null)
      try {
        const response = await fetch(url, {
          headers: {
            'Authorization': `Bearer ${accessToken}`,
          },
        })

        if (!response.ok) {
          throw new Error(`API error: ${response.status}`)
        }

        const result = await response.json()
        setData(result)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error')
      } finally {
        setIsLoading(false)
      }
    }

    fetchData()
  }, [url, accessToken, options?.skip])

  return { data, isLoading, error }
}
