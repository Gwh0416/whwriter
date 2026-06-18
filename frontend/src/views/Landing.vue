<template>
  <div class="landing-page">
    <header class="nav">
      <router-link to="/" class="brand">
        <span class="brand-mark">文</span>
        <span>文豪写作</span>
      </router-link>
      <nav class="nav-links">
        <button :class="{ active: activeSlide === 0 }" @click="goSlide(0)">首页</button>
        <button :class="{ active: activeSlide === 1 }" @click="goSlide(1)">功能</button>
        <button :class="{ active: activeSlide === 2 }" @click="goSlide(2)">流程</button>
        <button :class="{ active: activeSlide === 3 }" @click="goSlide(3)">记忆文件</button>
      </nav>
      <div class="nav-actions">
        <router-link v-if="!token" to="/login" class="login-link">登录</router-link>
        <router-link v-if="!token" to="/login?mode=register" class="primary-link">注册使用</router-link>
        <router-link v-else :to="homePath" class="primary-link">进入工作台</router-link>
      </div>
    </header>

    <main>
      <section class="showcase" @wheel.prevent="handleShowcaseWheel">
        <button class="slide-arrow left" :disabled="activeSlide === 0" @click="prevSlide">‹</button>
        <div class="slide-window">
          <div class="slides" :style="{ transform: `translateX(-${activeSlide * 100}%)` }">
            <section class="slide hero">
              <div class="hero-copy">
                <div class="eyebrow">AI Long-form Writing Studio</div>
                <h1>把长篇小说创作，从灵感推进到可回滚的生产流程。</h1>
                <p>
                  文豪写作把规划、创作、审查、修订、润色和记忆文件结算串成一条可追踪的链路，
                  适合需要长期维护人物、伏笔、状态和世界观的小说项目。
                </p>
                <div class="hero-actions">
                  <router-link v-if="!token" to="/login?mode=register" class="cta">开始创作</router-link>
                  <router-link v-else :to="homePath" class="cta">进入工作台</router-link>
                  <router-link to="/login" class="ghost">已有账号登录</router-link>
                </div>
                <div class="hero-metrics">
                  <div><strong>10</strong><span>写作环节</span></div>
                  <div><strong>7+</strong><span>专业 Agent</span></div>
                  <div><strong>可回滚</strong><span>章节状态</span></div>
                </div>
              </div>
              <div class="hero-card">
                <div class="card-top">
                  <span></span><span></span><span></span>
                </div>
                <div class="pipeline">
                  <div class="pipeline-step active">规划师：规划本章</div>
                  <div class="pipeline-step active">写手：创作正文</div>
                  <div class="pipeline-step">审稿官：审查结构</div>
                  <div class="pipeline-step">结算官：结算记忆</div>
                </div>
                <div class="state-card">
                  <div class="state-title">当前状态卡</div>
                  <p>地点：洛府东厢偏房</p>
                  <p>目标：查明灵根与戒指的关联</p>
                  <p>冲突：家族资源被重新分配</p>
                </div>
              </div>
            </section>

            <section class="slide feature-slide">
              <div class="feature-hero">
                <div>
                  <span class="section-kicker">核心能力</span>
                  <h2 class="feature-title">不只是生成正文，更是在管理长篇小说的运行状态</h2>
                  <p class="section-text feature-text-nowrap">
                    文豪写作把章节规划、上下文编排、正文生成、连续性审计和记忆结算放进同一条链路里，让长篇创作不再只依赖一段临时 prompt
                  </p>
                </div>
                <!-- <div class="feature-example">
                  <span>示例任务</span>
                  <strong>“下一章重点写师徒矛盾，但不要提前揭露功法秘密。”</strong>
                  <p>系统会把这句话转成章节目标、避让项、上下文选择和写后状态更新，而不是只把它拼进正文提示词。</p>
                </div> -->
              </div>
              <div class="feature-body">
                <div class="feature-grid">
                  <article v-for="item in features" :key="item.title" class="feature-card">
                    <div class="feature-card-head">
                      <div class="feature-icon">{{ item.icon }}</div>
                      <h3>{{ item.title }}</h3>
                    </div>
                    <p>{{ item.desc }}</p>
                  </article>
                </div>
              </div>
              <div class="feature-flow">
                <div v-for="item in featureFlow" :key="item.title" class="feature-flow-item">
                  <span>{{ item.step }}</span>
                  <strong>{{ item.title }}</strong>
                  <p>{{ item.desc }}</p>
                </div>
              </div>
            </section>

            <section class="slide split workflow-slide">
              <div class="workflow-main">
                <span class="section-kicker">Workflow</span>
                <h2 class="workflow-title">从用户指令到章节生成，每一步都有明确职责</h2>
                <p class="section-text">
                  系统会先整理上下文，再规划本章，随后生成正文，并通过审查、修订、润色和记忆文件结算把章节接回全书状态
                </p>
                <div class="workflow-stats">
                  <div v-for="item in workflowStats" :key="item.title" class="workflow-stat-card">
                    <strong>{{ item.value }}</strong>
                    <h3>{{ item.title }}</h3>
                    <p>{{ item.desc }}</p>
                  </div>
                </div>
                <div class="agent-strip">
                  <div v-for="agent in agents" :key="agent.name" class="agent-chip">
                    <strong>{{ agent.name }}</strong>
                    <span>{{ agent.desc }}</span>
                    <em>{{ agent.example }}</em>
                  </div>
                </div>
              </div>
              <div class="workflow-list">
                <div v-for="(step, index) in workflow" :key="step" class="workflow-item">
                  <span>{{ String(index + 1).padStart(2, '0') }}</span>
                  <p>{{ step }}</p>
                </div>
              </div>
            </section>

            <section class="slide">
              <div class="section-head section-head-wide">
                <span>Memory Files</span>
                <h2 class="section-title-desktop-nowrap">人物、设定、伏笔、状态卡统一沉淀</h2>
                <p class="section-text">
                  记忆文件不是把所有细节塞进一张表，而是区分当前状态、长期设定、证据笔记和章节快照，让模型取用的是“当前有效视图”
                </p>
              </div>
              <div class="memory-tabs">
                <button
                  v-for="item in memorySections"
                  :key="item.key"
                  type="button"
                  class="memory-tab"
                  :class="{ active: activeMemoryTab === item.key }"
                  @click="activeMemoryTab = item.key"
                >
                  <span>{{ item.icon }}</span>
                  <strong>{{ item.title }}</strong>
                  <em>{{ item.count }}</em>
                </button>
              </div>
              <div class="memory-section-intro">
                <strong>{{ currentMemorySection.title }}</strong>
                <p>{{ currentMemorySection.intro }}</p>
              </div>
              <div class="memory-file-list">
                <div v-for="item in currentMemorySection.files" :key="item.title" class="memory-file-card">
                  <div class="memory-file-head">
                    <span>{{ item.icon }}</span>
                    <h3>{{ item.title }}</h3>
                  </div>
                  <p>{{ item.desc }}</p>
                  <span class="panel-example">{{ item.example }}</span>
                </div>
              </div>
            </section>
          </div>
        </div>
        <button class="slide-arrow right" :disabled="activeSlide === slides.length - 1" @click="nextSlide">›</button>
        <div class="slide-dots">
          <button v-for="(slide, index) in slides" :key="slide" :class="{ active: activeSlide === index }" @click="goSlide(index)">
            {{ slide }}
          </button>
        </div>
      </section>
    </main>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'

