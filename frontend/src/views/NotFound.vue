<template>
  <div class="notfound-container">
    <v-card max-width="500" class="text-center">
      <v-card-text class="py-8">
        <v-icon size="128" color="grey">mdi-alert-circle-outline</v-icon>

        <div class="text-h3 mt-4">404</div>
        <div class="text-h6 mt-2">页面未找到</div>
        <div class="text-grey mt-2">
          您访问的页面不存在或已被移除
        </div>

        <v-btn
          color="primary"
          variant="tonal"
          class="mt-6"
          to="/"
        >
          <v-icon start>mdi-home</v-icon>
          返回首页
        </v-btn>
      </v-card-text>
    </v-card>
  </div>
</template>

<style scoped>
.notfound-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
}
</style>
