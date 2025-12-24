<template>
  <div>
    <router-view />
  </div>
</template>

<script setup>
import { useAuthStore } from '@/stores/auth'
import { useRouter } from 'vue-router'
import { computed, onMounted } from 'vue'

const router = useRouter()
const authStore = useAuthStore()
const isAdmin = computed(() => authStore.isAdmin)

onMounted(() => {
  if (!isAdmin.value) {
    router.push('/')
  }
})
</script>