const token = localStorage.getItem('token')
const role = localStorage.getItem('role')
const homePath = computed(() => (role === 'admin' ? '/admin' : '/write'))
const activeSlide = ref(0)
const activeMemoryTab = ref('foundations')
const slides = ['首页', '功能', '流程', '记忆文件']
let lastWheelAt = 0

const features = [
  { icon: '📚', title: '长篇上下文编排', desc: '自动整理设定、人物、伏笔和最近章节摘要，减少前后文断裂。' },
  { icon: '🧭', title: '多 Agent 写作链路', desc: '规划、创作、审查、修订、润色和结算分工明确，方便定位问题。' },
  { icon: '🗂️', title: '记忆文件系统', desc: '把当前状态、长期设定、证据笔记和章节快照分层保存。' },
  { icon: '🏷️', title: '自定义题材', desc: '用户可自定义题材画像，创建书籍时直接套用。' },
  { icon: '🤖', title: '多模型支持', desc: '支持多个热门大模型，并可按书选择具体写作模型。' },
  { icon: '📰', title: '多平台风格支持', desc: '可按目标平台切换风格约束，让节奏和表达更贴近不同平台读者。' },
  { icon: '⏯️', title: '写作运行控制', desc: '写作 run 失败后可取消、从当前节点重试，灵活控制。' },
    { icon: '↩️', title: '章节回滚与重写', desc: '支持删除或重写最后一章，并把相关状态恢复到正确快照。' },
  { icon: '📦', title: '书籍导出', desc: '支持多种格式书籍导出，方便整理、备份和后续发布。' },
]

