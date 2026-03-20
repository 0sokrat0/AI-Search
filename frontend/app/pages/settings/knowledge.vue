<script setup lang="ts">
import type { KnowledgeImportResult } from '~/types'

definePageMeta({
  middleware: 'auth'
})

const toast = useToast()
const auth = useAuthStore()

if (!auth.isSuperAdmin) {
  await navigateTo('/')
}

const knowledgeFile = ref<File | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const importing = ref(false)
const result = ref<KnowledgeImportResult | null>(null)
const importState = ref<'idle' | 'ready' | 'uploading' | 'done'>('idle')
const previewRows = ref<Array<{ text: string, category: string }>>([])
const previewError = ref('')
const formatHint = 'CSV с двумя колонками: "текст" и "категория". Также поддерживаются "целевое сообщение" + "тип лида" и старый format text,category. Допустимые типы: merchants, ps_offers, trader_search, traders, noise.'
const previewColumns = [
  { id: 'text', accessorKey: 'text', header: 'Целевое сообщение' },
  { id: 'category', accessorKey: 'category', header: 'Тип лида' }
]
const resultRows = computed(() => {
  if (!result.value) return []

  return [
    { category: 'merchants', count: result.value.merchants },
    { category: 'ps_offers', count: result.value.psOffers },
    { category: 'trader_search', count: result.value.traderSearch },
    { category: 'traders', count: result.value.traders },
    { category: 'noise', count: result.value.noise }
  ]
})
const resultColumns = [
  { id: 'category', accessorKey: 'category', header: 'Категория' },
  { id: 'count', accessorKey: 'count', header: 'Записей' }
]

function onFileChange(event: Event) {
  const target = event.target as HTMLInputElement | null
  const nextFile = target?.files?.[0] ?? null
  knowledgeFile.value = nextFile
  result.value = null
  previewRows.value = []
  previewError.value = ''
  importState.value = nextFile ? 'ready' : 'idle'

  if (!nextFile) {
    return
  }

  void buildPreview(nextFile)
}

function openFilePicker() {
  fileInput.value?.click()
}

async function importKnowledge() {
  if (!knowledgeFile.value) {
    toast.add({ title: 'Файл не выбран', description: 'Загрузите CSV файл.', color: 'warning' })
    return
  }

  importing.value = true
  importState.value = 'uploading'
  result.value = null

  try {
    const formData = new FormData()
    formData.append('file', knowledgeFile.value)

    const response = await $fetch<KnowledgeImportResult>('/api/settings/knowledge/import', {
      method: 'POST',
      body: formData
    })

    result.value = response
    knowledgeFile.value = null
    importState.value = 'done'
    if (fileInput.value) {
      fileInput.value.value = ''
    }
    toast.add({ title: 'Данные RAG загружены', description: `Импортировано ${response.imported} записей.`, color: 'success' })
  } catch (error: any) {
    importState.value = 'ready'
    toast.add({ title: 'Ошибка импорта', description: error?.data?.message || error?.message || 'Не удалось импортировать файл.', color: 'error' })
  } finally {
    importing.value = false
  }
}

async function buildPreview(file: File) {
  try {
    const content = await file.text()
    previewRows.value = parsePreviewRows(content)
    previewError.value = previewRows.value.length === 0 ? 'Не удалось распознать строки для предварительного просмотра.' : ''
  } catch {
    previewRows.value = []
    previewError.value = 'Не удалось прочитать файл для предварительного просмотра.'
  }
}

function parsePreviewRows(content: string) {
  const rows = parseCSVContent(content)
  if (rows.length <= 1) {
    return []
  }

  const [headerRow] = rows
  if (!headerRow) {
    return []
  }

  const headers = headerRow.map(value => normalizeColumnName(value))
  const textIndex = headers.findIndex(value => ['text', 'текст', 'targetmessage', 'targetmsg', 'целевоесообщение', 'сообщение'].includes(value))
  const categoryIndex = headers.findIndex(value => ['category', 'категория', 'leadtype', 'leadkind', 'типлида', 'тип'].includes(value))

  if (textIndex === -1 || categoryIndex === -1) {
    previewError.value = 'В файле должны быть колонки "текст" и "категория" либо совместимые названия вроде "целевое сообщение" и "тип лида".'
    return []
  }

  return rows.slice(1, 7).map((values) => {
    return {
      text: (values[textIndex] || '').trim(),
      category: (values[categoryIndex] || '').trim()
    }
  }).filter(row => row.text && row.category)
}

