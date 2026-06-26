<template>
  <div class="write-container">
    <aside class="sidebar">
      <div class="sidebar-logo">
        <h2>文豪写作</h2>
      </div>
      <nav class="sidebar-nav">
        <a href="#" class="nav-item" :class="{ active: tab === 'books' }" @click.prevent="switchTab('books')">书籍管理</a>
        <a href="#" class="nav-item" :class="{ active: tab === 'genres' }" @click.prevent="switchTab('genres')">题材管理</a>
        <a href="#" class="nav-item" :class="{ active: tab === 'settings' }" @click.prevent="switchTab('settings')">个人设置</a>
        <a href="#" class="nav-item" :class="{ active: tab === 'conversations' }" @click.prevent="switchTab('conversations')">对话管理</a>
      </nav>
      <div class="sidebar-footer">
        <div class="user-info">
          <div class="avatar">{{ user?.username?.[0] }}</div>
          <div>
            <div class="user-name">{{ user?.username }}</div>
          </div>
        </div>
        <button class="logout-btn" @click="logout">退出</button>
      </div>
    </aside>
    <main class="main-content">
      <header class="topbar">
        <h1>
          <template v-if="tab === 'books' && selectedBook">
            <a href="#" class="back-link" @click.prevent="leaveBookView()">← 返回</a>
            {{ selectedBook.title }}
          </template>
          <template v-else>{{ tabTitle }}</template>
        </h1>
        <div class="topbar-actions">
          <router-link v-if="tab === 'books' && !selectedBook" to="/create-book" class="create-btn">+ 创建新小说</router-link>
        </div>
      </header>
      <div class="content">

        <div v-if="tab === 'books' && !selectedBook">
          <div v-if="books.length === 0" class="empty-state">
            <div class="empty-icon">✍️</div>
            <h2>开始你的创作之旅</h2>
            <p>AI 驱动的智能写作助手，帮你完成从构思到成稿的全流程</p>
            <div class="empty-actions">
              <router-link to="/create-book" class="welcome-btn primary">创建新小说</router-link>
            </div>
          </div>
          <div v-else class="data-section">
            <div class="section-header">
              <div class="section-count">共 {{ books.length }} 本小说</div>
            </div>
            <div class="table-wrap">
              <table class="data-table">
                <thead>
                  <tr>
                    <th>书名</th>
                    <th>题材</th>
                    <th>平台</th>
                    <th>字数/章</th>
                    <th>目标章数</th>
                    <th>状态</th>
                    <th>创建时间</th>
                    <th>操作</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="b in books" :key="b.id">
                    <td class="book-title">{{ b.title }}</td>
                    <td>{{ b.genre?.name || '-' }}</td>
                    <td>{{ b.platform?.name || '-' }}</td>
                    <td>{{ b.chapter_word_count }}</td>
                    <td>{{ b.target_chapters }}</td>
                    <td>
                      <span class="status-tag" :class="b.status">{{ statusLabel(b.status) }}</span>
                    </td>
                    <td>{{ formatDate(b.created_at) }}</td>
                    <td class="table-actions">
                      <button class="action-btn" @click="openBook(b.id)">进入</button>
                      <button class="action-btn danger" :disabled="deletingBookID === b.id" @click="deleteBook(b, $event)">
                        {{ deletingBookID === b.id ? '删除中...' : '删除' }}
                      </button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>

        <div v-if="tab === 'books' && selectedBook" class="book-detail">
          <div class="book-meta">
            <div class="meta-item">
              <span class="meta-label">题材</span>
              <span class="meta-value">{{ selectedBook.genre?.name }}</span>
            </div>
            <div class="meta-item">
              <span class="meta-label">平台</span>
              <span class="meta-value">{{ selectedBook.platform?.name }}</span>
            </div>
            <div class="meta-item">
              <span class="meta-label">状态</span>
              <span class="status-tag" :class="selectedBook.status">{{ statusLabel(selectedBook.status) }}</span>
            </div>
            <div class="meta-item">
              <span class="meta-label">章节</span>
              <span class="meta-value">{{ chapters.length }} / {{ selectedBook.target_chapters }}</span>
            </div>
            <div class="book-meta-actions">
              <button class="action-btn" :disabled="exporting" @click="exportBook('txt')">导出 TXT</button>
              <button class="action-btn" :disabled="exporting" @click="exportBook('md')">导出 MD</button>
              <button class="action-btn danger" :disabled="deletingBookID === selectedBook.id" @click="deleteBook(selectedBook)">
                {{ deletingBookID === selectedBook.id ? '删除中...' : '删除本书' }}
              </button>
            </div>
          </div>

          <div class="book-tabs">
            <button class="book-tab" :class="{ active: bookTab === 'write' }" @click="bookTab = 'write'">写作</button>
            <button class="book-tab" :class="{ active: bookTab === 'truth' }" @click="bookTab = 'truth'; loadTruthFiles()">真相文件</button>
          </div>

          <div v-if="bookTab === 'write'">
            <div class="write-panel">
              <div class="write-controls">
                <div class="control-group">
                  <label>写作模型</label>
                  <select v-model="writeModelID" :disabled="writing">
                    <option :value="0">默认</option>
                    <optgroup v-for="cfg in llmConfigs" :key="cfg.id" :label="cfg.label">
                      <option v-for="m in cfg.models" :key="m.id" :value="m.id" :disabled="!m.is_enabled">
                        {{ m.model_name }}{{ m.is_default ? ' (默认)' : '' }}
                      </option>
                    </optgroup>
                  </select>
                </div>
                <div class="control-group">
                  <label>写作指令（可选）</label>
                  <input v-model="writeInput" placeholder="如：本章要推进主角升级到金丹期" :disabled="writing" @keyup.enter="writeChapter" />
                </div>
                <button class="write-btn" @click="writeChapter" :disabled="!canWrite">
                  {{ writing ? 'AI 正在创作...' : (selectedBook?.status === 'initializing' ? '初始化中...' : selectedBook?.status === 'writing' ? '写作中...' : '写下一章') }}
                </button>
              </div>
              <div v-if="writeDisabledReason" class="write-hint">{{ writeDisabledReason }}</div>
            </div>

            <div class="chapters-section">
              <h3>章节列表</h3>
              <div v-if="chapters.length === 0" class="no-chapters">还没有章节，点击「写下一章」开始创作</div>
              <div v-else class="chapter-list">
                <div v-for="ch in chapters" :key="ch.id" class="chapter-item" @click="viewChapter(ch)">
                  <div class="chapter-num">第{{ ch.chapter_number }}章</div>
                  <div class="chapter-title">{{ ch.title || '未命名' }}</div>
                  <div class="chapter-meta">
                    <span>{{ ch.word_count }} 字</span>
                    <span class="chapter-status" :class="ch.status">{{ ch.status }}</span>
                  </div>
                  <div v-if="isLatestChapter(ch)" class="chapter-actions">
                    <button class="action-btn sm" :disabled="writing" @click.stop="rewriteLatestChapter(ch)">
                      重写本章
                    </button>
                    <button class="action-btn danger sm" :disabled="deletingChapterNumber === ch.chapter_number" @click.stop="deleteChapter(ch)">
                      {{ deletingChapterNumber === ch.chapter_number ? '删除中...' : '删除本章' }}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div v-if="bookTab === 'truth'" class="truth-section">
            <div class="truth-loading" v-if="truthLoading">加载中...</div>
            <div v-else>
              <div class="truth-tabs">
                <button v-for="tt in truthTabs" :key="tt.key" class="truth-tab" :class="{ active: truthSubTab === tt.key }" @click="truthSubTab = tt.key">
                  <span class="truth-tab-icon">{{ tt.icon }}</span>
                  {{ tt.label }}
                  <span class="truth-count" v-if="truthCounts[tt.key] > 0">{{ truthCounts[tt.key] }}</span>
                </button>
              </div>

              <div v-if="truthSubTab === 'state'" class="truth-panel">
                <div v-if="!truthData.book_state" class="no-data">
                  <div class="no-data-icon">🧭</div>
                  <p>暂无当前状态</p>
                  <span>初始化完成或写作推进后，这里会显示可直接驱动写作的状态卡。</span>
                </div>
                <div v-else class="truth-card state-card">
                  <div class="state-head">
                    <div>
                      <div class="state-title">当前状态卡</div>
                      <div class="state-subtitle">来源章节：第{{ truthData.book_state.source_chapter || 0 }}章</div>
                    </div>
                    <span class="char-role protagonist" v-if="truthData.book_state.protagonist_name">{{ truthData.book_state.protagonist_name }}</span>
                  </div>
                  <div class="state-summary" v-if="truthData.book_state.situation_summary">{{ truthData.book_state.situation_summary }}</div>
                  <div class="state-grid">
                    <div class="state-item" v-if="truthData.book_state.current_location">
                      <span class="state-label">当前位置</span>
                      <span class="state-value">{{ truthData.book_state.current_location }}</span>
                    </div>
                    <div class="state-item" v-if="truthData.book_state.protagonist_state">
                      <span class="state-label">主角状态</span>
                      <span class="state-value">{{ truthData.book_state.protagonist_state }}</span>
                    </div>
                    <div class="state-item" v-if="truthData.book_state.current_goal">
                      <span class="state-label">当前目标</span>
                      <span class="state-value">{{ truthData.book_state.current_goal }}</span>
                    </div>
                    <div class="state-item" v-if="truthData.book_state.current_constraint">
                      <span class="state-label">当前限制</span>
                      <span class="state-value">{{ truthData.book_state.current_constraint }}</span>
                    </div>
                    <div class="state-item" v-if="truthData.book_state.current_alliances">
                      <span class="state-label">当前敌我</span>
                      <span class="state-value">{{ truthData.book_state.current_alliances }}</span>
                    </div>
                    <div class="state-item" v-if="truthData.book_state.current_conflict">
                      <span class="state-label">当前冲突</span>
                      <span class="state-value">{{ truthData.book_state.current_conflict }}</span>
                    </div>
                  </div>
                </div>
              </div>

              <div v-if="truthSubTab === 'characters'" class="truth-panel">
                <div v-if="!truthData.characters?.length" class="no-data">
                  <div class="no-data-icon">👤</div>
                  <p>暂无角色数据</p>
                  <span>写作完成后，AI 会自动提取角色信息</span>
                </div>
                <div v-else class="truth-cards">
                  <div v-for="c in truthData.characters" :key="c.id" class="truth-card char-card">
                    <div class="char-avatar">{{ c.name[0] }}</div>
                    <div class="char-info">
                      <div class="char-header">
                        <span class="char-name">{{ c.name }}</span>
                        <span class="char-role" :class="c.role_type">{{ roleLabel(c.role_type) }}</span>
                        <span class="char-role minor" v-if="c.is_placeholder">占位角色</span>
                      </div>
                      <div class="char-profile" v-if="c.profile">{{ c.profile }}</div>
                      <div class="char-sections">
                        <div class="char-section" v-if="c.core_tags">
                          <span class="char-section-label">核心标签</span>
                          <div class="char-section-body">{{ c.core_tags }}</div>
                        </div>
                        <div class="char-section" v-if="c.current_status">
                          <span class="char-section-label">当前现状</span>
                          <div class="char-section-body">{{ c.current_status }}</div>
                        </div>
                        <div class="char-section" v-if="c.inner_drive">
                          <span class="char-section-label">内在驱动</span>
                          <div class="char-section-body">{{ c.inner_drive }}</div>
                        </div>
                        <div class="char-section" v-if="c.relationship_network">
                          <span class="char-section-label">关系网络</span>
                          <div class="char-section-body">{{ c.relationship_network }}</div>
                        </div>
                        <div class="char-section" v-if="c.growth_arc">
                          <span class="char-section-label">成长弧光</span>
                          <div class="char-section-body">{{ c.growth_arc }}</div>
                        </div>
                      </div>
                      <div class="char-meta">
                        <span>创建于 {{ formatDate(c.created_at) }}</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <div v-if="truthSubTab === 'facts'" class="truth-panel">
                <div v-if="!truthData.facts?.length" class="no-data">
                  <div class="no-data-icon">📋</div>
                  <p>暂无长期有效设定</p>
                  <span>这里只展示会持续影响后续章节的身份、资源、物品、规则和关系设定。</span>
                </div>
                <div v-else class="fact-groups">
                  <div v-for="group in factGroups" :key="group.key" class="fact-group-card">
                    <div class="fact-group-header">
                      <div class="fact-group-title-row">
                        <span class="fact-group-icon">{{ group.icon }}</span>
                        <span class="fact-group-title">{{ group.label }}</span>
                      </div>
                      <span class="truth-count">{{ group.items.length }}</span>
                    </div>
                    <div class="fact-items">
                      <div v-for="f in group.items" :key="f.id" class="fact-item-card">
                        <div class="fact-item-main">
                          <span class="fact-subject">{{ f.subject }}</span>
                          <span class="fact-predicate">{{ f.predicate }}</span>
                          <span class="fact-object">{{ f.object }}</span>
                        </div>
                        <div class="fact-item-meta">
                          <span class="fact-chapter-badge">第{{ f.valid_from_chapter }}章生效</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <div v-if="truthSubTab === 'hooks'" class="truth-panel">
                <div v-if="!truthData.hooks?.length" class="no-data">
                  <div class="no-data-icon">🪝</div>
                  <p>暂无伏笔数据</p>
                  <span>写作完成后，AI 会自动识别伏笔</span>
                </div>
                <div v-else class="truth-cards">
                  <div v-for="h in truthData.hooks" :key="h.id" class="truth-card hook-card" :class="'hook-' + h.status">
                    <div class="hook-card-top">
                      <div class="hook-id-row">
                        <span class="hook-id">{{ h.hook_id }}</span>
                        <span class="hook-type-tag" :class="h.type">{{ hookTypeLabel(h.type) }}</span>
                        <span class="hook-status-tag" :class="h.status">{{ hookStatusLabel(h.status) }}</span>
                        <span v-if="h.is_critical" class="critical-badge">关键</span>
                      </div>
                      <div class="hook-notes" v-if="h.notes">{{ h.notes }}</div>
                    </div>
                    <div class="hook-card-bottom">
                      <div class="hook-meta-item">
                        <span class="hook-meta-label">起始</span>
                        <span>第{{ h.start_chapter }}章</span>
                      </div>
                      <div class="hook-meta-item" v-if="h.last_advanced_chapter">
                        <span class="hook-meta-label">最近推进</span>
                        <span>第{{ h.last_advanced_chapter }}章</span>
                      </div>
                      <div class="hook-meta-item" v-if="h.payoff_timing">
                        <span class="hook-meta-label">回收节奏</span>
                        <span class="payoff-tag">{{ payoffLabel(h.payoff_timing) }}</span>
                      </div>
                      <div class="hook-meta-item" v-if="h.expected_payoff">
                        <span class="hook-meta-label">预期回收</span>
                        <span>{{ h.expected_payoff }}</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <div v-if="truthSubTab === 'summaries'" class="truth-panel">
                <div v-if="!truthData.summaries?.length" class="no-data">
                  <div class="no-data-icon">📝</div>
                  <p>暂无章节摘要</p>
                  <span>写作完成后，AI 会自动生成章节摘要</span>
                </div>
                <div v-else class="summary-cards">
                  <div v-for="s in truthData.summaries" :key="s.id" class="summary-card">
                    <div class="summary-header">
                      <span class="summary-chapter">第{{ s.chapter_number }}章</span>
                      <span class="summary-title">{{ s.title }}</span>
                      <span class="chapter-type-tag">{{ s.chapter_type }}</span>
                    </div>
                    <div class="summary-body">
                      <div class="summary-row" v-if="s.characters_appeared">
                        <span class="summary-label">出场人物</span>
                        <span>{{ s.characters_appeared }}</span>
                      </div>
                      <div class="summary-row" v-if="s.key_events">
                        <span class="summary-label">关键事件</span>
                        <span>{{ s.key_events }}</span>
                      </div>
                      <div class="summary-row" v-if="s.state_changes">
                        <span class="summary-label">状态变化</span>
                        <span>{{ s.state_changes }}</span>
                      </div>
                      <div class="summary-row" v-if="s.hook_activity">
                        <span class="summary-label">伏笔动态</span>
                        <span>{{ s.hook_activity }}</span>
                      </div>
                    </div>
                    <div class="summary-footer">
                      <span class="mood-tag" v-if="s.mood">{{ s.mood }}</span>
                    </div>
                  </div>
                </div>
              </div>

              <div v-if="truthSubTab === 'foundations'" class="truth-panel">
                <div v-if="!truthData.foundations?.length" class="no-data">
                  <div class="no-data-icon">📐</div>
                  <p>暂无基础文件</p>
                  <span>基础文件是小说创作的核心设定文档</span>
                </div>
                <div v-else class="foundation-list">
                  <div v-for="f in truthData.foundations" :key="f.id" class="foundation-card">
                    <div class="foundation-header" @click="toggleFoundation(f.id)">
                      <div class="foundation-header-left">
                        <span class="foundation-icon">{{ foundationIcon(f.file_type) }}</span>
                        <span class="foundation-name">{{ foundationLabel(f.file_type) }}</span>
                      </div>
                      <div class="foundation-header-right">
                        <span class="foundation-time">{{ formatDate(f.updated_at) }}</span>
                        <span class="foundation-toggle">{{ expandedFoundations[f.id] ? '收起 ▲' : '展开 ▼' }}</span>
                      </div>
                    </div>
                    <div v-if="expandedFoundations[f.id]" class="foundation-body">
                      <div class="foundation-content" v-html="renderMarkdown(f.content)"></div>
                    </div>
                  </div>
                </div>
              </div>

              <div v-if="truthSubTab === 'snapshots'" class="truth-panel">
                <div v-if="!truthData.snapshots?.length" class="no-data">
                  <div class="no-data-icon">📸</div>
                  <p>暂无章节快照</p>
                  <span>每章写作完成后会自动保存快照</span>
                </div>
                <div v-else class="snapshot-timeline">
                  <div v-for="sn in truthData.snapshots" :key="sn.id" class="snapshot-card">
                    <div class="snapshot-header" @click="toggleSnapshot(sn.id)">
                      <div class="snapshot-header-left">
                        <div class="snapshot-dot"></div>
                        <span class="snapshot-title">第{{ sn.chapter_number }}章快照</span>
                      </div>
                      <div class="snapshot-header-right">
                        <span class="snapshot-time">{{ formatDate(sn.created_at) }}</span>
                        <span class="snapshot-toggle">{{ expandedSnapshots[sn.id] ? '▼' : '▶' }}</span>
                      </div>
                    </div>
                    <div v-if="expandedSnapshots[sn.id]" class="snapshot-body">
                      <div class="snapshot-grid">
                        <div class="snapshot-section">
                          <h4>📋 清单</h4>
                          <pre>{{ formatJSON(sn.manifest_json) }}</pre>
                        </div>
                        <div class="snapshot-section">
                          <h4>📊 当前状态</h4>
                          <pre>{{ formatJSON(sn.current_state_json) }}</pre>
                        </div>
                      </div>
                      <div class="snapshot-grid">
                        <div class="snapshot-section">
                          <h4>🪝 伏笔快照</h4>
                          <pre>{{ formatJSON(sn.hooks_json) }}</pre>
                        </div>
                        <div class="snapshot-section">
                          <h4>📝 摘要快照</h4>
                          <pre>{{ formatJSON(sn.chapter_summaries_json) }}</pre>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-if="tab === 'genres'" class="data-section">
          <div class="section-header">
            <div class="section-count">共 {{ genres.length }} 个题材</div>
            <button class="add-btn" @click="openGenreModal()">+ 新建题材</button>
          </div>
          <div class="table-wrap">
            <table class="data-table">
              <thead>
                <tr>
                  <th>名称</th>
                  <th>简介</th>
                  <th>类型</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="g in genres" :key="g.id">
                  <td>{{ g.name }}</td>
                  <td>
                    <button class="view-btn" @click="viewMarkdown(g)">查看</button>
                  </td>
                  <td>
                    <span class="type-tag" :class="g.user_id === 0 ? 'public' : 'private'">
                      {{ g.user_id === 0 ? '公共' : '私有' }}
                    </span>
                  </td>
                  <td>
                    <div class="action-group" v-if="g.user_id !== 0">
                      <button class="action-btn" @click="openGenreModal(g)">编辑</button>
                      <button class="action-btn danger" @click="deleteMyGenre(g.id)">删除</button>
                    </div>
                    <span v-else class="no-action">—</span>
                  </td>
                </tr>
                <tr v-if="genres.length === 0">
                  <td colspan="4" class="empty-row">暂无可用的题材</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div v-if="tab === 'settings'" class="settings-section">
          <div class="settings-card">
            <div class="setting-item" @click="$router.push('/change-password')">
              <div class="setting-info">
                <div class="setting-label">修改密码</div>
                <div class="setting-desc">更新你的账户密码</div>
              </div>
              <div class="setting-arrow">→</div>
            </div>
          </div>
        </div>

        <div v-if="tab === 'conversations'" class="empty-state">
          <div class="empty-icon">💬</div>
          <h2>对话管理</h2>
          <p>即将上线，敬请期待</p>
        </div>

      </div>
    </main>

    <div v-if="showProgressModal" class="modal-overlay">
      <div class="modal progress-modal">
        <h3>AI 创作中</h3>
        <div v-if="currentWriteRun" class="run-summary">
          <div>Run #{{ currentWriteRun.id }} · 第 {{ currentWriteRun.target_chapter }} 章</div>
          <div>{{ writeRunStatusLabel(currentWriteRun.status) }}<span v-if="currentWriteRun.current_stage"> · {{ currentWriteRun.current_stage }}</span></div>
        </div>
        <div class="progress-steps">
          <div v-for="(step, i) in progressSteps" :key="i" class="progress-step" :class="step.status">
            <div class="step-icon">
              <span v-if="step.status === 'done'">✓</span>
              <span v-else-if="step.status === 'active'" class="spinner"></span>
              <span v-else-if="step.status === 'failed'">!</span>
              <span v-else>{{ i + 1 }}</span>
            </div>
            <div class="step-info">
              <div class="step-label">{{ step.label }}</div>
              <div class="step-desc" v-if="step.desc">{{ step.desc }}</div>
              <div class="step-msg" v-if="step.msg">{{ step.msg }}</div>
            </div>
          </div>
        </div>
        <div v-if="progressError" class="progress-error">{{ progressError }}</div>
        <div v-if="currentWriteRun && (currentWriteRun.status === 'running' || currentWriteRun.status === 'queued')" class="progress-actions">
          <button class="cancel-btn" :disabled="cancellingRun" @click="cancelCurrentRun">
            {{ cancellingRun ? '取消中...' : '取消本次写作' }}
          </button>
        </div>
        <div v-if="currentWriteRun && (currentWriteRun.status === 'failed' || currentWriteRun.status === 'cancelled')" class="progress-actions">
          <button class="cancel-btn" @click="abandonCurrentRun">
            取消本次写作
          </button>
          <button class="cancel-btn" :disabled="retryingRun" @click="retryCurrentRun('restart')">
            {{ retryingRun ? '重试中...' : '整章重跑' }}
          </button>
          <button class="save-btn" :disabled="retryingRun" @click="retryCurrentRun('resume_failed_stage')">
            {{ retryingRun ? '重试中...' : '从失败阶段继续' }}
          </button>
        </div>
        <div v-if="progressStage === 'complete'" class="progress-complete">
          <p>{{ currentWriteRun?.run_type === 'rewrite_latest' ? '重写完成，最新章节已更新。' : '创作完成，最新章节已加入章节列表。' }}</p>
          <button class="save-btn" @click="closeProgress">查看章节</button>
        </div>
      </div>
    </div>

    <div v-if="showMarkdownModal" class="modal-overlay" @click.self="showMarkdownModal = false">
      <div class="modal md-modal">
        <h3>{{ markdownTarget?.name }}</h3>
        <div class="md-content" v-html="renderedMarkdown"></div>
        <div class="modal-actions">
          <button class="cancel-btn" @click="showMarkdownModal = false">关闭</button>
        </div>
      </div>
    </div>

    <div v-if="showGenreModal" class="modal-overlay" @click.self="showGenreModal = false">
      <div class="modal">
        <h3>{{ editingGenre ? '编辑题材' : '新建题材' }}</h3>
        <div class="form-group">
          <label>名称</label>
          <input v-model="genreForm.name" placeholder="题材名称" />
        </div>
        <div class="form-group">
          <label>简介 (Markdown)</label>
          <textarea v-model="genreForm.profile_markdown" placeholder="题材简介，支持 Markdown" rows="6"></textarea>
        </div>
        <div class="modal-actions">
          <button class="cancel-btn" @click="showGenreModal = false">取消</button>
          <button class="save-btn" @click="saveGenre" :disabled="genreSaving">{{ genreSaving ? '保存中...' : '保存' }}</button>
        </div>
      </div>
    </div>

    <div v-if="showChapterModal" class="modal-overlay" @click.self="showChapterModal = false">
      <div class="modal chapter-modal">
        <h3>第{{ viewingChapter?.chapter_number }}章 {{ viewingChapter?.title }}</h3>
        <div class="chapter-content">{{ viewingChapter?.content }}</div>
        <div class="modal-actions">
          <button class="cancel-btn" @click="showChapterModal = false">关闭</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { marked } from 'marked'
