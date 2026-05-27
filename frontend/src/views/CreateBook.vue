<template>
  <div class="create-book-container">
    <header class="topbar">
      <div class="topbar-left">
        <router-link to="/write" class="back-link">← 返回</router-link>
        <h1>创建新小说</h1>
      </div>
      <div class="topbar-right">
        <span class="user-name">{{ user?.username }}</span>
        <button class="logout-btn" @click="logout">退出</button>
      </div>
    </header>

    <main class="main-content">
      <form class="book-form" @submit.prevent="submit">
        <div class="form-section">
          <h3>基本信息</h3>

          <div class="form-group">
            <label>书名 <span class="required">*</span></label>
            <input
              v-model="form.title"
              type="text"
              placeholder="给你的小说起个名字"
              maxlength="255"
              required
            />
          </div>

          <div class="form-row">
            <div class="form-group">
              <label>题材 <span class="required">*</span></label>
              <select v-model="form.genre_id" required>
                <option value="" disabled>请选择题材</option>
                <option v-for="g in genres" :key="g.id" :value="g.id">{{ g.name }}</option>
              </select>
            </div>

            <div class="form-group">
              <label>目标平台 <span class="required">*</span></label>
              <select v-model="form.platform_id" required>
                <option value="" disabled>请选择平台</option>
                <option v-for="p in platforms" :key="p.id" :value="p.id">{{ p.name }}</option>
              </select>
            </div>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label>每章目标字数 <span class="required">*</span></label>
              <input
                v-model.number="form.chapter_word_count"
                type="number"
                placeholder="3000"
                min="500"
                max="50000"
                required
              />
              <span class="hint">建议 2000-5000 字</span>
            </div>

            <div class="form-group">
              <label>目标总章数 <span class="required">*</span></label>
              <input
                v-model.number="form.target_chapters"
                type="number"
                placeholder="200"
                min="1"
                max="10000"
                required
              />
              <span class="hint">长篇建议 100-500 章</span>
            </div>
          </div>
        </div>

        <div class="form-section">
          <h3>书籍简介</h3>
          <div class="form-group">
            <textarea
              v-model="form.description"
              placeholder="简要描述你的故事背景、核心冲突和主角设定，AI 将基于此生成世界观和角色..."
              rows="6"
              maxlength="5000"
            ></textarea>
            <span class="hint char-count">{{ form.description.length }}/5000</span>
          </div>
        </div>

        <div class="form-section">
          <h3>AI 模型配置</h3>
          <div class="form-group">
            <label>写作模型</label>
            <select v-model="form.llm_config_id">
              <option :value="0">暂不选择（后续配置）</option>
              <option v-for="c in llmConfigs" :key="c.id" :value="c.id">
                {{ c.provider }} / {{ c.model || c.service }}
              </option>
            </select>
            <span class="hint" v-if="llmConfigs.length === 0">
              暂无可用模型，请先在设置中配置 LLM
            </span>
          </div>
        </div>

        <div class="form-actions">
          <button type="button" class="btn-cancel" @click="$router.push('/write')">取消</button>
          <button type="submit" class="btn-submit" :disabled="submitting">
            {{ submitting ? '创建中...' : '创建小说' }}
          </button>
        </div>

        <div v-if="error" class="error-msg">{{ error }}</div>
      </form>
    </main>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const user = ref(null)
const genres = ref([])
const platforms = ref([])
const llmConfigs = ref([])
const submitting = ref(false)
const error = ref('')

const form = reactive({
  title: '',
  genre_id: '',
  platform_id: '',
  chapter_word_count: 3000,
  target_chapters: 200,
  description: '',
  llm_config_id: 0,
})

const token = localStorage.getItem('token')
const headers = { 'Authorization': 'Bearer ' + token, 'Content-Type': 'application/json' }

onMounted(async () => {
  try {
    const [userRes, genreRes, platformRes, llmRes] = await Promise.all([
      fetch('/api/v1/me', { headers }),
      fetch('/api/v1/genres', { headers }),
      fetch('/api/v1/platforms', { headers }),
      fetch('/api/v1/llm-configs', { headers }),
    ])

    if (userRes.ok) user.value = await userRes.json()
    if (genreRes.ok) genres.value = await genreRes.json()
    if (platformRes.ok) platforms.value = await platformRes.json()
    if (llmRes.ok) llmConfigs.value = await llmRes.json()
  } catch (e) {
    error.value = '加载配置失败，请刷新重试'
  }
})