function parseCSVContent(content: string) {
  const delimiter = detectDelimiter(content)
  const rows: string[][] = []
  let current = ''
  let row: string[] = []
  let inQuotes = false

  for (let i = 0; i < content.length; i++) {
    const char = content[i]
    if (char === '"') {
      if (inQuotes && content[i + 1] === '"') {
        current += '"'
        i++
        continue
      }
      inQuotes = !inQuotes
      continue
    }
    if (char === delimiter && !inQuotes) {
      row.push(current)
      current = ''
      continue
    }
    if ((char === '\n' || char === '\r') && !inQuotes) {
      if (char === '\r' && content[i + 1] === '\n') {
        i++
      }
      row.push(current)
      if (row.some(value => value.trim() !== '')) {
        rows.push(row.map(value => value.trim()))
      }
      row = []
      current = ''
      continue
    }
    current += char
  }

  row.push(current)
  if (row.some(value => value.trim() !== '')) {
    rows.push(row.map(value => value.trim()))
  }

  return rows
}

function detectDelimiter(content: string) {
  const firstLine = content.split(/\r?\n/, 1)[0] || ''
  return firstLine.includes(';') && !firstLine.includes(',') ? ';' : ','
}

function normalizeColumnName(value: string) {
  return value
    .trim()
    .toLowerCase()
    .replace(/[^\p{L}\p{N}]+/gu, '')
}
</script>

<template>
  <div class="space-y-6">
    <UPageCard
      title="RAG Данные"
      description="Загрузите стартовый CSV с эталонными сообщениями. Файл будет проэмбежен и загружен в Qdrant."
      variant="naked"
      orientation="horizontal"
    >
      <UButton
        label="Импортировать"
        color="primary"
        :loading="importing"
        :disabled="!knowledgeFile"
        class="w-fit lg:ms-auto"
        @click="importKnowledge"
      />
    </UPageCard>

    <UPageCard variant="subtle">
      <div class="space-y-4">
        <UAlert
          color="info"
          variant="soft"
          title="Формат файла"
          :description="formatHint"
        />

        <UAlert
          v-if="importState === 'ready' && knowledgeFile"
          color="neutral"
          variant="soft"
          title="Файл готов к импорту"
          :description="`Проверен файл ${knowledgeFile.name}. Можно запускать импорт.`"
        />

        <UAlert
          v-if="importState === 'uploading'"
          color="primary"
          variant="soft"
          title="Импорт выполняется"
          description="Запрос отправлен. Идёт чтение файла, эмбеддинг и загрузка точек в Qdrant."
        />

        <UFormField label="CSV файл">
          <input
            ref="fileInput"
            type="file"
            accept=".csv,text/csv"
            class="hidden"
            @change="onFileChange"
          >

          <div class="flex items-center gap-3 rounded-xl border border-default bg-elevated/60 p-3">
            <UButton
              label="Обзор"
              color="neutral"
              variant="solid"
              icon="i-lucide-file-up"
              @click="openFilePicker"
            />

            <div class="min-w-0 flex-1">
              <p class="truncate text-sm text-highlighted">
                {{ knowledgeFile?.name || 'Файл не выбран' }}
              </p>
              <p class="text-xs text-muted">
                Поддерживается только CSV
              </p>
            </div>
          </div>
        </UFormField>

        <UAlert
          v-if="previewError"
          color="warning"
          variant="soft"
          title="Проблема с предпросмотром"
          :description="previewError"
        />

        <UPageCard
          v-if="previewRows.length"
          variant="soft"
          title="Предпросмотр данных"
          description="Первые строки файла перед импортом."
        >
          <UTable
            :data="previewRows"
            :columns="previewColumns"
          />
        </UPageCard>

        <UPageCard
          v-if="result"
          variant="soft"
          :title="`Импорт завершен: ${result.imported} записей`"
          :description="`Файл: ${result.fileName}`"
        >
          <UTable
            :data="resultRows"
            :columns="resultColumns"
          />
        </UPageCard>
      </div>
    </UPageCard>
  </div>
</template>