const featureFlow = [
  { step: '01', title: '输入目标', desc: '用户给出本章方向、限制或重写要求。' },
  { step: '02', title: '编排上下文', desc: '系统选择相关人物、状态、伏笔和近期摘要。' },
  { step: '03', title: '生成与审查', desc: '写手产出正文，审稿官检查结构和连续性。' },
  { step: '04', title: '结算记忆', desc: '章节通过后更新状态卡、摘要、伏笔和快照。' },
]

const workflow = [
  '加载书籍、章节历史和当前状态，准备本章写作上下文。',
  '整理作者意图、当前焦点和用户本次写作要求。',
  '筛选本章相关人物、状态、伏笔和近期摘要。',
  '规划师生成章节备忘，明确本章目标、禁区和推进点。',
  '写手根据章节备忘和上下文创作章节正文。',
  '审稿官检查结构、节奏、设定和状态一致性。',
  '修订师根据问题回改单章逻辑与承接。',
  '润色师优化表达、节奏和正文可读性。',
  '结算官与记忆提取器更新状态卡、摘要和伏笔变化。',
  '统一保存章节快照，支持失败取消、重试和最后一章重写。',
]

const agents = [
  { name: '规划师', desc: '规划本章目标', example: '例：必须推进师徒冲突，不能提前泄露戒指秘密。' },
  { name: '写手', desc: '生成章节正文', example: '例：把章节备忘展开成场景、对话和动作推进。' },
  { name: '审稿官', desc: '审查结构一致性', example: '例：指出人物动机跳变、资源前后不一致。' },
  { name: '修订师', desc: '按问题修订', example: '例：补回断掉的伏笔承接，修正冲突逻辑。' },
  { name: '润色师', desc: '优化表达节奏', example: '例：压掉总结腔，把句子改得更像正文。' },
  { name: '结算官', desc: '结算记忆变化', example: '例：更新状态卡、章节摘要、伏笔和章节快照。' },
]

const workflowStats = [
  { value: '10 步', title: '完整写作流程', desc: '从上下文加载到章节快照，前端按完整链路展示每一步。' },
  { value: '6 位', title: '核心 Agent', desc: '规划、写作、审查、修订、润色、结算 6 位 Agent 各司其职。' },
  { value: '可取消 / 重试', title: '失败处理', desc: '某一步失败后可取消本次写作，也能从失败处继续或重写最后一章。' },
]

