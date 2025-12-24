import { ref } from 'vue'

const snackbar = ref({
  show: false,
  text: '',
  color: 'success',
  timeout: 3000
})

export function useSnackbar() {
  function showSnackbar(text, color = 'success') {
    snackbar.value = {
      show: true,
      text,
      color,
      timeout: color === 'error' ? 5000 : 3000
    }
  }

  function hideSnackbar() {
    snackbar.value.show = false
  }

  return {
    snackbar,
    showSnackbar,
    hideSnackbar
  }
}
