<template>
  <div class="cp-container">
    <div class="cp-card">
      <div class="logo">
        <h1>文豪写作</h1>
        <p>修改密码</p>
      </div>

      <form @submit.prevent="handleChangePassword">
        <div class="form-group">
          <label>邮箱</label>
          <input :value="email" type="email" disabled />
        </div>
        <div class="form-group">
          <label>验证码</label>
          <div class="code-row">
            <input v-model="form.code" type="text" placeholder="请输入6位验证码" maxlength="6" required />
            <button type="button" class="code-btn" :disabled="countdown > 0" @click="sendCode">
              {{ countdown > 0 ? countdown + 's' : '发送验证码' }}
            </button>
          </div>
        </div>
        <div class="form-group">
          <label>新密码</label>
          <input v-model="form.newPassword" type="password" placeholder="至少8位，包含大小写字母和数字" required />
          <p class="hint">至少8位，需包含大写字母、小写字母和数字</p>
        </div>
        <p v-if="errorMsg" class="error-msg">{{ errorMsg }}</p>
        <p v-if="successMsg" class="success-msg">{{ successMsg }}</p>
        <button type="submit" class="submit-btn" :disabled="loading">
          {{ loading ? '提交中...' : '修改密码' }}
        </button>
      </form>

      <div class="back-link">
        <a href="#" @click.prevent="goBack">返回</a>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const email = ref('')
const loading = ref(false)
const countdown = ref(0)
const errorMsg = ref('')
const successMsg = ref('')

const form = reactive({ code: '', newPassword: '' })

onMounted(async () => {
  const token = localStorage.getItem('token')
  const res = await fetch('/api/v1/me', {
    headers: { 'Authorization': 'Bearer ' + token },
  })
  if (res.status === 200) {
    const user = await res.json()
    email.value = user.email
  }
})

async function api(path, method, body) {
  const headers = { 'Content-Type': 'application/json' }
  const token = localStorage.getItem('token')
  if (token) headers['Authorization'] = 'Bearer ' + token
  const res = await fetch('/api/v1' + path, { method, headers, body: body ? JSON.stringify(body) : undefined })
  const data = await res.json()
  return { status: res.status, data }
}

async function sendCode() {
  errorMsg.value = ''
  successMsg.value = ''
  const { status, data } = await api('/auth/send-change-password-code', 'POST')
  if (status === 200) {
    successMsg.value = '验证码已发送'
    countdown.value = 60
    const timer = setInterval(() => {
      countdown.value--
      if (countdown.value <= 0) clearInterval(timer)
    }, 1000)
  } else {
    errorMsg.value = data.error || '发送失败'
  }
}

async function handleChangePassword() {
  loading.value = true
  errorMsg.value = ''
  successMsg.value = ''

  const { status, data } = await api('/auth/change-password', 'POST', {
    code: form.code,
    new_password: form.newPassword,
  })
  loading.value = false

  if (status === 200) {
    successMsg.value = '密码修改成功，请重新登录'
    setTimeout(() => {
      localStorage.removeItem('token')
      localStorage.removeItem('role')
      router.push('/login')
    }, 1500)
  } else {
    errorMsg.value = data.error || '修改失败'
  }
}

function goBack() {
  router.back()
}
</script>

<style scoped>
.cp-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #0f0c29, #302b63, #24243e);
}
.cp-card {
  width: 420px;
  padding: 40px;
}
.logo {
  text-align: center;
  margin-bottom: 36px;
}
.logo h1 {
  font-size: 36px;
  font-weight: 700;
  background: linear-gradient(135deg, #f5af19, #f12711);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  letter-spacing: 4px;
}
.logo p {
  color: #888;
  margin-top: 8px;
  font-size: 14px;
  letter-spacing: 2px;
}
.form-group {
  margin-bottom: 20px;
}
.form-group label {
  display: block;
  color: #aaa;
  font-size: 13px;
  margin-bottom: 8px;
}
.form-group input {
  width: 100%;
  padding: 12px 16px;
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 8px;
  background: rgba(255,255,255,0.05);
  color: #e0e0e0;
  font-size: 14px;
  outline: none;
  transition: border-color 0.3s;
  box-sizing: border-box;
}
.form-group input:focus {
  border-color: #f5af19;
}
.form-group input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.code-row {
  display: flex;
  gap: 10px;
}
.code-row input {
  flex: 1;
}
.code-btn {
  padding: 12px 16px;
  border: 1px solid #f5af19;
  border-radius: 8px;
  background: transparent;
  color: #f5af19;
  font-size: 13px;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.3s;
}
.code-btn:hover:not(:disabled) {
  background: rgba(245,175,25,0.1);
}
.code-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.hint {
  color: #666;
  font-size: 12px;
  margin-top: 6px;
}
.error-msg {
  color: #f12711;
  font-size: 13px;
  margin-bottom: 12px;
  text-align: center;
}
.success-msg {
  color: #4caf50;
  font-size: 13px;
  margin-bottom: 12px;
  text-align: center;
}
.submit-btn {
  width: 100%;
  padding: 14px;
  border: none;
  border-radius: 8px;
  background: linear-gradient(135deg, #f5af19, #f12711);
  color: #fff;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.3s;
}
.submit-btn:hover:not(:disabled) { opacity: 0.9; }
.submit-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.back-link {
  text-align: center;
  margin-top: 20px;
}
.back-link a {
  color: #888;
  font-size: 13px;
  text-decoration: none;
}
.back-link a:hover { color: #f5af19; }
</style>