import yaml from 'js-yaml'

const router = useRouter()
const user = ref(null)
const tab = ref('books')
const books = ref([])
const genres = ref([])
const llmConfigs = ref([])

const selectedBook = ref(null)
const chapters = ref([])
const writeModelID = ref(0)
const writeInput = ref('')
const writing = ref(false)
const writeResult = ref(null)

const bookTab = ref('write')

const truthLoading = ref(false)
const truthData = ref({ book_state: null, characters: [], facts: [], hooks: [], summaries: [], snapshots: [], foundations: [] })
const truthSubTab = ref('state')
const truthTabs = [
  { key: 'state', label: '当前状态', icon: '🧭' },
  { key: 'characters', label: '人物', icon: '👤' },
  { key: 'facts', label: '设定', icon: '📋' },
  { key: 'hooks', label: '伏笔', icon: '🪝' },
  { key: 'summaries', label: '章节摘要', icon: '📝' },
  { key: 'foundations', label: '基础文件', icon: '📐' },
  { key: 'snapshots', label: '快照', icon: '📸' },
]
const truthCounts = computed(() => ({
  state: truthData.value.book_state ? 1 : 0,
  characters: truthData.value.characters?.length || 0,
  facts: truthData.value.facts?.length || 0,
  hooks: truthData.value.hooks?.length || 0,
  summaries: truthData.value.summaries?.length || 0,
  foundations: truthData.value.foundations?.length || 0,
  snapshots: truthData.value.snapshots?.length || 0,
}))