async function submit() {
  error.value = ''
  submitting.value = true

  const body = { ...form }
  if (body.llm_config_id === 0) delete body.llm_config_id

  try {
    const res = await fetch('/api/v1/books', {
      method: 'POST',
      headers,
      body: JSON.stringify(body),
    })

    if (res.ok) {
      router.push('/write')
    } else {
      const data = await res.json()
      error.value = data.error || '创建失败'
    }
  } catch (e) {
    error.value = '网络错误，请重试'
  } finally {
    submitting.value = false
  }
}

function logout() {
  localStorage.removeItem('token')
  localStorage.removeItem('role')
  router.push('/login')
}
</script>

<style scoped>
.create-book-container {
  min-height: 100vh;
  background: linear-gradient(135deg, #0f0c29, #302b63, #24243e);
  display: flex;
  flex-direction: column;
}

.topbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 32px;
  border-bottom: 1px solid rgba(255,255,255,0.06);
}

.topbar-left {
  display: flex;
  align-items: center;
  gap: 20px;
}

.back-link {
  color: #888;
  text-decoration: none;
  font-size: 14px;
  transition: color 0.2s;
}

.back-link:hover { color: #f5af19; }

.topbar-left h1 {
  font-size: 20px;
  background: linear-gradient(135deg, #f5af19, #f12711);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.user-name { color: #aaa; font-size: 14px; }

.logout-btn {
  padding: 6px 16px;
  border: 1px solid rgba(255,255,255,0.15);
  border-radius: 6px;
  background: transparent;
  color: #888;
  cursor: pointer;
  font-size: 13px;
}

.logout-btn:hover { color: #f12711; border-color: #f12711; }

.main-content {
  flex: 1;
  display: flex;
  justify-content: center;
  padding: 40px 20px;
}

.book-form {
  width: 100%;
  max-width: 680px;
}

.form-section {
  background: rgba(255,255,255,0.04);
  border: 1px solid rgba(255,255,255,0.06);
  border-radius: 12px;
  padding: 28px;
  margin-bottom: 20px;
}

.form-section h3 {
  font-size: 16px;
  color: #ccc;
  margin-bottom: 20px;
  padding-bottom: 12px;
  border-bottom: 1px solid rgba(255,255,255,0.06);
}

.form-group {
  margin-bottom: 18px;
}

.form-group:last-child { margin-bottom: 0; }

.form-group label {
  display: block;
  font-size: 13px;
  color: #999;
  margin-bottom: 6px;
}

.required { color: #f12711; }

.form-group input,
.form-group select,
.form-group textarea {
  width: 100%;
  padding: 10px 14px;
  background: rgba(255,255,255,0.06);
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 8px;
  color: #e0e0e0;
  font-size: 14px;
  font-family: inherit;
  outline: none;
  transition: border-color 0.2s;
}

.form-group input:focus,
.form-group select:focus,
.form-group textarea:focus {
  border-color: rgba(245,175,25,0.4);
}

.form-group select {
  cursor: pointer;
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%23888' d='M6 8L1 3h10z'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 12px center;
  padding-right: 36px;
}

.form-group select option {
  background: #1a1a2e;
  color: #e0e0e0;
}

.form-group textarea {
  resize: vertical;
  min-height: 120px;
  line-height: 1.6;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.hint {
  display: block;
  font-size: 12px;
  color: #666;
  margin-top: 4px;
}

.char-count {
  text-align: right;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 8px;
}

.btn-cancel {
  padding: 10px 24px;
  border: 1px solid rgba(255,255,255,0.15);
  border-radius: 8px;
  background: transparent;
  color: #888;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s;
}

.btn-cancel:hover { color: #ccc; border-color: rgba(255,255,255,0.3); }

.btn-submit {
  padding: 10px 32px;
  border: none;
  border-radius: 8px;
  background: linear-gradient(135deg, #f5af19, #f12711);
  color: #fff;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
  transition: opacity 0.2s;
}

.btn-submit:hover { opacity: 0.9; }
.btn-submit:disabled { opacity: 0.5; cursor: not-allowed; }

.error-msg {
  margin-top: 16px;
  padding: 12px 16px;
  background: rgba(241,39,17,0.1);
  border: 1px solid rgba(241,39,17,0.3);
  border-radius: 8px;
  color: #f12711;
  font-size: 13px;
  text-align: center;
}
</style>
