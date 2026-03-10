import { useState, useEffect } from "react"
import { useQuery } from "@tanstack/react-query"
import { checkSlug } from "@/lib/api-client"

export function useSlugCheck(slug: string) {
  const [debouncedSlug, setDebouncedSlug] = useState("")

  useEffect(() => {
    const value = slug.length < 3 ? "" : slug
    const timer = setTimeout(() => setDebouncedSlug(value), slug.length < 3 ? 0 : 300)
    return () => clearTimeout(timer)
  }, [slug])

  const { data, isFetching } = useQuery({
    queryKey: ["slug-check", debouncedSlug],
    queryFn: () => checkSlug(debouncedSlug),
    enabled: debouncedSlug.length >= 3,
  })

  return {
    isAvailable: data?.available ?? null,
    isChecking: isFetching || (slug.length >= 3 && slug !== debouncedSlug),
  }
}
