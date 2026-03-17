<template>
  <UCard>
    <template #header>
      <div class="flex items-start justify-between gap-4">
        <div>
          <p class="text-xs text-muted uppercase tracking-wide mb-1.5">
            Качество детекции лидов
          </p>
          <USkeleton class="h-8 w-24" />
        </div>
        <div class="flex items-center gap-5">
          <USkeleton class="h-10 w-16" />
          <USkeleton class="h-10 w-16" />
          <USkeleton class="h-10 w-16" />
        </div>
      </div>
    </template>
    <div class="flex items-end gap-1.5 px-1" style="height: 164px">
      <USkeleton
        v-for="i in 10"
        :key="i"
        class="flex-1 rounded-t"
        :style="`height: ${30 + i * 10}%`"
      />
    </div>
  </UCard>
</template>