const factCategoryMeta = {
  identity: { label: '身份', icon: '🧬' },
  resource: { label: '资源', icon: '💠' },
  item: { label: '物品', icon: '📦' },
  rule: { label: '规则', icon: '📏' },
  relationship: { label: '关系', icon: '🤝' },
}

const factGroups = computed(() => {
  const grouped = new Map()
  for (const fact of truthData.value.facts || []) {
    const key = fact.category && factCategoryMeta[fact.category] ? fact.category : 'relationship'
    if (!grouped.has(key)) {
      grouped.set(key, { key, label: factCategoryMeta[key].label, icon: factCategoryMeta[key].icon, items: [] })
    }
    grouped.get(key).items.push(fact)
  }
  return Array.from(grouped.values()).sort((a, b) => (
    Object.keys(factCategoryMeta).indexOf(a.key) - Object.keys(factCategoryMeta).indexOf(b.key)
  ))
})
const expandedSnapshots = reactive({})
const expandedFoundations = reactive({})

const showProgressModal = ref(false)
const progressStage = ref('')
const progressError = ref('')
const currentWriteRun = ref(null)
const writeRunStages = ref([])
const cancellingRun = ref(false)
const retryingRun = ref(false)
let writeRunPollTimer = null
const progressSteps = reactive([
  { key: 'loading', label: '加载书籍信息', desc: '创建本次写作任务，锁定书籍并准备进入创作链路。', status: 'pending', msg: '' },
  { key: 'context', label: '构建上下文', desc: '整理当前书籍的状态、设定、伏笔和历史章节信息。', status: 'pending', msg: '' },
  { key: 'planning', label: 'Planner 规划本章', desc: '生成本章目标、推进点和关键冲突的写作计划。', status: 'pending', msg: '' },
  { key: 'writing', label: 'Writer 创作正文', desc: '根据规划与上下文生成章节正文和结构化分段结果。', status: 'pending', msg: '' },
  { key: 'parsing', label: '解析 Writer 输出', desc: '把 Writer 返回结果拆成标题、正文和后续结算所需片段。', status: 'pending', msg: '' },
  { key: 'auditing', label: 'Auditor 审查结构', desc: '检查章节结构、推进节奏和状态一致性是否合理。', status: 'pending', msg: '' },
  { key: 'revising', label: 'Reviser 修订正文', desc: '当审查不过时，按问题清单修订正文和状态片段。', status: 'pending', msg: '' },
  { key: 'polishing', label: 'Polisher 润色文稿', desc: '在不改变既有设定的前提下优化表达、节奏和可读性。', status: 'pending', msg: '' },
  { key: 'extracting', label: '提取真相文件', desc: '结算本章带来的状态变化，并抽取人物、设定、伏笔等真相数据。', status: 'pending', msg: '' },
  { key: 'snapshot', label: '保存章节快照', desc: '统一提交章节和真相状态，并生成用于回滚的快照。', status: 'pending', msg: '' },
])

