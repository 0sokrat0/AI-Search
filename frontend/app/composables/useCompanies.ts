export interface Company {
  id: string
  name: string
}

const SETTINGS_KEY = 'companies_json'

export function useCompanies() {
  const companies = ref<Company[]>([])
  const loading = ref(false)

  async function fetchCompanies() {
    loading.value = true
    try {
      const settings = await $fetch<Record<string, string>>('/api/settings')
      const raw = settings?.[SETTINGS_KEY]
      if (raw) {
        try {
          companies.value = JSON.parse(raw) as Company[]
        } catch {
          companies.value = []
        }
      } else {
        companies.value = []
      }
    } finally {
      loading.value = false
    }
  }

  async function saveCompanies(list: Company[]) {
    await $fetch('/api/settings', {
      method: 'PUT',
      body: { [SETTINGS_KEY]: JSON.stringify(list) }
    })
    companies.value = list
  }

  async function addCompany(name: string) {
    const trimmed = name.trim()
    if (!trimmed) return
    const next = [...companies.value, { id: crypto.randomUUID(), name: trimmed }]
    await saveCompanies(next)
  }

  async function removeCompany(id: string) {
    const next = companies.value.filter(c => c.id !== id)
    await saveCompanies(next)
  }

  onMounted(fetchCompanies)

  return {
    companies,
    loading,
    addCompany,
    removeCompany,
    refresh: fetchCompanies
  }
}