const memorySections = [
  {
    key: 'state',
    icon: '🧭',
    title: '当前状态',
    count: 1,
    intro: '当前状态只保留“此刻对下一章最重要”的即时信息，避免和长期设定混在一起。',
    files: [
      { icon: '📍', title: '状态卡', desc: '记录当前位置、当前目标、限制、冲突和近期变化，供下一章直接承接。', example: '例如：当前位置在洛府东厢偏房，短期目标是查清戒指与灵根的关系。' },
    ],
  },
  {
    key: 'characters',
    icon: '👤',
    title: '人物',
    count: 3,
    intro: '人物相关文件会持续合并更新，避免角色设定、关系和成长线越写越乱。',
    files: [
      { icon: '🧑', title: '人物卡', desc: '按角色名维护身份、动机、关系和成长阶段。', example: '例如：林烬突破到炼气七层，与师门信任继续下降。' },
      { icon: '🕸️', title: '关系线', desc: '跟踪人物之间的信任、敌意、盟友关系和信息边界。', example: '例如：主角知道洛红袖在试探自己，但对方仍不知戒指真相。' },
      { icon: '💬', title: '情绪变化', desc: '沉淀关键人物的情绪波动和阶段状态，辅助后续承接。', example: '例如：本章后主角由压抑转为主动试探。' },
    ],
  },
  {
    key: 'facts',
    icon: '📋',
    title: '设定',
    count: 5,
    intro: '设定类文件沉淀长期稳定的信息，比如资源、规则、物品和世界背景，不会被一章里的短期状态覆盖。',
    files: [
      { icon: '📦', title: '资源账本', desc: '记录灵石、法器、丹药、材料等资源变化。', example: '例如：本章后灵石余量减少，新增一枚残缺令牌。' },
      { icon: '📜', title: '世界规则', desc: '记录修炼体系、家族规矩、宗门禁令和能力边界。', example: '例如：洛府内院夜禁仍然有效，外门弟子不能私入藏经阁。' },
      { icon: '🏛️', title: '势力与地点', desc: '整理家族、宗门、区域和关键地点信息，维持世界结构稳定。', example: '例如：洛府内外院权限不同，东厢偏房属于旁支居所。' },
      { icon: '🧩', title: '已知信息', desc: '汇总当前已经确认的设定信息，避免后续重复揭示或前后矛盾。', example: '例如：戒指确实与主角灵根异常有关，但来源尚未公开。' },
      { icon: '📚', title: '章节设定沉淀', desc: '把一章里真正落定的长期信息提炼出来，进入全书设定。', example: '例如：主角确认账本被人动过手脚，这一点进入长期设定。' },
    ],
  },
  {
    key: 'hooks',
    icon: '🪝',
    title: '伏笔',
    count: 4,
    intro: '伏笔区负责保存承诺给读者、但还没解决的问题，方便持续追踪和回收。',
    files: [
      { icon: '🎣', title: '开放伏笔', desc: '记录尚未回收的关键线索、悬念和承诺。', example: '例如：戒指来源、账本缺页背后的人、师父真实立场。' },
      { icon: '⏳', title: '推进状态', desc: '标记每个伏笔目前是刚埋下、推进中还是接近回收。', example: '例如：账本缺页线从“埋下”推进到“主角开始主动调查”。' },
      { icon: '🔗', title: '关联人物', desc: '把伏笔和涉及人物、地点、资源关联起来，方便后续承接。', example: '例如：戒指线关联主角、洛红袖、洛府旧藏。' },
      { icon: '✅', title: '已回收记录', desc: '保留已经回收的伏笔结果，避免后面重复再用。', example: '例如：家族试探主角身份的伏笔已在前一卷收束。' },
    ],
  },
  {
    key: 'summaries',
    icon: '📝',
    title: '章节摘要',
    count: 1,
    intro: '章节摘要用来快速回顾上一章发生了什么，以及它给下一章留下了哪些接力点。',
    files: [
      { icon: '🗒️', title: '章节摘要文件', desc: '汇总每章的关键事件、人物出场、状态变化和悬念尾钩。', example: '例如：第 12 章结束时主角发现账本异常，并决定夜探藏经阁。' },
    ],
  },
  {
    key: 'foundations',
    icon: '📐',
    title: '基础文件',
    count: 6,
    intro: '基础文件决定这本书长期往哪里走、最近几章该盯什么，以及要遵守哪些写作护栏。',
    files: [
      { icon: '🎯', title: '作者意图', desc: '长期定义这本书想成为什么，约束主线方向、情绪基调和不该偏掉的核心冲突。', example: '例如：这本书始终要围绕师徒裂痕和功法秘密展开。' },
      { icon: '📏', title: '创作规则', desc: '把题材规则、平台节奏、禁区和长期写作约束收进规则文件里，避免写着写着跑偏。', example: '例如：前期必须持续给反馈，不能突然转成纯说明文叙事。' },
      { icon: '🔍', title: '当前焦点', desc: '只管最近 1 到 3 章该把注意力拉回哪里，用来纠正近期章节的推进方向。', example: '例如：最近两章重点拉回洛府线，先别继续扩新地图。' },
      { icon: '🏗️', title: '故事框架', desc: '记录整本书的故事骨架，包括主线冲突、卷结构和长期推进节奏。', example: '例如：第一卷先完成家族压迫与主角反击的起盘。' },
      { icon: '🎨', title: '风格指南', desc: '沉淀这本书的表达偏好、叙事习惯和平台风格，让后续章节文风更稳定。', example: '例如：保持网文推进感，多场景对话，少总结腔。' },
      { icon: '🗺️', title: '卷地图', desc: '把每一卷的阶段目标、关键节点和承接关系整理出来，方便后续章节规划。', example: '例如：卷一收家族线，卷二正式进入宗门与外部势力冲突。' },
    ],
  },
  {
    key: 'snapshots',
    icon: '📸',
    title: '快照',
    count: 2,
    intro: '快照负责在章节提交时保存一份可恢复视图，删除或重写最后一章时就靠它回滚。',
    files: [
      { icon: '📷', title: '章节快照', desc: '每章提交后保存人物、状态、设定、伏笔等关键数据的快照。', example: '例如：重写第 12 章前，先恢复第 11 章提交后的整套状态。' },
      { icon: '↩️', title: '回滚恢复', desc: '删除或重写最后一章时同步恢复基础文件和记忆状态。', example: '例如：删掉最新章后，人物关系、当前焦点和伏笔进度一起回退。' },
    ],
  },
]