const showChapterModal = ref(false)
const viewingChapter = ref(null)

const showMarkdownModal = ref(false)
const markdownTarget = ref(null)
const renderedMarkdown = computed(() => {
  const md = markdownTarget.value?.profile_markdown || ''
  if (!md) return '<p style="color:#94a3b8">暂无简介</p>'

  const frontMatterMatch = md.match(/^---\s*\n([\s\S]*?)\n---\s*\n?([\s\S]*)$/)
  let fmHTML = ''
  let bodyMD = md

  if (frontMatterMatch) {
    try {
      const parsed = yaml.load(frontMatterMatch[1])
      if (parsed && typeof parsed === 'object') {
        fmHTML = '<div class="fm-table-wrap"><table class="fm-table"><tbody>'
        for (const [key, val] of Object.entries(parsed)) {
          const displayVal = Array.isArray(val) ? val.join(', ') : String(val)
          fmHTML += `<tr><td class="fm-key">${key}</td><td class="fm-val">${displayVal}</td></tr>`
        }
        fmHTML += '</tbody></table></div>'
      }
      bodyMD = frontMatterMatch[2] || ''
    } catch {
      bodyMD = md
    }
  }

  const bodyHTML = bodyMD.trim() ? marked(bodyMD) : ''
  return fmHTML + bodyHTML
})

const showGenreModal = ref(false)
const editingGenre = ref(null)
const genreSaving = ref(false)
const genreForm = ref({ name: '', profile_markdown: '' })
const deletingBookID = ref(0)
const deletingChapterNumber = ref(0)

const token = localStorage.getItem('token')

const tabTitle = computed(() => {
  const titles = { books: '书籍管理', genres: '题材管理', settings: '个人设置', conversations: '对话管理' }
  return titles[tab.value] || ''
})

const statusLabels = {
  initializing: '初始化中',
  outlining: '大纲阶段',
  active: '创作中',
  writing: '写作中',
  paused: '已暂停',
  completed: '已完成',
}
const writeRunStatusLabels = {
  queued: '排队中',
  running: '运行中',
  succeeded: '成功',
  failed: '失败',
  cancelled: '已取消',
}

const canWrite = computed(() => {
  if (!selectedBook.value || writing.value) return false
  return selectedBook.value.status === 'outlining' || selectedBook.value.status === 'active'
})

const writeDisabledReason = computed(() => {
  if (!selectedBook.value) return ''
  if (writing.value) return '当前章节正在生成中，请等待本次写作结束。'
  if (selectedBook.value.status === 'initializing') return '书籍仍在初始化基础设定，请稍后再写下一章。'
  if (selectedBook.value.status === 'writing') return '该书已有写作任务在进行中，请勿重复提交。'
  if (selectedBook.value.status === 'paused') return '该书当前已暂停，需恢复到可写状态后再继续。'
  if (selectedBook.value.status === 'completed') return '该书已完结，如需续写请先调整状态。'
  return ''
})

function statusLabel(s) {
  return statusLabels[s] || s
}

function roleLabel(r) {
  const labels = { protagonist: '主角', major: '重要角色', minor: '次要角色' }
  return labels[r] || r
}

function hookStatusLabel(s) {
  const labels = { seed: '种子', open: '开放', progressing: '推进中', resolved: '已回收', deferred: '延后', stale: '过期' }
  return labels[s] || s
}

function hookTypeLabel(t) {
  const labels = { plot: '剧情', conflict: '冲突', item: '道具', mystery: '悬疑', character: '人物' }
  return labels[t] || t
}

function payoffLabel(p) {
  const labels = { immediate: '立即', 'near-term': '近期', 'mid-term': '中期', 'slow-burn': '长线' }
  return labels[p] || p
}

function foundationLabel(ft) {
  const labels = {
    story_frame: '故事框架',
    volume_map: '卷地图',
    book_rules: '创作规则',
    author_intent: '作者意图',
    style_guide: '风格指南',
    current_focus: '当前焦点',
    audit_drift: '偏离审计',
  }
  return labels[ft] || ft
}

function foundationIcon(ft) {
  const icons = {
    story_frame: '🏗️',
    volume_map: '🗺️',
    book_rules: '📏',
    author_intent: '🎯',
    style_guide: '🎨',
    current_focus: '🔍',
    audit_drift: '⚠️',
  }
  return icons[ft] || '📄'
}

function toggleFoundation(id) {
  expandedFoundations[id] = !expandedFoundations[id]
}

function renderMarkdown(md) {
  if (!md) return ''
  return marked(md)
}

