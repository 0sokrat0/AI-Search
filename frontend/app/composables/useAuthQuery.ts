import { useQuery } from '@tanstack/vue-query'
import type { UseQueryOptions, QueryKey } from '@tanstack/vue-query'
import type { Ref, ComputedRef } from 'vue'

type MaybeRef<T> = T | Ref<T> | ComputedRef<T>

export function useAuthQuery<TData, TError = Error, TQueryKey extends QueryKey = QueryKey>(
  queryKey: MaybeRef<TQueryKey>,
  queryFn: () => Promise<TData>,
  options: Omit<UseQueryOptions<TData, TError>, 'queryKey' | 'queryFn'> = {}
) {
  const authStore = useAuthStore()
  const router = useRouter()

  const query = useQuery<TData, TError>({
    queryKey: queryKey as any,
    queryFn,
    ...options
  })

  watch(query.error, (err) => {
    if (!err) return
    const e = err as Record<string, unknown>
    const status = e?.statusCode ?? e?.status ?? (e?.data as Record<string, unknown>)?.statusCode
    if (status === 401) {
      authStore.clearAuth()
      router.push('/login')
    }
  })

  return query
}