const currentMemorySection = computed(
  () => memorySections.find((item) => item.key === activeMemoryTab.value) || memorySections[0],
)

function goSlide(index) {
  activeSlide.value = Math.min(Math.max(index, 0), slides.length - 1)
}

function prevSlide() {
  goSlide(activeSlide.value - 1)
}

function nextSlide() {
  goSlide(activeSlide.value + 1)
}

function handleShowcaseWheel(event) {
  const now = Date.now()
  if (now - lastWheelAt < 520 || Math.abs(event.deltaY) < 12) return
  lastWheelAt = now
  if (event.deltaY > 0) {
    nextSlide()
  } else {
    prevSlide()
  }
}
</script>

<style scoped>
.landing-page {
  min-height: 100vh;
  background: #f8fafc;
  color: #0f172a;
}
.nav {
  position: sticky;
  top: 0;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 18px 6vw;
  background: rgba(248, 250, 252, 0.9);
  backdrop-filter: blur(16px);
  border-bottom: 1px solid rgba(148, 163, 184, 0.2);
}
.brand,
.nav a,
.nav button {
  text-decoration: none;
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #0f172a;
  font-weight: 800;
  font-size: 18px;
}
.brand-mark {
  width: 34px;
  height: 34px;
  display: grid;
  place-items: center;
  border-radius: 10px;
  background: linear-gradient(135deg, #2563eb, #7c3aed);
  color: #fff;
}
.nav-links {
  display: flex;
  gap: 10px;
}
.nav-links button,
.login-link {
  border: none;
  background: transparent;
  color: #475569;
  font-size: 14px;
  cursor: pointer;
}
.nav-links button {
  padding: 9px 14px;
  border-radius: 999px;
  transition: all 0.2s ease;
}
.nav-links button.active,
.nav-links button:hover {
  background: #e0e7ff;
  color: #1d4ed8;
}
.nav-actions {
  display: flex;
  align-items: center;
  gap: 14px;
}
.primary-link,
.cta {
  border-radius: 999px;
  background: #2563eb;
  color: #fff;
  text-decoration: none;
  font-weight: 700;
}
.primary-link {
  padding: 10px 18px;
  font-size: 14px;
}
.hero {
  display: grid;
  grid-template-columns: minmax(0, 1.05fr) minmax(360px, 0.95fr);
  gap: 52px;
  align-items: center;
  padding: 86px 6vw 72px;
}
.eyebrow,
.section-head span,
.section-kicker {
  color: #2563eb;
  font-size: 13px;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}
.hero h1 {
  margin: 18px 0;
  font-size: clamp(40px, 6vw, 72px);
  line-height: 1.04;
  letter-spacing: -0.05em;
}
.hero p,
.section-text {
  color: #475569;
  font-size: 17px;
  line-height: 1.8;
}
.hero-actions {
  display: flex;
  gap: 14px;
  margin-top: 32px;
}
.cta {
  padding: 14px 24px;
}
.ghost {
  padding: 14px 22px;
  border-radius: 999px;
  color: #1e293b;
  background: #fff;
  border: 1px solid #e2e8f0;
  text-decoration: none;
  font-weight: 700;
}
.hero-metrics {
  display: flex;
  gap: 28px;
  margin-top: 38px;
}
.hero-metrics div {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.hero-metrics strong {
  font-size: 22px;
}
.hero-metrics span {
  color: #64748b;
  font-size: 13px;
}
.hero-card {
  padding: 22px;
  border-radius: 28px;
  background: #0f172a;
  box-shadow: 0 30px 80px rgba(15, 23, 42, 0.24);
}
.card-top {
  display: flex;
  gap: 8px;
  margin-bottom: 20px;
}
.card-top span {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #64748b;
}
.pipeline {
  display: grid;
  gap: 12px;
}
.pipeline-step,
.state-card {
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.08);
  color: #cbd5e1;
}
.pipeline-step {
  padding: 16px;
}
.pipeline-step.active {
  background: rgba(37, 99, 235, 0.24);
  color: #fff;
}
.state-card {
  margin-top: 20px;
  padding: 18px;
}
.state-title {
  color: #fff;
  font-weight: 800;
  margin-bottom: 10px;
}
.state-card p {
  margin: 7px 0;
  color: #cbd5e1;
  font-size: 13px;
}
.showcase {
  position: relative;
  height: calc(100vh - 77px);
  min-height: calc(100vh - 77px);
  padding: 34px 6vw 82px;
  overflow: hidden;
  background: linear-gradient(180deg, #f8fafc 0%, #eef2ff 100%);
  box-sizing: border-box;
}
.slide-window {
  height: 100%;
  overflow: hidden;
  border-radius: 34px;
}
.slides {
  display: flex;
  height: 100%;
  transition: transform 0.55s cubic-bezier(0.22, 1, 0.36, 1);
  will-change: transform;
}
.slide {
  flex: 0 0 100%;
  height: 100%;
  min-height: 100%;
  padding: 56px;
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 34px;
  background: rgba(255, 255, 255, 0.78);
  box-shadow: 0 24px 70px rgba(15, 23, 42, 0.08);
  box-sizing: border-box;
}
.section-head {
  max-width: 760px;
  margin-bottom: 30px;
}
.section-head-wide {
  max-width: none;
}
.section-title-desktop-nowrap {
  white-space: nowrap;
}
.slide h2 {
  margin: 12px 0 0;
  font-size: clamp(30px, 4vw, 48px);
  line-height: 1.12;
  letter-spacing: -0.04em;
}
.feature-slide {
  display: flex;
  flex-direction: column;
  gap: 22px;
}
.feature-hero {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(320px, 0.56fr);
  gap: 28px;
  align-items: start;
}
.feature-title {
  white-space: nowrap;
}
.feature-text-nowrap {
  white-space: nowrap;
}
.feature-hero .section-text {
  max-width: 860px;
  margin-bottom: 0;
}
.feature-example,
.feature-side,
.feature-flow-item {
  border: 1px solid #e2e8f0;
  background: #fff;
  box-shadow: 0 14px 36px rgba(15, 23, 42, 0.05);
}
.feature-example {
  padding: 22px;
  border-radius: 24px;
}
.feature-example span,
.feature-flow-item span {
  color: #2563eb;
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.08em;
}
.feature-example strong {
  display: block;
  margin: 10px 0;
  color: #0f172a;
  font-size: 19px;
  line-height: 1.5;
}
.feature-example p,
.feature-flow-item p {
  margin: 0;
  color: #64748b;
  line-height: 1.7;
}
.feature-body {
  display: block;
  flex: 1;
}
.feature-body .feature-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}
.feature-grid,
.truth-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 18px;
}
.feature-card,
.truth-panel {
  padding: 20px;
  border-radius: 22px;
  background: #fff;
  border: 1px solid #e2e8f0;
  box-shadow: 0 16px 40px rgba(15, 23, 42, 0.05);
}
.feature-card-head {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 10px;
}
.feature-icon {
  font-size: 30px;
  line-height: 1;
}
.feature-card h3,
.truth-panel h3 {
  margin: 0;
}
.feature-card p,
.truth-panel p {
  margin: 0;
  color: #64748b;
  line-height: 1.7;
}
.feature-flow {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
}
.feature-flow-item {
  padding: 20px;
  border-radius: 22px;
}
.feature-flow-item strong {
  display: block;
  margin: 9px 0 7px;
  color: #0f172a;
}
.detail-grid {
  display: grid;
  grid-template-columns: 1.1fr 0.9fr;
  gap: 18px;
  margin-top: 22px;
}
.detail-panel,
.workflow-note,
.agent-chip,
.memory-column {
  border: 1px solid #e2e8f0;
  border-radius: 20px;
  background: #fff;
  box-shadow: 0 14px 36px rgba(15, 23, 42, 0.05);
}
.detail-panel {
  padding: 24px;
}
.detail-panel h3,
.workflow-note strong,
.memory-column h3 {
  margin: 0 0 12px;
  color: #0f172a;
}
.detail-panel ul {
  margin: 0;
  padding-left: 18px;
  color: #475569;
  line-height: 1.9;
}
.detail-panel.dark {
  background: #0f172a;
  color: #e2e8f0;
}
.detail-panel.dark h3 {
  color: #fff;
}
.mini-metrics {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}
.mini-metrics div {
  padding: 14px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.08);
}
.mini-metrics strong,
.mini-metrics span {
  display: block;
}
.mini-metrics span {
  margin-top: 6px;
  color: #cbd5e1;
  font-size: 12px;
  line-height: 1.5;
}
.split {
  display: grid;
  grid-template-columns: minmax(0, 1.18fr) minmax(420px, 0.82fr);
  column-gap: 62px;
  row-gap: 16px;
  align-items: start;
}
.workflow-slide {
  padding-bottom: 32px;
}
.workflow-main {
  width: 100%;
}
.workflow-title {
  white-space: nowrap;
}
.workflow-note {
  margin-top: 26px;
  padding: 20px;
}
.workflow-note span {
  display: block;
  color: #64748b;
  line-height: 1.7;
}
.workflow-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(175px, 1fr));
  gap: 12px;
  margin-top: 12px;
}
.workflow-stat-card,
.memory-rail-item {
  border: 1px solid #e2e8f0;
  border-radius: 20px;
  background: #fff;
  box-shadow: 0 14px 36px rgba(15, 23, 42, 0.05);
}
.workflow-stat-card {
  padding: 18px;
}
.workflow-stat-card strong {
  color: #2563eb;
  font-size: 24px;
  font-weight: 900;
}
.workflow-stat-card h3 {
  margin: 10px 0 8px;
  font-size: 18px;
}
.workflow-stat-card p {
  margin: 0;
  color: #64748b;
  line-height: 1.7;
}
.workflow-list {
  display: grid;
  gap: 9px;
  padding-left: 56px;
  margin-top: 22px;
  align-self: center;
}
.workflow-item {
  display: grid;
  grid-template-columns: 50px 1fr;
  gap: 12px;
  align-items: center;
  padding: 14px 16px;
  border-radius: 18px;
  background: #fff;
}
.workflow-item span {
  color: #2563eb;
  font-weight: 900;
}
.workflow-item p {
  margin: 0;
  color: #334155;
  line-height: 1.45;
}
.agent-strip {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  margin-top: 14px;
}
.agent-chip {
  padding: 18px;
}
.agent-chip strong,
.agent-chip span,
.agent-chip em {
  display: block;
}
.agent-chip strong {
  color: #2563eb;
}
.agent-chip span {
  margin-top: 6px;
  color: #64748b;
  font-size: 13px;
}
.agent-chip em,
.panel-example {
  display: block;
  margin-top: 10px;
  color: #475569;
  font-size: 12px;
  line-height: 1.6;
  font-style: normal;
}
.memory-tabs {
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 18px;
}
.memory-tab {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px 14px;
  border: 1px solid #e2e8f0;
  border-radius: 16px;
  background: #fff;
  color: #64748b;
  box-shadow: 0 14px 36px rgba(15, 23, 42, 0.05);
}
.memory-tab.active {
  background: #2563eb;
  border-color: #2563eb;
  color: #fff;
}
.memory-tab strong,
.memory-tab em {
  font-style: normal;
  font-weight: 700;
}
.memory-file-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}
.memory-file-card {
  padding: 22px;
  border: 1px solid #e2e8f0;
  border-radius: 20px;
  background: #fff;
  box-shadow: 0 14px 36px rgba(15, 23, 42, 0.05);
}
.memory-file-head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}
.memory-file-head h3 {
  margin: 0;
  color: #0f172a;
}
.memory-file-card p {
  margin: 0;
  color: #64748b;
  line-height: 1.75;
}
.slide-arrow {
  position: absolute;
  top: 50%;
  z-index: 3;
  width: 52px;
  height: 52px;
  border: 1px solid #dbeafe;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.9);
  color: #2563eb;
  font-size: 38px;
  line-height: 1;
  cursor: pointer;
  box-shadow: 0 12px 30px rgba(37, 99, 235, 0.16);
  transform: translateY(-50%);
}
.slide-arrow.left {
  left: 2vw;
}
.slide-arrow.right {
  right: 2vw;
}
.slide-arrow:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}
.slide-dots {
  position: absolute;
  left: 50%;
  bottom: 28px;
  display: flex;
  gap: 10px;
  transform: translateX(-50%);
}
.slide-dots button {
  border: none;
  border-radius: 999px;
  padding: 9px 16px;
  background: #fff;
  color: #64748b;
  cursor: pointer;
  box-shadow: 0 10px 26px rgba(15, 23, 42, 0.08);
}
.slide-dots button.active {
  background: #2563eb;
  color: #fff;
}
@media (max-width: 1280px) {
  .feature-title,
  .feature-text-nowrap,
  .workflow-title,
  .section-title-desktop-nowrap {
    white-space: normal;
  }
  .feature-body .feature-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .feature-flow {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .memory-tabs {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}
@media (max-width: 1100px) {
  .hero,
  .split,
  .feature-hero {
    grid-template-columns: 1fr;
  }
  .feature-grid,
  .workflow-stats,
  .memory-file-list {
    grid-template-columns: 1fr 1fr;
  }
  .workflow-list {
    padding-left: 0;
    margin-top: 8px;
  }
  .agent-strip {
    grid-template-columns: 1fr 1fr;
  }
}
@media (max-width: 900px) {
  .section-title-desktop-nowrap {
    white-space: normal;
  }
  .feature-title {
    white-space: normal;
  }
  .feature-text-nowrap {
    white-space: normal;
  }
  .workflow-title {
    white-space: normal;
  }
  .nav-links {
    display: none;
  }
  .hero,
  .split {
    grid-template-columns: 1fr;
  }
  .feature-hero,
  .feature-body,
  .feature-flow,
  .feature-grid,
  .workflow-stats,
  .memory-tabs,
  .truth-grid,
  .detail-grid,
  .agent-strip,
  .memory-file-list {
    grid-template-columns: 1fr;
  }
  .showcase {
    padding: 30px 4vw 90px;
  }
  .slide {
    min-height: auto;
    padding: 28px;
  }
  .slide-arrow {
    display: none;
  }
}
</style>