function formatDate(d) {
  if (!d) return '-'
  return new Date(d).toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function formatJSON(str) {
  try {
    return JSON.stringify(JSON.parse(str), null, 2)
  } catch {
    return str
  }
}

function toggleSnapshot(id) {
  expandedSnapshots[id] = !expandedSnapshots[id]
}

function isLatestChapter(ch) {
  if (!chapters.value.length) return false
  return ch.chapter_number === chapters.value[chapters.value.length - 1].chapter_number
}

function leaveBookView() {
  stopWriteRunPolling()
  selectedBook.value = null
  chapters.value = []
  bookTab.value = 'write'
  truthSubTab.value = 'state'
  truthData.value = { book_state: null, characters: [], facts: [], hooks: [], summaries: [], snapshots: [], foundations: [] }
  writeInput.value = ''
  writeResult.value = null
  showChapterModal.value = false
  viewingChapter.value = null
  currentWriteRun.value = null
  writeRunStages.value = []
}

async function switchTab(nextTab) {
  tab.value = nextTab
  if (nextTab !== 'books') {
    leaveBookView()
  }
  if (nextTab === 'genres') {
    await loadGenres()
  }
}

async function loadBooks() {
  const [userRes, booksRes, llmRes] = await Promise.all([
    fetch('/api/v1/me', { headers: { 'Authorization': 'Bearer ' + token } }),
    fetch('/api/v1/books', { headers: { 'Authorization': 'Bearer ' + token } }),
    fetch('/api/v1/llm-configs', { headers: { 'Authorization': 'Bearer ' + token } }),
  ])
  if (userRes.ok) user.value = await userRes.json()
  if (booksRes.ok) books.value = await booksRes.json()
  if (llmRes.ok) llmConfigs.value = await llmRes.json()
}

onMounted(async () => {
  await loadBooks()
})

onUnmounted(() => {
  stopWriteRunPolling()
})

async function loadGenres() {
  const res = await fetch('/api/v1/my-genres', { headers: { 'Authorization': 'Bearer ' + token } })
  if (res.ok) genres.value = await res.json()
}

async function openBook(id) {
  const res = await fetch(`/api/v1/books/${id}`, { headers: { 'Authorization': 'Bearer ' + token } })
  if (res.ok) {
    const data = await res.json()
    selectedBook.value = data.book
    chapters.value = data.chapters
    writeModelID.value = data.book.llm_model_id || 0
    if (data.book.status === 'writing') {
      await restoreActiveWriteRun(id)
    }
  }
}

async function deleteBook(book, evt = null) {
  evt?.stopPropagation?.()
  if (!book?.id) return
  if (!confirm(`确定删除书籍《${book.title}》吗？这会删除该书的章节、真相文件和运行产物，且不可恢复。`)) return

  deletingBookID.value = book.id
  try {
    const res = await fetch(`/api/v1/books/${book.id}`, {
      method: 'DELETE',
      headers: { 'Authorization': 'Bearer ' + token },
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      alert(data.error || '删除书籍失败')
      return
    }

    if (selectedBook.value?.id === book.id) {
      leaveBookView()
      showProgressModal.value = false
      writing.value = false
      progressError.value = ''
      progressStage.value = ''
    }

    await loadBooks()
  } finally {
    deletingBookID.value = 0
  }
}

async function exportBook(format) {
  if (!selectedBook.value?.id || exporting.value) return
  exporting.value = true
  try {
    const res = await fetch(`/api/v1/books/${selectedBook.value.id}/export?format=${format}`, {
      headers: { 'Authorization': 'Bearer ' + token },
    })
    if (!res.ok) {
      let msg = '导出失败'
      try { const data = await res.json(); msg = data.error || msg } catch (e) {}
      alert(msg)
      return
    }
    const blob = await res.blob()
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${selectedBook.value.title || 'book'}.${format}`
    document.body.appendChild(a)
    a.click()
    a.remove()
    window.URL.revokeObjectURL(url)
  } catch (e) {
    alert('网络错误：' + e.message)
  } finally {
    exporting.value = false
  }
}

async function loadTruthFiles() {
  if (!selectedBook.value) return
  truthLoading.value = true
  try {
    const res = await fetch(`/api/v1/books/${selectedBook.value.id}/truth-files`, {
      headers: { 'Authorization': 'Bearer ' + token },
    })
    if (res.ok) {
      truthData.value = await res.json()
    }
  } finally {
    truthLoading.value = false
  }
}

function resetProgress() {
  progressSteps.forEach(s => { s.status = 'pending'; s.msg = '' })
  progressStage.value = ''
  progressError.value = ''
  writeRunStages.value = []
  currentWriteRun.value = null
}

async function finalizeWrite(result = null) {
  await loadBooks()
  if (selectedBook.value?.id) {
    await openBook(selectedBook.value.id)
    await loadTruthFiles()
  }

  const latestChapter = chapters.value.length ? chapters.value[chapters.value.length - 1] : null
  if (result) {
    writeResult.value = {
      ...result,
      content: result.content || latestChapter?.content || '',
      title: result.title || latestChapter?.title || '',
      chapter_number: result.chapter_number || latestChapter?.chapter_number || '',
    }
  } else if (latestChapter) {
    writeResult.value = {
      chapter_number: latestChapter.chapter_number,
      title: latestChapter.title,
      content: latestChapter.content,
      memo: writeResult.value?.memo || '',
    }
  }

  progressStage.value = 'complete'
  progressSteps.forEach(s => { s.status = 'done' })
  writeInput.value = ''
}

function stopWriteRunPolling() {
  if (writeRunPollTimer) {
    clearTimeout(writeRunPollTimer)
    writeRunPollTimer = null
  }
}

function writeRunStatusLabel(status) {
  return writeRunStatusLabels[status] || status
}

function syncProgressFromRun(run, stages) {
  progressError.value = ''
  progressStage.value = run?.current_stage || ''
  const stageMap = new Map(stages.map(stage => [stage.stage, stage]))

  progressSteps.forEach(step => {
    if (step.key === 'loading') {
      step.status = run ? 'done' : 'pending'
      step.msg = run ? '任务已创建' : ''
      return
    }
    if (step.key === 'parsing') {
      const writingStage = stageMap.get('writing')
      const auditingStage = stageMap.get('auditing')
      if (auditingStage || stageMap.get('revising') || stageMap.get('polishing') || stageMap.get('extracting') || stageMap.get('snapshot')) {
        step.status = 'done'
        step.msg = '已完成'
      } else if (writingStage && writingStage.status === 'succeeded') {
        step.status = 'active'
        step.msg = '执行中'
      } else {
        step.status = 'pending'
        step.msg = ''
      }
      return
    }
    const stage = stageMap.get(step.key)
    if (!stage) {
      step.status = 'pending'
      step.msg = ''
      return
    }
    if (stage.status === 'succeeded' || stage.status === 'skipped') {
      step.status = 'done'
      step.msg = stage.status === 'skipped' ? '已跳过' : '已完成'
    } else if (stage.status === 'running') {
      step.status = 'active'
      step.msg = '执行中'
    } else if (stage.status === 'failed' || stage.status === 'cancelled') {
      step.status = 'failed'
      step.msg = stage.status === 'cancelled' ? '已取消' : '失败'
      progressError.value = run?.error_message || stage.error_message || ''
    } else {
      step.status = 'pending'
      step.msg = ''
    }
  })

  if (run?.status === 'succeeded') {
    progressStage.value = 'complete'
    progressSteps.forEach(step => { step.status = 'done' })
  }
  if ((run?.status === 'failed' || run?.status === 'cancelled') && !progressError.value) {
    progressError.value = run?.error_message || (run?.status === 'cancelled' ? '写作已取消' : '写作失败')
  }
}

async function refreshWriteRun(runID) {
  const res = await fetch(`/api/v1/write-runs/${runID}/stages`, {
    headers: { 'Authorization': 'Bearer ' + token },
  })
  if (!res.ok) throw new Error('获取写作任务详情失败')
  const data = await res.json()
  currentWriteRun.value = data.run
  writeRunStages.value = data.stages || []
  syncProgressFromRun(data.run, writeRunStages.value)
  return data.run
}

async function pollWriteRun(runID) {
  stopWriteRunPolling()
  try {
    const run = await refreshWriteRun(runID)
    if (!run) return
    if (run.status === 'succeeded') {
      writing.value = false
      await finalizeWrite({
        chapter_number: run.target_chapter,
        title: currentWriteRun.value?.title || '',
      })
      return
    }
    if (run.status === 'failed' || run.status === 'cancelled') {
      writing.value = false
      await loadBooks()
      if (selectedBook.value?.id) {
        await openBook(selectedBook.value.id)
      }
      return
    }
    writeRunPollTimer = setTimeout(() => {
      pollWriteRun(runID)
    }, 1500)
  } catch (e) {
    progressError.value = '获取写作进度失败：' + e.message
    writing.value = false
  }
}

async function restoreActiveWriteRun(bookID) {
  const res = await fetch(`/api/v1/books/${bookID}/write-runs/active`, {
    headers: { 'Authorization': 'Bearer ' + token },
  })
  if (!res.ok) return
  const data = await res.json()
  if (!data.run) return
  showProgressModal.value = true
  writing.value = true
  currentWriteRun.value = data.run
  await pollWriteRun(data.run.id)
}

async function writeChapter() {
  writing.value = true
  writeResult.value = null
  resetProgress()
  showProgressModal.value = true

  try {
    const res = await fetch(`/api/v1/books/${selectedBook.value.id}/write-runs`, {
      method: 'POST',
      headers: {
        'Authorization': 'Bearer ' + token,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
      model_id: writeModelID.value,
      user_input: writeInput.value,
      }),
    })

    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      progressError.value = err.error || '写作失败'
      writing.value = false
      return
    }
    const data = await res.json()
    currentWriteRun.value = data.run
    await pollWriteRun(data.run.id)
  } catch (e) {
    progressError.value = '网络错误: ' + e.message
    writing.value = false
  }
}

async function rewriteLatestChapter(ch) {
  if (!selectedBook.value?.id || !ch?.chapter_number) return
  if (!isLatestChapter(ch)) {
    alert('当前只支持重写最后一章')
    return
  }
  if (!canWrite.value) {
    alert(writeDisabledReason.value || '当前状态不可重写')
    return
  }
  const instruction = window.prompt(`请输入第${ch.chapter_number}章《${ch.title || '未命名'}》的重写要求`, writeInput.value || '')
  if (instruction === null) return
  if (!instruction.trim()) {
    alert('重写最后一章需要填写重写要求')
    return
  }

  writing.value = true
  writeResult.value = null
  resetProgress()
  showProgressModal.value = true
  progressError.value = ''

  try {
    const res = await fetch(`/api/v1/books/${selectedBook.value.id}/write-runs`, {
      method: 'POST',
      headers: {
        'Authorization': 'Bearer ' + token,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        model_id: writeModelID.value,
        user_input: instruction.trim(),
        run_type: 'rewrite_latest',
      }),
    })

    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      progressError.value = data.error || '重写章节失败'
      writing.value = false
      return
    }
    currentWriteRun.value = data.run
    await pollWriteRun(data.run.id)
  } catch (e) {
    progressError.value = '网络错误: ' + e.message
    writing.value = false
  }
}

async function cancelCurrentRun() {
  if (!currentWriteRun.value?.id || cancellingRun.value) return
  cancellingRun.value = true
  try {
    const res = await fetch(`/api/v1/write-runs/${currentWriteRun.value.id}/cancel`, {
      method: 'POST',
      headers: { 'Authorization': 'Bearer ' + token },
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      progressError.value = data.error || '取消写作失败'
      return
    }
    await pollWriteRun(currentWriteRun.value.id)
  } finally {
    cancellingRun.value = false
  }
}

async function retryCurrentRun(mode) {
  if (!currentWriteRun.value?.id || retryingRun.value) return
  retryingRun.value = true
  progressError.value = ''
  try {
    const res = await fetch(`/api/v1/write-runs/${currentWriteRun.value.id}/retry`, {
      method: 'POST',
      headers: {
        'Authorization': 'Bearer ' + token,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ mode }),
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      progressError.value = data.error || '重试失败'
      return
    }
    writing.value = true
    resetProgress()
    currentWriteRun.value = data.run
    await pollWriteRun(data.run.id)
  } finally {
    retryingRun.value = false
  }
}

async function abandonCurrentRun() {
  stopWriteRunPolling()
  writing.value = false
  cancellingRun.value = false
  retryingRun.value = false
  currentWriteRun.value = null
  writeRunStages.value = []
  progressStage.value = ''
  progressError.value = ''
  showProgressModal.value = false
  await loadBooks()
  if (selectedBook.value?.id) {
    await openBook(selectedBook.value.id)
  }
}

async function deleteChapter(ch) {
  if (!selectedBook.value?.id || !ch?.chapter_number) return
  if (!isLatestChapter(ch)) {
    alert('当前只支持删除最后一章')
    return
  }
  if (!confirm(`确定删除第${ch.chapter_number}章《${ch.title || '未命名'}》吗？这会删除该章关联产物，并恢复到可继续写作的状态。`)) return

  deletingChapterNumber.value = ch.chapter_number
  try {
    const res = await fetch(`/api/v1/books/${selectedBook.value.id}/chapters/${ch.chapter_number}`, {
      method: 'DELETE',
      headers: { 'Authorization': 'Bearer ' + token },
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      alert(data.error || '删除章节失败')
      return
    }

    if (viewingChapter.value?.id === ch.id) {
      viewingChapter.value = null
      showChapterModal.value = false
    }
    writeResult.value = null
    await openBook(selectedBook.value.id)
    await loadTruthFiles()
  } finally {
    deletingChapterNumber.value = 0
  }
}

function closeProgress() {
  stopWriteRunPolling()
  showProgressModal.value = false
}

function viewChapter(ch) {
  viewingChapter.value = ch
  showChapterModal.value = true
}

function viewMarkdown(g) {
  markdownTarget.value = g
  showMarkdownModal.value = true
}

function openGenreModal(genre) {
  if (genre) {
    editingGenre.value = genre
    genreForm.value = { name: genre.name, profile_markdown: genre.profile_markdown || '' }
  } else {
    editingGenre.value = null
    genreForm.value = { name: '', profile_markdown: '' }
  }
  showGenreModal.value = true
}

async function saveGenre() {
  genreSaving.value = true
  try {
    const body = JSON.stringify(genreForm.value)
    let res
    if (editingGenre.value) {
      res = await fetch(`/api/v1/my-genres/${editingGenre.value.id}`, {
        method: 'PUT',
        headers: { 'Authorization': 'Bearer ' + token, 'Content-Type': 'application/json' },
        body,
      })
    } else {
      res = await fetch('/api/v1/my-genres', {
        method: 'POST',
        headers: { 'Authorization': 'Bearer ' + token, 'Content-Type': 'application/json' },
        body,
      })
    }
    if (res.ok) {
      showGenreModal.value = false
      await loadGenres()
    }
  } finally {
    genreSaving.value = false
  }
}

async function deleteMyGenre(id) {
  if (!confirm('确定删除该题材？')) return
  const res = await fetch(`/api/v1/my-genres/${id}`, {
    method: 'DELETE',
    headers: { 'Authorization': 'Bearer ' + token },
  })
  if (res.ok) await loadGenres()
}

function logout() {
  localStorage.removeItem('token')
  localStorage.removeItem('role')
  router.push('/login')
}
</script>

<style scoped>
.write-container {
  display: flex;
  min-height: 100vh;
  background: #f1f5f9;
}
.sidebar {
  width: 220px;
  background: #fff;
  border-right: 1px solid #e2e8f0;
  display: flex;
  flex-direction: column;
  padding: 24px 0;
}
.sidebar-logo {
  padding: 0 20px 20px;
  border-bottom: 1px solid #e2e8f0;
}
.sidebar-logo h2 {
  font-size: 20px;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
.sidebar-nav {
  flex: 1;
  padding: 12px 0;
}
.nav-item {
  display: block;
  padding: 12px 20px;
  color: #64748b;
  text-decoration: none;
  font-size: 14px;
  transition: all 0.2s;
  border-left: 3px solid transparent;
  cursor: pointer;
}
.nav-item:hover, .nav-item.active {
  color: #1e293b;
  background: #f1f5f9;
  border-left-color: #2563eb;
}
.sidebar-footer {
  padding: 16px 20px;
  border-top: 1px solid #e2e8f0;
}
.user-info {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}
.avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  color: #fff;
}
.user-name { font-size: 14px; color: #1e293b; }
.logout-btn {
  width: 100%;
  padding: 8px;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  background: transparent;
  color: #64748b;
  cursor: pointer;
  font-size: 13px;
}
.logout-btn:hover { color: #dc2626; border-color: #dc2626; }
.main-content { flex: 1; display: flex; flex-direction: column; background: #fff; }
.topbar {
  padding: 18px 32px;
  border-bottom: 1px solid #e2e8f0;
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.topbar h1 { font-size: 20px; color: #1e293b; }
.back-link { color: #64748b; text-decoration: none; margin-right: 12px; font-size: 16px; }
.back-link:hover { color: #2563eb; }
.create-btn {
  padding: 8px 20px;
  border: none;
  border-radius: 6px;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  color: #fff;
  font-size: 13px;
  cursor: pointer;
  text-decoration: none;
  transition: opacity 0.2s;
}
.create-btn:hover { opacity: 0.9; }
.content { padding: 32px; flex: 1; background: #f8fafc; }

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 40px;
  text-align: center;
}
.empty-icon { font-size: 48px; margin-bottom: 20px; }
.empty-state h2 { font-size: 26px; color: #1e293b; margin-bottom: 12px; }
.empty-state p { color: #64748b; font-size: 15px; max-width: 420px; line-height: 1.6; }
.empty-actions { margin-top: 28px; }
.welcome-btn {
  padding: 12px 32px;
  border-radius: 8px;
  font-size: 15px;
  cursor: pointer;
  transition: all 0.3s;
  border: 1px solid #e2e8f0;
  background: transparent;
  color: #64748b;
  text-decoration: none;
  display: inline-block;
}
.welcome-btn:hover { border-color: #2563eb; color: #2563eb; }
.welcome-btn.primary {
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  border: none;
  color: #fff;
  font-weight: 600;
}
.welcome-btn.primary:hover { opacity: 0.9; }

.data-section { }
.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.section-count { color: #64748b; font-size: 14px; }
.table-wrap {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  overflow-x: auto;
  overflow-y: hidden;
}
.data-table {
  width: 100%;
  border-collapse: collapse;
}
.data-table th {
  text-align: left;
  padding: 14px 20px;
  font-size: 13px;
  color: #64748b;
  background: #f8fafc;
  border-bottom: 1px solid #e2e8f0;
  font-weight: 600;
}
.data-table td {
  padding: 14px 20px;
  font-size: 14px;
  color: #334155;
  border-bottom: 1px solid #f1f5f9;
}
.data-table tr:last-child td { border-bottom: none; }
.data-table tr:hover td { background: #f8fafc; }
.book-title { font-weight: 600; color: #1e293b; }
.empty-row { text-align: center; color: #94a3b8; padding: 32px !important; }
.status-tag {
  display: inline-block;
  padding: 2px 10px;
  border-radius: 4px;
  font-size: 12px;
}
.status-tag.initializing { background: rgba(245,158,11,0.12); color: #d97706; }
.status-tag.outlining { background: rgba(37,99,235,0.1); color: #2563eb; }
.status-tag.active { background: rgba(22,163,74,0.1); color: #16a34a; }
.status-tag.writing { background: rgba(14,165,233,0.12); color: #0284c7; }
.status-tag.paused { background: #f1f5f9; color: #94a3b8; }
.status-tag.completed { background: rgba(139,92,246,0.1); color: #8b5cf6; }

.book-detail { }
.book-meta {
  display: flex;
  align-items: flex-start;
  flex-wrap: wrap;
  gap: 24px;
  padding: 16px 20px;
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  margin-bottom: 20px;
}
.book-meta-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
}
.meta-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.meta-label { font-size: 12px; color: #94a3b8; }
.meta-value { font-size: 14px; color: #1e293b; font-weight: 500; }

.book-tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 20px;
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  padding: 4px;
}
.book-tab {
  flex: 1;
  padding: 10px 20px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: #64748b;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}
.book-tab.active {
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  color: #fff;
}
.book-tab:hover:not(.active) { color: #1e293b; background: #f1f5f9; }

.write-panel {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 20px;
  margin-bottom: 20px;
}
.write-controls {
  display: flex;
  gap: 12px;
  align-items: flex-end;
}
.control-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.control-group label {
  font-size: 12px;
  color: #64748b;
}
.control-group select,
.control-group input {
  padding: 8px 12px;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  background: #f8fafc;
  color: #1e293b;
  font-size: 13px;
  outline: none;
}
.control-group select:focus,
.control-group input:focus { border-color: #2563eb; }
.control-group input { width: 280px; }
.write-btn {
  padding: 8px 24px;
  border: none;
  border-radius: 6px;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
}
.write-btn:hover:not(:disabled) { opacity: 0.9; }
.write-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.write-hint {
  margin-top: 10px;
  font-size: 12px;
  color: #94a3b8;
}

.write-result {
  margin-top: 20px;
  border-top: 1px solid #e2e8f0;
  padding-top: 20px;
}
.result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.result-header h3 { font-size: 18px; color: #1e293b; }
.result-memo { margin-bottom: 16px; }
.result-memo details { }
.result-memo summary {
  cursor: pointer;
  color: #2563eb;
  font-size: 13px;
  margin-bottom: 8px;
}
.result-memo pre {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 16px;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  max-height: 300px;
  overflow-y: auto;
}
.result-content {
  color: #334155;
  font-size: 15px;
  line-height: 2;
}
.result-content :deep(p) { margin: 12px 0; text-indent: 2em; }

.chapters-section { }
.chapters-section h3 {
  font-size: 16px;
  color: #1e293b;
  margin-bottom: 16px;
}
.no-chapters {
  text-align: center;
  color: #94a3b8;
  padding: 40px;
  font-size: 14px;
}
.chapter-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.chapter-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 16px;
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}
.chapter-item:hover { border-color: #2563eb; background: #f8fafc; }
.chapter-num { font-size: 13px; color: #2563eb; font-weight: 600; min-width: 60px; }
.chapter-title { flex: 1; font-size: 14px; color: #1e293b; }
.chapter-meta { display: flex; gap: 12px; font-size: 12px; color: #94a3b8; }
.chapter-actions { display: flex; justify-content: flex-end; }
.chapter-status.draft { color: #f59e0b; }
.chapter-status.reviewed { color: #2563eb; }
.chapter-status.published { color: #16a34a; }

.truth-section { }
.truth-loading {
  text-align: center;
  color: #94a3b8;
  padding: 40px;
}
.truth-tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 20px;
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  padding: 4px;
  overflow-x: auto;
}
.truth-tab {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: #64748b;
  font-size: 13px;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.2s;
}
.truth-tab-icon { font-size: 14px; }
.truth-tab.active {
  background: #2563eb;
  color: #fff;
}
.truth-tab:hover:not(.active) { color: #1e293b; background: #f1f5f9; }
.truth-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 9px;
  font-size: 11px;
  font-weight: 600;
  background: rgba(255,255,255,0.25);
}
.truth-tab:not(.active) .truth-count {
  background: #e2e8f0;
  color: #64748b;
}

.truth-panel { }
.no-data {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  text-align: center;
  color: #94a3b8;
  padding: 60px 40px;
}
.no-data-icon { font-size: 40px; margin-bottom: 8px; }
.no-data p { font-size: 15px; color: #64748b; margin: 0; }
.no-data span { font-size: 13px; color: #94a3b8; }

.truth-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.truth-card {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  padding: 14px 18px;
  transition: border-color 0.2s;
}
.truth-card:hover { border-color: #cbd5e1; }

.char-card {
  display: flex;
  gap: 14px;
  align-items: flex-start;
}
.char-avatar {
  width: 42px;
  height: 42px;
  border-radius: 50%;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  font-weight: 700;
  color: #fff;
  flex-shrink: 0;
}
.char-info { flex: 1; min-width: 0; }
.char-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 6px;
}
.char-name {
  font-size: 15px;
  font-weight: 600;
  color: #1e293b;
}
.char-role {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 500;
}
.char-role.protagonist { background: rgba(37,99,235,0.1); color: #2563eb; }
.char-role.major { background: rgba(245,158,11,0.1); color: #d97706; }
.char-role.minor { background: #f1f5f9; color: #64748b; }
.char-profile {
  font-size: 13px;
  color: #475569;
  line-height: 1.6;
}
.state-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.state-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}
.state-title {
  font-size: 16px;
  font-weight: 700;
  color: #0f172a;
}
.state-subtitle {
  margin-top: 4px;
  font-size: 12px;
  color: #94a3b8;
}
.state-summary {
  font-size: 14px;
  color: #334155;
  line-height: 1.7;
}
.state-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
.state-item {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.state-label {
  font-size: 12px;
  color: #64748b;
}
.state-value {
  font-size: 14px;
  color: #1e293b;
  line-height: 1.6;
}
.char-sections {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 10px;
}
.char-section {
  background: #f8fafc;
  border-radius: 8px;
  padding: 10px 12px;
}
.char-section-label {
  display: block;
  font-size: 12px;
  font-weight: 600;
  color: #2563eb;
  margin-bottom: 4px;
}
.char-section-body {
  font-size: 13px;
  color: #475569;
  line-height: 1.65;
  white-space: pre-wrap;
}
.char-meta {
  margin-top: 6px;
  font-size: 11px;
  color: #94a3b8;
}

.hook-card {
  border-left: 3px solid #e2e8f0;
}
.hook-card.hook-seed { border-left-color: #8b5cf6; }
.hook-card.hook-open { border-left-color: #2563eb; }
.hook-card.hook-progressing { border-left-color: #f59e0b; }
.hook-card.hook-resolved { border-left-color: #16a34a; }
.hook-card.hook-deferred { border-left-color: #94a3b8; }
.hook-card.hook-stale { border-left-color: #dc2626; }
.hook-card-top { margin-bottom: 10px; }
.hook-id-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
  flex-wrap: wrap;
}
.hook-id {
  font-size: 13px;
  font-weight: 700;
  color: #2563eb;
  font-family: 'SF Mono', 'Fira Code', monospace;
}
.hook-type-tag {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  background: #f1f5f9;
  color: #64748b;
}
.hook-status-tag {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 500;
}
.hook-status-tag.seed { background: rgba(139,92,246,0.1); color: #8b5cf6; }
.hook-status-tag.open { background: rgba(37,99,235,0.1); color: #2563eb; }
.hook-status-tag.progressing { background: rgba(245,158,11,0.1); color: #d97706; }
.hook-status-tag.resolved { background: rgba(22,163,74,0.1); color: #16a34a; }
.hook-status-tag.deferred { background: #f1f5f9; color: #94a3b8; }
.hook-status-tag.stale { background: rgba(220,38,38,0.06); color: #dc2626; }
.critical-badge {
  display: inline-block;
  padding: 1px 6px;
  border-radius: 3px;
  background: rgba(220,38,38,0.1);
  color: #dc2626;
  font-size: 11px;
  font-weight: 600;
}
.hook-notes {
  font-size: 13px;
  color: #475569;
  line-height: 1.6;
}
.hook-card-bottom {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
  padding-top: 10px;
  border-top: 1px solid #f1f5f9;
}
.hook-meta-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 12px;
  color: #64748b;
}
.hook-meta-label {
  font-size: 10px;
  color: #94a3b8;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.payoff-tag {
  display: inline-block;
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 11px;
  background: rgba(37,99,235,0.08);
  color: #2563eb;
}

.fact-groups {
  display: grid;
  gap: 16px;
}
.fact-group-card {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 16px;
}
.fact-group-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.fact-group-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.fact-group-icon {
  font-size: 18px;
}
.fact-group-title {
  font-size: 14px;
  font-weight: 700;
  color: #1e293b;
}
.fact-items {
  display: grid;
  gap: 10px;
}
.fact-item-card {
  border: 1px solid #f1f5f9;
  border-radius: 10px;
  padding: 12px;
  background: #f8fafc;
}
.fact-item-main {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  margin-bottom: 8px;
}
.fact-item-meta {
  display: flex;
  justify-content: flex-end;
}
.fact-subject { font-weight: 700; color: #1e293b; }
.fact-predicate {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  background: rgba(37,99,235,0.06);
  color: #2563eb;
  font-size: 12px;
  font-weight: 500;
}
.fact-object { color: #475569; line-height: 1.6; }
.fact-chapter-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  background: #f1f5f9;
  color: #64748b;
  font-size: 11px;
}
.chapter-type-tag {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  background: rgba(37,99,235,0.08);
  color: #2563eb;
}

.summary-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.summary-card {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  overflow: hidden;
  transition: border-color 0.2s;
}
.summary-card:hover { border-color: #cbd5e1; }
.summary-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 18px;
  background: #f8fafc;
  border-bottom: 1px solid #f1f5f9;
}
.summary-chapter {
  font-size: 13px;
  font-weight: 700;
  color: #2563eb;
}
.summary-title {
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
  flex: 1;
}
.summary-body {
  padding: 14px 18px;
}
.summary-row {
  display: flex;
  gap: 12px;
  padding: 6px 0;
  font-size: 13px;
  color: #475569;
  line-height: 1.6;
}
.summary-row + .summary-row { border-top: 1px solid #f8fafc; }
.summary-label {
  font-size: 12px;
  color: #94a3b8;
  min-width: 72px;
  flex-shrink: 0;
}
.summary-footer {
  padding: 8px 18px;
  border-top: 1px solid #f1f5f9;
}
.mood-tag {
  display: inline-block;
  padding: 2px 10px;
  border-radius: 4px;
  font-size: 11px;
  background: rgba(139,92,246,0.08);
  color: #8b5cf6;
}

.foundation-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.foundation-card {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  overflow: hidden;
  transition: border-color 0.2s;
}
.foundation-card:hover { border-color: #cbd5e1; }
.foundation-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 18px;
  cursor: pointer;
  transition: background 0.2s;
}
.foundation-header:hover { background: #f8fafc; }
.foundation-header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}
.foundation-icon { font-size: 18px; }
.foundation-name {
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
}
.foundation-header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}
.foundation-time { font-size: 12px; color: #94a3b8; }
.foundation-toggle { font-size: 12px; color: #64748b; }
.foundation-body {
  padding: 0 18px 18px;
  border-top: 1px solid #f1f5f9;
}
.foundation-content {
  padding-top: 16px;
  color: #334155;
  font-size: 14px;
  line-height: 1.8;
}
.foundation-content :deep(h1),
.foundation-content :deep(h2),
.foundation-content :deep(h3) { color: #1e293b; margin: 16px 0 8px; }
.foundation-content :deep(p) { margin: 8px 0; }
.foundation-content :deep(ul),
.foundation-content :deep(ol) { padding-left: 20px; margin: 8px 0; }
.foundation-content :deep(code) {
  background: #f1f5f9;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 13px;
}
.foundation-content :deep(pre) {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 16px;
  overflow-x: auto;
}
.foundation-content :deep(blockquote) {
  border-left: 3px solid #2563eb;
  padding-left: 16px;
  color: #64748b;
  margin: 12px 0;
}

.snapshot-timeline {
  display: flex;
  flex-direction: column;
  gap: 0;
  position: relative;
  padding-left: 24px;
}
.snapshot-timeline::before {
  content: '';
  position: absolute;
  left: 7px;
  top: 8px;
  bottom: 8px;
  width: 2px;
  background: #e2e8f0;
}
.snapshot-card {
  position: relative;
  margin-bottom: 12px;
}
.snapshot-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s;
}
.snapshot-header:hover { border-color: #2563eb; background: #f8fafc; }
.snapshot-header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}
.snapshot-dot {
  position: absolute;
  left: -20px;
  top: 18px;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #2563eb;
  border: 2px solid #fff;
  box-shadow: 0 0 0 2px #2563eb;
}
.snapshot-title {
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
}
.snapshot-header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}
.snapshot-time {
  font-size: 12px;
  color: #94a3b8;
}
.snapshot-toggle {
  font-size: 12px;
  color: #64748b;
}
.snapshot-body {
  padding: 16px;
  background: #fff;
  border: 1px solid #e2e8f0;
  border-top: none;
  border-radius: 0 0 10px 10px;
}
.snapshot-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}
.snapshot-grid + .snapshot-grid { margin-top: 16px; }
.snapshot-section { }
.snapshot-section h4 {
  font-size: 13px;
  color: #475569;
  margin-bottom: 8px;
}
.snapshot-section pre {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 14px;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  max-height: 300px;
  overflow-y: auto;
  color: #334155;
}

.settings-section { max-width: 480px; }
.settings-card {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  overflow: hidden;
}
.setting-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  cursor: pointer;
  transition: background 0.2s;
}
.setting-item:hover { background: #f8fafc; }
.setting-item + .setting-item { border-top: 1px solid #f1f5f9; }
.setting-label { font-size: 15px; color: #1e293b; margin-bottom: 4px; }
.setting-desc { font-size: 13px; color: #94a3b8; }
.setting-arrow { color: #94a3b8; font-size: 16px; }

.add-btn {
  padding: 8px 20px;
  border: 1px solid #2563eb;
  border-radius: 6px;
  background: transparent;
  color: #2563eb;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}
.add-btn:hover { background: rgba(37,99,235,0.08); }

.view-btn {
  padding: 4px 12px;
  border: 1px solid #e2e8f0;
  border-radius: 4px;
  background: transparent;
  color: #64748b;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}
.view-btn:hover { color: #2563eb; border-color: #2563eb; }

.type-tag {
  display: inline-block;
  padding: 2px 10px;
  border-radius: 4px;
  font-size: 12px;
}
.type-tag.public { background: rgba(37,99,235,0.08); color: #2563eb; }
.type-tag.private { background: #f1f5f9; color: #64748b; }

.action-group { display: flex; gap: 6px; }
.table-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}
.action-btn {
  padding: 4px 12px;
  border: 1px solid #e2e8f0;
  border-radius: 4px;
  background: transparent;
  color: #64748b;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}
.action-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.action-btn:hover { color: #2563eb; border-color: #2563eb; }
.action-btn.danger:hover { color: #dc2626; border-color: #dc2626; }
.action-btn.sm { padding: 2px 8px; font-size: 11px; }
.no-action { color: #94a3b8; font-size: 13px; }

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}
.modal {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 16px;
  padding: 32px;
  width: 520px;
  max-height: 80vh;
  overflow-y: auto;
  box-shadow: 0 8px 40px rgba(0,0,0,0.12);
}
.md-modal { width: 640px; }
.chapter-modal { width: 700px; }
.modal h3 {
  color: #1e293b;
  font-size: 18px;
  margin-bottom: 24px;
}

.progress-modal { width: min(960px, 92vw); }
.run-summary {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  padding: 12px 14px;
  background: #eff6ff;
  border: 1px solid #bfdbfe;
  border-radius: 10px;
  color: #1e3a8a;
  font-size: 13px;
}
.progress-steps {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.progress-step {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 10px 14px;
  border-radius: 10px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  transition: all 0.3s;
}
.progress-step.active {
  background: rgba(37,99,235,0.05);
  border-color: #2563eb;
}
.progress-step.done {
  background: rgba(22,163,74,0.05);
  border-color: #16a34a;
}
.progress-step.failed {
  background: rgba(220,38,38,0.05);
  border-color: rgba(220,38,38,0.35);
}
.step-icon {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  flex-shrink: 0;
  background: #e2e8f0;
  color: #64748b;
}
.progress-step.active .step-icon {
  background: #2563eb;
  color: #fff;
}
.progress-step.done .step-icon {
  background: #16a34a;
  color: #fff;
}
.progress-step.failed .step-icon {
  background: #dc2626;
  color: #fff;
}
.spinner {
  width: 14px;
  height: 14px;
  border: 2px solid rgba(255,255,255,0.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
.step-info { flex: 1; }
.step-label { font-size: 13px; color: #1e293b; font-weight: 500; }
.step-desc { font-size: 12px; color: #64748b; margin-top: 2px; line-height: 1.5; }
.step-msg { font-size: 12px; color: #2563eb; margin-top: 6px; font-weight: 500; }
.progress-error {
  margin-top: 16px;
  padding: 12px;
  background: rgba(220,38,38,0.06);
  border: 1px solid rgba(220,38,38,0.2);
  border-radius: 8px;
  color: #dc2626;
  font-size: 13px;
}
.progress-actions {
  display: flex;
  gap: 12px;
  margin-top: 16px;
}
.progress-complete {
  margin-top: 20px;
  text-align: center;
}
.progress-complete p {
  font-size: 15px;
  color: #16a34a;
  margin-bottom: 16px;
}

.md-content {
  color: #334155;
  font-size: 14px;
  line-height: 1.8;
}
.md-content :deep(h1),
.md-content :deep(h2),
.md-content :deep(h3) { color: #1e293b; margin: 16px 0 8px; }
.md-content :deep(p) { margin: 8px 0; }
.md-content :deep(ul),
.md-content :deep(ol) { padding-left: 20px; margin: 8px 0; }
.md-content :deep(code) {
  background: #f1f5f9;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 13px;
}
.md-content :deep(pre) {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 16px;
  overflow-x: auto;
}
.md-content :deep(blockquote) {
  border-left: 3px solid #2563eb;
  padding-left: 16px;
  color: #64748b;
  margin: 12px 0;
}
.md-content :deep(.fm-table-wrap) {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  padding: 16px;
  margin-bottom: 20px;
}
.md-content :deep(.fm-table) {
  width: 100%;
  border-collapse: collapse;
}
.md-content :deep(.fm-table td) {
  padding: 8px 12px;
  font-size: 13px;
  border-bottom: 1px solid #e2e8f0;
}
.md-content :deep(.fm-table tr:last-child td) { border-bottom: none; }
.md-content :deep(.fm-key) {
  color: #2563eb;
  font-weight: 600;
  white-space: nowrap;
  width: 1%;
  padding-right: 20px !important;
}
.md-content :deep(.fm-val) {
  color: #334155;
  word-break: break-all;
}

.chapter-content {
  color: #334155;
  font-size: 15px;
  line-height: 2;
  white-space: pre-wrap;
  max-height: 60vh;
  overflow-y: auto;
}

.form-group { margin-bottom: 16px; }
.form-group label {
  display: block;
  color: #475569;
  font-size: 13px;
  margin-bottom: 6px;
}
.form-group input,
.form-group textarea {
  width: 100%;
  padding: 10px 14px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #f8fafc;
  color: #1e293b;
  font-size: 14px;
  outline: none;
  transition: border-color 0.2s;
  box-sizing: border-box;
}
.form-group input:focus,
.form-group textarea:focus { border-color: #2563eb; }
.form-group textarea { resize: vertical; font-family: inherit; }
.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 24px;
}
.cancel-btn {
  padding: 8px 20px;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  background: transparent;
  color: #64748b;
  font-size: 13px;
  cursor: pointer;
}
.cancel-btn:hover { color: #1e293b; }
.save-btn {
  padding: 8px 20px;
  border: none;
  border-radius: 6px;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  color: #fff;
  font-size: 13px;
  cursor: pointer;
}
.save-btn:hover:not(:disabled) { opacity: 0.9; }
.save-btn:disabled { opacity: 0.5; cursor: not-allowed; }

@media (max-width: 1100px) {
  .write-container {
    flex-direction: column;
  }
  .sidebar {
    width: 100%;
    border-right: none;
    border-bottom: 1px solid #e2e8f0;
    padding: 16px 0;
  }
  .sidebar-nav {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    padding: 16px 20px 0;
  }
  .nav-item {
    border-left: none;
    border: 1px solid #e2e8f0;
    border-radius: 999px;
    padding: 8px 14px;
  }
  .nav-item:hover,
  .nav-item.active {
    border-left-color: transparent;
    border-color: #2563eb;
    background: rgba(37,99,235,0.08);
  }
  .sidebar-footer {
    padding: 12px 20px 0;
  }
  .topbar {
    padding: 16px 20px;
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }
  .content {
    padding: 24px 16px;
  }
  .write-controls {
    flex-direction: column;
    align-items: stretch;
  }
  .control-group input {
    width: 100%;
  }
  .book-meta {
    gap: 16px;
  }
  .book-meta-actions {
    margin-left: 0;
    width: 100%;
    justify-content: flex-start;
  }
  .state-grid,
  .snapshot-grid {
    grid-template-columns: 1fr;
  }
  .action-group,
  .table-actions,
  .chapter-actions,
  .progress-actions {
    flex-wrap: wrap;
  }
}

@media (max-width: 640px) {
  .content {
    padding: 16px 12px;
  }
  .empty-state {
    padding: 48px 16px;
  }
  .book-tabs {
    overflow-x: auto;
  }
  .book-tab {
    white-space: nowrap;
    flex: 0 0 auto;
  }
  .chapter-item {
    align-items: flex-start;
    flex-direction: column;
    gap: 10px;
  }
  .chapter-meta {
    flex-wrap: wrap;
  }
  .truth-card,
  .write-panel,
  .book-meta {
    padding: 14px;
  }
  .data-table th,
  .data-table td {
    padding: 12px 14px;
    white-space: nowrap;
  }
  .modal,
  .md-modal,
  .chapter-modal,
  .progress-modal {
    width: min(94vw, 960px);
    padding: 20px 16px;
  }
}
</style>
