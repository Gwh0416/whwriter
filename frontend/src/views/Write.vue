<template>
  <div class="write-container">
    <aside class="sidebar">
      <div class="sidebar-logo">
        <h2>文豪写作</h2>
      </div>
      <nav class="sidebar-nav">
        <a href="#" class="nav-item" :class="{ active: tab === 'books' }" @click.prevent="switchTab('books')">书籍管理</a>
        <a href="#" class="nav-item" :class="{ active: tab === 'radar' }" @click.prevent="switchTab('radar')">我的雷达</a>
        <div v-if="tab === 'radar'" class="nav-subitems">
          <a
            v-for="item in radarSubTabs"
            :key="item.key"
            href="#"
            class="nav-subitem"
            :class="{ active: radarSubTab === item.key }"
            @click.prevent="switchRadarSubTab(item.key)"
          >
            {{ item.label }}
          </a>
        </div>
        <a href="#" class="nav-item" :class="{ active: tab === 'settings' }" @click.prevent="switchTab('settings')">系统设置</a>
      </nav>
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
            <button class="book-tab" :class="{ active: bookTab === 'truth' }" @click="bookTab = 'truth'; loadTruthFiles()">记忆文件</button>
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
                  <label>本章提示（可选）</label>
                  <textarea
                    v-model="writeInput"
                    placeholder="如：承接上一章冲突，让主角在公司会议上反击；女主只旁观不表态；章尾留下新订单被截胡的悬念"
                    rows="3"
                    :disabled="writing"
                  ></textarea>
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
                <button v-for="tt in truthTabs" :key="tt.key" class="truth-tab" :class="{ active: truthSubTab === tt.key }" @click="switchTruthTab(tt.key)">
                  <span class="truth-tab-icon">{{ tt.icon }}</span>
                  {{ tt.label }}
                  <span class="truth-count" v-if="truthCounts[tt.key] > 0">{{ truthCounts[tt.key] }}</span>
                </button>
              </div>

              <div v-if="truthSubTab === 'wiki'" class="truth-panel">
                <div class="wiki-toolbar">
                  <input
                    v-model="wikiQuery"
                    class="wiki-search"
                    placeholder="搜索实体或别名"
                    @keyup.enter="loadWikiEntities()"
                  />
                  <select v-model="wikiType" class="wiki-type-select" @change="loadWikiEntities()">
                    <option v-for="option in wikiTypeOptions" :key="option.value" :value="option.value">
                      {{ option.label }}
                    </option>
                  </select>
                  <button class="action-btn" :disabled="wikiLoading" @click="loadWikiEntities()">
                    {{ wikiLoading ? '检索中...' : '检索' }}
                  </button>
                </div>

                <div v-if="wikiLoading && !wikiEntities.length" class="truth-loading">加载 Wiki...</div>
                <div v-else-if="!wikiEntities.length" class="no-data">
                  <div class="no-data-icon">◫</div>
                  <p>暂无 Wiki 实体</p>
                  <span>初始化或完成章节写作后，系统会建立实体与关系索引。</span>
                </div>
                <div v-else class="wiki-layout">
                  <aside class="wiki-entity-nav">
                    <div class="wiki-result-count">{{ wikiTotal }} 个实体</div>
                    <button
                      v-for="entity in wikiEntities"
                      :key="entity.id"
                      class="wiki-entity-row"
                      :class="{ active: selectedWikiEntityID === entity.id }"
                      @click="selectWikiEntity(entity.id)"
                    >
                      <span class="wiki-entity-type">{{ wikiEntityTypeLabel(entity.entity_type) }}</span>
                      <span class="wiki-entity-name">{{ entity.canonical_name }}</span>
                      <span v-if="entity.status !== 'active'" class="wiki-inactive">历史</span>
                    </button>
                  </aside>

                  <section class="wiki-page">
                    <div v-if="!wikiPage" class="no-data wiki-page-empty">
                      <p>选择一个实体查看 Wiki 页面</p>
                    </div>
                    <template v-else>
                      <header class="wiki-page-header">
                        <div>
                          <span class="wiki-page-type">{{ wikiEntityTypeLabel(wikiPage.entity.entity_type) }}</span>
                          <h3>{{ wikiPage.entity.canonical_name }}</h3>
                        </div>
                        <span class="wiki-chapter-range">
                          第{{ wikiPage.entity.first_seen_chapter || 0 }}章
                          <template v-if="wikiPage.entity.last_seen_chapter">至第{{ wikiPage.entity.last_seen_chapter }}章</template>
                        </span>
                      </header>
                      <p v-if="wikiPage.entity.summary" class="wiki-summary">{{ wikiPage.entity.summary }}</p>
                      <div v-if="wikiDisplayAliases.length" class="wiki-aliases">
                        <span class="wiki-section-label">别名</span>
                        <span v-for="alias in wikiDisplayAliases" :key="alias.id" class="wiki-alias">{{ alias.alias }}</span>
                      </div>

                      <section class="wiki-section-block">
                        <div class="wiki-section-heading">
                          <h4>关系时间线</h4>
                          <span>{{ wikiPage.relations?.length || 0 }}</span>
                        </div>
                        <div v-if="!wikiPage.relations?.length" class="wiki-empty-line">暂无关系</div>
                        <div v-else class="wiki-timeline">
                          <article v-for="relation in wikiPage.relations" :key="relation.id" class="wiki-timeline-item">
                            <div class="wiki-timeline-marker"></div>
                            <div class="wiki-timeline-content">
                              <div class="wiki-relation-main">
                                <strong>{{ wikiRelationLabel(relation) }}</strong>
                                <span>{{ wikiRelationTarget(relation) }}</span>
                              </div>
                              <div class="wiki-relation-meta">
                                <span>第{{ relation.valid_from_chapter }}章起</span>
                                <span v-if="relation.valid_until_chapter">至第{{ relation.valid_until_chapter }}章</span>
                                <span v-else class="wiki-current-tag">当前有效</span>
                              </div>
                              <blockquote
                                v-for="evidence in wikiEvidenceByRelation.get(relation.id) || []"
                                :key="evidence.id"
                                class="wiki-evidence"
                              >
                                “{{ evidence.quote }}”
                                <span>第{{ evidence.chapter_number }}章 · {{ evidence.start_offset }}-{{ evidence.end_offset }}</span>
                              </blockquote>
                            </div>
                          </article>
                        </div>
                      </section>

                      <section class="wiki-section-block">
                        <div class="wiki-section-heading">
                          <h4>事件时间线</h4>
                          <span>{{ wikiPage.events?.length || 0 }}</span>
                        </div>
                        <div v-if="!wikiPage.events?.length" class="wiki-empty-line">暂无关联事件</div>
                        <div v-else class="wiki-events">
                          <article v-for="event in wikiPage.events" :key="event.id" class="wiki-event-row">
                            <div class="wiki-event-chapter">第{{ event.chapter_number }}章</div>
                            <div class="wiki-event-content">
                              <div class="wiki-event-title">{{ event.title }}</div>
                              <p>{{ event.summary }}</p>
                              <div class="wiki-event-meta">
                                <span v-if="event.location_name">{{ event.location_name }}</span>
                                <span v-if="event.participants?.length">
                                  {{ wikiParticipantNames(event) }}
                                </span>
                              </div>
                            </div>
                          </article>
                        </div>
                      </section>
                    </template>
                  </section>
                </div>
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
                    <button class="action-btn" @click="openGenreModal(g)">编辑</button>
                    <button class="action-btn danger" @click="deleteGenre(g.id)">删除</button>
                  </td>
                </tr>
                <tr v-if="genres.length === 0">
                  <td colspan="3" class="empty-row">暂无可用的题材</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <MyRadar v-if="tab === 'radar'" :active-tab="radarSubTab" />

        <div v-if="tab === 'settings'" class="settings-section">
          <section class="settings-hero">
            <div>
              <h2>系统设置</h2>
              <p>维护本地基础数据、模型厂商和 Token 用量。这里不再区分管理员和普通用户，所有设置都属于当前本地工作台。</p>
            </div>
            <div class="token-summary">
              <span>总 Token 消耗</span>
              <strong>{{ totalTokenUsage.toLocaleString() }}</strong>
              <em>输入 {{ totalPromptTokens.toLocaleString() }} · 输出 {{ totalCompletionTokens.toLocaleString() }} · 缓存 {{ totalCachedTokens.toLocaleString() }}</em>
            </div>
          </section>

          <div class="settings-grid">
            <section class="settings-card">
              <div class="settings-card-head">
                <div>
                  <h3>基础数据</h3>
                  <p>初始化平台、默认兼容题材和番茄官方标签。写作维度以标签为准。</p>
                </div>
              </div>
              <button class="save-btn" :disabled="initializing" @click="initializeBaseData">{{ initializing ? '初始化中...' : '初始化基础数据' }}</button>
              <div v-if="initMessage" class="settings-message">{{ initMessage }}</div>
            </section>

            <section class="settings-card">
              <div class="settings-card-head">
                <div>
                  <h3>Token 用量</h3>
                  <p>按模型统计当前本地工作台的调用消耗。</p>
                </div>
              </div>
              <div v-if="modelLoading" class="empty-row">加载中...</div>
              <div v-else-if="tokenDetails.length" class="token-row">
                <span v-for="d in tokenDetails" :key="d.id" class="token-pill">
                  <strong>{{ d.provider }} / {{ d.model_name }}</strong>
                  总量 {{ (d.token_usage || 0).toLocaleString() }}
                  <em>输入 {{ (d.prompt_tokens || 0).toLocaleString() }} · 输出 {{ (d.completion_tokens || 0).toLocaleString() }} · 缓存 {{ (d.cached_tokens || 0).toLocaleString() }}</em>
                </span>
              </div>
              <div v-else class="empty-settings">暂无 Token 记录</div>
              <div v-if="agentTokenDetails.length" class="agent-token-row">
                <div v-for="d in agentTokenDetails" :key="d.agent_name" class="agent-token-pill">
                  <strong>{{ agentNameLabel(d.agent_name) }}</strong>
                  <span>总量 {{ (d.total_tokens || 0).toLocaleString() }}</span>
                  <em>输入 {{ (d.prompt_tokens || 0).toLocaleString() }} · 输出 {{ (d.completion_tokens || 0).toLocaleString() }} · 缓存 {{ (d.cached_tokens || 0).toLocaleString() }}</em>
                </div>
              </div>
            </section>
          </div>

          <section class="settings-card model-settings-card">
            <div class="settings-card-head model-settings-head">
              <div>
                <h3>模型厂商</h3>
                <p>配置 OpenAI 兼容模型服务，测试连接后导入可用模型，并设置默认模型。</p>
              </div>
              <button class="add-btn" @click="openProviderModal()">+ 新增厂商</button>
            </div>

            <div v-if="modelLoading" class="empty-row">加载中...</div>
            <div v-else-if="!llmConfigs.length" class="empty-settings">暂无模型厂商配置</div>
            <div v-else class="provider-grid">
              <div v-for="cfg in llmConfigs" :key="cfg.id" class="provider-card">
                <div class="provider-header">
                  <div>
                    <div class="provider-name">{{ cfg.label || cfg.provider }}</div>
                    <div class="provider-url">{{ cfg.base_url || '未配置 Base URL' }}</div>
                  </div>
                  <div class="action-group">
                    <button class="action-btn" @click="openProviderModal(cfg)">编辑</button>
                    <button class="action-btn" @click="openTestConnection(cfg)">测试连接</button>
                    <button class="action-btn danger" @click="deleteProvider(cfg.id)">删除</button>
                  </div>
                </div>

                <div v-if="testingId === cfg.id" class="test-box" :class="{ success: testSuccess }">
                  <div v-if="testLoading">正在连接模型服务...</div>
                  <div v-else-if="testSuccess">
                    <div class="test-title">连接成功，发现 {{ testModels.length }} 个模型</div>
                    <div class="model-check-list">
                      <label v-for="m in testModels" :key="m" class="model-check">
                        <input type="checkbox" :value="m" v-model="selectedTestModels" />
                        <span>{{ m }}</span>
                      </label>
                    </div>
                    <button class="save-btn sm" :disabled="selectedTestModels.length === 0" @click="importModels(cfg.id)">导入选中模型</button>
                  </div>
                  <div v-else>{{ testError }}</div>
                </div>

                <div class="model-list">
                  <div v-for="m in cfg.models || []" :key="m.id" class="model-item" :class="{ disabled: !m.is_enabled }">
                    <div class="model-title">
                      <strong>{{ m.model_name }}</strong>
                      <span v-if="m.is_default" class="default-badge">默认</span>
                      <span v-if="!m.is_enabled" class="disabled-badge">已禁用</span>
                    </div>
                    <div class="model-actions">
                      <span>{{ (m.token_usage || 0).toLocaleString() }} tokens</span>
                      <button class="action-btn sm" @click="toggleModel(m)">{{ m.is_enabled ? '禁用' : '启用' }}</button>
                      <button v-if="!m.is_default" class="action-btn sm" @click="setDefaultModel(m.id)">设为默认</button>
                    </div>
                  </div>
                  <div v-if="!cfg.models?.length" class="empty-settings">暂无模型，请先测试连接并导入</div>
                </div>
              </div>
            </div>
          </section>
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
        <div v-if="writeRunTokenSummary.total_tokens > 0" class="run-token-card">
          <div class="run-token-head">
            <span>本章 Token 消耗</span>
            <strong>{{ formatTokenCompact(writeRunTokenSummary.total_tokens) }}</strong>
          </div>
          <div class="run-token-meta">
            输入 {{ formatTokenCompact(writeRunTokenSummary.prompt_tokens) }} ·
            输出 {{ formatTokenCompact(writeRunTokenSummary.completion_tokens) }} ·
            缓存 {{ formatTokenCompact(writeRunTokenSummary.cached_tokens) }}
          </div>
          <div class="run-token-stages">
            <div v-for="row in writeRunStageTokenRows" :key="row.stage" class="run-token-stage">
              <span>{{ row.label }}</span>
              <strong>{{ formatTokenCompact(row.total_tokens) }}</strong>
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
            不写了
          </button>
          <button class="cancel-btn" :disabled="retryingRun" @click="retryCurrentRun('restart')">
            {{ retryingRun ? '重写中...' : '整章重写' }}
          </button>
          <button class="save-btn" :disabled="retryingRun" @click="retryCurrentRun('resume_failed_stage')">
            {{ retryingRun ? '重试中...' : '从失败阶段重试' }}
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

    <div v-if="showProviderModal" class="modal-overlay" @click.self="showProviderModal = false">
      <div class="modal">
        <h3>{{ editingProvider ? '编辑模型厂商' : '新增模型厂商' }}</h3>
        <div class="form-group">
          <label>厂商标识</label>
          <input v-model="providerForm.provider" placeholder="如 openai / deepseek" />
        </div>
        <div class="form-group">
          <label>显示名称</label>
          <input v-model="providerForm.label" placeholder="如 OpenAI" />
        </div>
        <div class="form-group">
          <label>Base URL</label>
          <input v-model="providerForm.base_url" placeholder="https://api.openai.com/v1" />
        </div>
        <div class="form-group">
          <label>API Key</label>
          <input v-model="providerForm.api_key" type="password" :placeholder="editingProvider ? '留空则不修改' : '请输入 API Key'" />
        </div>
        <div class="modal-actions">
          <button class="cancel-btn" @click="showProviderModal = false">取消</button>
          <button class="save-btn" :disabled="providerSaving" @click="saveProvider">{{ providerSaving ? '保存中...' : '保存' }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, reactive, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { marked } from 'marked'
import yaml from 'js-yaml'
import MyRadar from './MyRadar.vue'

const route = useRoute()
const router = useRouter()
const workspaceTabs = new Set(['books', 'radar', 'settings'])
const radarTabs = new Set(['scan', 'profiles', 'rules', 'intros'])
const tab = ref(normalizeWorkspaceTab(route.query.tab))
const radarSubTab = ref(normalizeRadarTab(route.query.radar))
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
const wikiLoading = ref(false)
const wikiEntities = ref([])
const wikiTotal = ref(0)
const wikiQuery = ref('')
const wikiType = ref('')
const selectedWikiEntityID = ref(0)
const wikiPage = ref(null)
const wikiTypeOptions = [
  { value: '', label: '全部类型' },
  { value: 'character', label: '角色' },
  { value: 'place', label: '地点' },
  { value: 'item', label: '物品' },
  { value: 'organization', label: '组织' },
  { value: 'event', label: '事件' },
  { value: 'hook', label: '伏笔' },
  { value: 'rule', label: '规则' },
  { value: 'concept', label: '概念' },
]
const truthTabs = [
  { key: 'state', label: '当前状态', icon: '🧭' },
  { key: 'wiki', label: 'Wiki', icon: '◫' },
  { key: 'characters', label: '人物', icon: '👤' },
  { key: 'facts', label: '设定', icon: '📋' },
  { key: 'hooks', label: '伏笔', icon: '🪝' },
  { key: 'summaries', label: '章节摘要', icon: '📝' },
  { key: 'foundations', label: '基础文件', icon: '📐' },
  { key: 'snapshots', label: '快照', icon: '📸' },
]
const truthCounts = computed(() => ({
  state: truthData.value.book_state ? 1 : 0,
  wiki: wikiTotal.value,
  characters: truthData.value.characters?.length || 0,
  facts: truthData.value.facts?.length || 0,
  hooks: truthData.value.hooks?.length || 0,
  summaries: truthData.value.summaries?.length || 0,
  foundations: truthData.value.foundations?.length || 0,
  snapshots: truthData.value.snapshots?.length || 0,
}))
const wikiEvidenceByRelation = computed(() => {
  const grouped = new Map()
  for (const evidence of wikiPage.value?.relation_evidence || []) {
    if (!grouped.has(evidence.relation_id)) grouped.set(evidence.relation_id, [])
    grouped.get(evidence.relation_id).push(evidence)
  }
  return grouped
})
const wikiDisplayAliases = computed(() => (
  (wikiPage.value?.aliases || []).filter(alias => !alias.is_canonical)
))

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
const activeWriteRunStorageKey = 'whwriter.activeWriteRun'
const progressSteps = reactive([
  { key: 'loading', label: '加载书籍信息', desc: '创建本次写作任务，锁定书籍并准备进入创作链路。', status: 'pending', msg: '' },
  { key: 'context', label: '构建上下文', desc: '整理当前书籍的状态、设定、伏笔和历史章节信息。', status: 'pending', msg: '' },
  { key: 'planning', label: 'Planner 规划本章', desc: '生成本章目标、推进点和关键冲突的写作计划。', status: 'pending', msg: '' },
  { key: 'writing', label: 'Writer 创作正文', desc: '根据规划与上下文生成章节正文和结构化分段结果。', status: 'pending', msg: '' },
  { key: 'parsing', label: '解析 Writer 输出', desc: '把 Writer 返回结果拆成标题、正文和后续结算所需片段。', status: 'pending', msg: '' },
  { key: 'auditing', label: 'Auditor 审查结构', desc: '检查章节结构、推进节奏和状态一致性是否合理。', status: 'pending', msg: '' },
  { key: 'revising', label: 'Reviser 修订正文', desc: '当审查不过时，按问题清单修订正文和状态片段。', status: 'pending', msg: '' },
  { key: 'polishing', label: 'Polisher 润色文稿', desc: '在不改变既有设定的前提下优化表达、节奏和可读性。', status: 'pending', msg: '' },
  { key: 'extracting', label: '提取记忆文件', desc: '结算本章带来的状态变化，并抽取人物、设定、伏笔等记忆数据。', status: 'pending', msg: '' },
  { key: 'snapshot', label: '保存章节快照', desc: '统一提交章节和记忆状态，并生成用于回滚的快照。', status: 'pending', msg: '' },
])
const writeRunStageTokenRows = computed(() => {
  const byStage = new Map()
  for (const stage of writeRunStages.value || []) {
    const meta = stageMeta(stage)
    const summary = sumTokenRows(meta.token_summary || [])
    if (summary.total_tokens <= 0 && summary.prompt_tokens <= 0 && summary.completion_tokens <= 0 && summary.cached_tokens <= 0) continue
    if (!byStage.has(stage.stage)) {
      byStage.set(stage.stage, {
        stage: stage.stage,
        label: progressStepLabel(stage.stage),
        prompt_tokens: 0,
        completion_tokens: 0,
        cached_tokens: 0,
        total_tokens: 0,
      })
    }
    addTokenSummary(byStage.get(stage.stage), summary)
  }
  return Array.from(byStage.values())
})
const writeRunTokenSummary = computed(() => {
  const total = { prompt_tokens: 0, completion_tokens: 0, cached_tokens: 0, total_tokens: 0 }
  for (const row of writeRunStageTokenRows.value) addTokenSummary(total, row)
  return total
})

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
const initializing = ref(false)
const initMessage = ref('')
const totalTokenUsage = ref(0)
const totalPromptTokens = ref(0)
const totalCompletionTokens = ref(0)
const totalCachedTokens = ref(0)
const tokenDetails = ref([])
const agentTokenDetails = ref([])
const modelLoading = ref(false)
const showProviderModal = ref(false)
const editingProvider = ref(null)
const providerSaving = ref(false)
const providerForm = ref({ provider: '', label: '', base_url: '', api_key: '' })
const testingId = ref(null)
const testLoading = ref(false)
const testSuccess = ref(false)
const testError = ref('')
const testModels = ref([])
const selectedTestModels = ref([])


const tabTitle = computed(() => {
  const titles = { books: '书籍管理', radar: '我的雷达', settings: '系统设置' }
  return titles[tab.value] || ''
})
const radarSubTabs = [
  { key: 'scan', label: '扫描书籍' },
  { key: 'profiles', label: '聚合画像' },
  { key: 'rules', label: '写作规则' },
  { key: 'intros', label: '简介雷达' },
]

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

function normalizeWorkspaceTab(raw) {
  const value = Array.isArray(raw) ? raw[0] : raw
  return workspaceTabs.has(value) ? value : 'books'
}

function normalizeRadarTab(raw) {
  const value = Array.isArray(raw) ? raw[0] : raw
  return radarTabs.has(value) ? value : 'scan'
}

function statusLabel(s) {
  return statusLabels[s] || s
}

function roleLabel(r) {
  const labels = { protagonist: '主角', major: '重要角色', minor: '次要角色' }
  return labels[r] || r
}

function wikiEntityTypeLabel(type) {
  const labels = {
    character: '角色',
    place: '地点',
    item: '物品',
    organization: '组织',
    event: '事件',
    hook: '伏笔',
    rule: '规则',
    concept: '概念',
  }
  return labels[type] || type
}

function wikiRelationLabel(relation) {
  if (!wikiPage.value?.entity) return relation.predicate
  return relation.subject_entity_id === wikiPage.value.entity.id
    ? relation.predicate
    : `${relation.subject_name} · ${relation.predicate}`
}

function wikiRelationTarget(relation) {
  if (!wikiPage.value?.entity) return relation.object_name || relation.object_literal || '-'
  if (relation.subject_entity_id === wikiPage.value.entity.id) {
    return relation.object_name || relation.object_literal || '-'
  }
  return wikiPage.value.entity.canonical_name
}

function wikiParticipantNames(event) {
  return (event.participants || []).map(item => item.canonical_name).join('、')
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
  wikiEntities.value = []
  wikiTotal.value = 0
  wikiQuery.value = ''
  wikiType.value = ''
  selectedWikiEntityID.value = 0
  wikiPage.value = null
  writeInput.value = ''
  writeResult.value = null
  showChapterModal.value = false
  viewingChapter.value = null
  currentWriteRun.value = null
  writeRunStages.value = []
}

async function activateTab(nextTab, syncURL = true) {
  nextTab = workspaceTabs.has(nextTab) ? nextTab : 'books'
  tab.value = nextTab
  if (syncURL) {
    const query = { ...route.query }
    if (nextTab === 'books') {
      delete query.tab
      delete query.radar
    } else {
      query.tab = nextTab
      if (nextTab === 'radar') query.radar = radarSubTab.value
      else delete query.radar
    }
    await router.replace({ path: '/write', query })
  }
  if (nextTab !== 'books') {
    leaveBookView()
  }
  if (nextTab === 'settings') {
    await loadModelSettings()
  }
}

async function switchTab(nextTab) {
  await activateTab(nextTab, true)
}

async function switchRadarSubTab(nextTab) {
  radarSubTab.value = normalizeRadarTab(nextTab)
  if (tab.value !== 'radar') {
    await activateTab('radar', true)
    return
  }
  const query = { ...route.query, tab: 'radar', radar: radarSubTab.value }
  await router.replace({ path: '/write', query })
}

async function loadBooks() {
  const [booksRes, llmRes] = await Promise.all([
    fetch('/api/v1/books', { headers: { } }),
    fetch('/api/v1/llm-configs', { headers: { } }),
  ])
  if (booksRes.ok) books.value = await booksRes.json()
  if (llmRes.ok) llmConfigs.value = await llmRes.json()
}

onMounted(async () => {
  await loadBooks()
  if (tab.value === 'settings') {
    await loadModelSettings()
  }
  await restoreStoredWriteRun()
})

onUnmounted(() => {
  stopWriteRunPolling()
})

watch(() => route.query.tab, async (next) => {
  const nextTab = normalizeWorkspaceTab(next)
  if (nextTab !== tab.value) {
    await activateTab(nextTab, false)
  }
})

watch(() => route.query.radar, (next) => {
  radarSubTab.value = normalizeRadarTab(next)
})

async function loadGenres() {
  const res = await fetch('/api/v1/genres', { headers: { } })
  if (res.ok) genres.value = await res.json()
}

async function openBook(id) {
  const res = await fetch(`/api/v1/books/${id}`, { headers: { } })
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

function rememberWriteRun(run) {
  if (!run?.id) return
  localStorage.setItem(activeWriteRunStorageKey, JSON.stringify({
    run_id: run.id,
    book_id: run.book_id,
  }))
}

function forgetWriteRun() {
  localStorage.removeItem(activeWriteRunStorageKey)
}

async function restoreStoredWriteRun() {
  const raw = localStorage.getItem(activeWriteRunStorageKey)
  if (!raw || currentWriteRun.value?.id) return
  let stored = null
  try {
    stored = JSON.parse(raw)
  } catch {
    forgetWriteRun()
    return
  }
  if (!stored?.run_id) return
  try {
    const run = await refreshWriteRun(stored.run_id)
    if (!run) {
      forgetWriteRun()
      return
    }
    if (run.book_id && selectedBook.value?.id !== run.book_id) {
      await openBook(run.book_id)
      currentWriteRun.value = run
      await refreshWriteRun(run.id)
    }
    showProgressModal.value = true
    if (run.status === 'queued' || run.status === 'running') {
      writing.value = true
      await pollWriteRun(run.id)
    } else if (run.status === 'succeeded') {
      writing.value = false
      await finalizeWrite({
        chapter_number: run.target_chapter,
        title: currentWriteRun.value?.title || '',
      })
    } else {
      writing.value = false
    }
  } catch {
    forgetWriteRun()
  }
}

async function deleteBook(book, evt = null) {
  evt?.stopPropagation?.()
  if (!book?.id) return
  if (!confirm(`确定删除书籍《${book.title}》吗？这会删除该书的章节、记忆文件和运行产物，且不可恢复。`)) return

  deletingBookID.value = book.id
  try {
    const res = await fetch(`/api/v1/books/${book.id}`, {
      method: 'DELETE',
      headers: { },
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
      headers: { },
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
      headers: { },
    })
    if (res.ok) {
      truthData.value = await res.json()
      if (truthSubTab.value === 'wiki') await loadWikiEntities()
    }
  } finally {
    truthLoading.value = false
  }
}

async function switchTruthTab(nextTab) {
  truthSubTab.value = nextTab
  if (nextTab === 'wiki' && !wikiEntities.value.length) {
    await loadWikiEntities()
  }
}

async function loadWikiEntities() {
  if (!selectedBook.value) return
  wikiLoading.value = true
  try {
    const params = new URLSearchParams({ limit: '200' })
    if (wikiQuery.value.trim()) params.set('q', wikiQuery.value.trim())
    if (wikiType.value) params.set('type', wikiType.value)
    const res = await fetch(`/api/v1/books/${selectedBook.value.id}/wiki/entities?${params.toString()}`)
    if (!res.ok) {
      wikiEntities.value = []
      wikiTotal.value = 0
      wikiPage.value = null
      selectedWikiEntityID.value = 0
      return
    }
    const data = await res.json()
    wikiEntities.value = data.items || []
    wikiTotal.value = data.total || 0
    const selectedStillVisible = wikiEntities.value.some(entity => entity.id === selectedWikiEntityID.value)
    const nextID = selectedStillVisible ? selectedWikiEntityID.value : (wikiEntities.value[0]?.id || 0)
    if (nextID) {
      await selectWikiEntity(nextID)
    } else {
      selectedWikiEntityID.value = 0
      wikiPage.value = null
    }
  } finally {
    wikiLoading.value = false
  }
}

async function selectWikiEntity(entityID) {
  if (!selectedBook.value || !entityID) return
  selectedWikiEntityID.value = entityID
  wikiLoading.value = true
  try {
    const res = await fetch(`/api/v1/books/${selectedBook.value.id}/wiki/entities/${entityID}`)
    if (res.ok) {
      wikiPage.value = await res.json()
    } else {
      wikiPage.value = null
    }
  } finally {
    wikiLoading.value = false
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

function stageMeta(stage) {
  if (!stage?.output_payload) return {}
  try {
    const payload = JSON.parse(stage.output_payload)
    return payload?.meta || {}
  } catch {
    return {}
  }
}

function sumTokenRows(rows) {
  const total = { prompt_tokens: 0, completion_tokens: 0, cached_tokens: 0, total_tokens: 0 }
  for (const row of rows || []) addTokenSummary(total, row)
  return total
}

function addTokenSummary(target, row) {
  target.prompt_tokens += Number(row?.prompt_tokens || 0)
  target.completion_tokens += Number(row?.completion_tokens || 0)
  target.cached_tokens += Number(row?.cached_tokens || 0)
  target.total_tokens += Number(row?.total_tokens || 0)
}

function formatTokenCompact(value) {
  return Number(value || 0).toLocaleString()
}

function progressStepLabel(stageKey) {
  return progressSteps.find(step => step.key === stageKey)?.label || stageKey || '未知阶段'
}

function stageAttemptMsg(stage, base) {
  const meta = stageMeta(stage)
  const attempt = Number(meta.attempt || stage?.attempt || 1)
  const maxAttempts = Number(meta.max_attempts || 3)
  if (attempt <= 1 && base !== '失败') return base
  if (base === '失败') return `${base}（已尝试 ${attempt}/${maxAttempts} 次）`
  return `${base}（第 ${attempt}/${maxAttempts} 次）`
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
      const skippedMsg = stage.stage === 'revising' ? '审查通过，无需修订' : '已跳过'
      step.msg = stageAttemptMsg(stage, stage.status === 'skipped' ? skippedMsg : '已完成')
    } else if (stage.status === 'running') {
      step.status = 'active'
      step.msg = stageAttemptMsg(stage, '执行中')
    } else if (stage.status === 'failed' || stage.status === 'cancelled') {
      step.status = 'failed'
      step.msg = stage.status === 'cancelled' ? '已取消' : stageAttemptMsg(stage, '失败')
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
    headers: { },
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
    headers: { },
  })
  if (!res.ok) return
  const data = await res.json()
  if (!data.run) return
  showProgressModal.value = true
  writing.value = true
  currentWriteRun.value = data.run
  rememberWriteRun(data.run)
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
    rememberWriteRun(data.run)
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
    rememberWriteRun(data.run)
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
      headers: { },
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
    rememberWriteRun(data.run)
    await pollWriteRun(data.run.id)
  } finally {
    retryingRun.value = false
  }
}

async function abandonCurrentRun() {
  stopWriteRunPolling()
  forgetWriteRun()
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
      headers: { },
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
  forgetWriteRun()
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
      res = await fetch(`/api/v1/genres/${editingGenre.value.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body,
      })
    } else {
      res = await fetch('/api/v1/genres', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body,
      })
    }
    if (res.ok) {
      showGenreModal.value = false
      await loadModelSettings()
    }
  } finally {
    genreSaving.value = false
  }
}

async function deleteGenre(id) {
  if (!confirm('确定删除该题材？')) return
  const res = await fetch(`/api/v1/genres/${id}`, {
    method: 'DELETE',
  })
  if (res.ok) await loadGenres()
}

async function initializeBaseData() {
  initializing.value = true
  initMessage.value = ''
  try {
    const res = await fetch('/api/v1/initialize', { method: 'POST' })
    const data = await res.json().catch(() => ({}))
    initMessage.value = res.ok ? '基础题材和平台已初始化' : (data.message || data.error || '初始化失败')
    if (res.ok) {
      await loadGenres()
    }
  } finally {
    initializing.value = false
  }
}

async function loadModelSettings() {
  modelLoading.value = true
  try {
    const [configsRes, usageRes] = await Promise.all([
      fetch('/api/v1/llm-configs'),
      fetch('/api/v1/llm-configs/token-usage'),
    ])
    if (configsRes.ok) llmConfigs.value = await configsRes.json()
    if (usageRes.ok) {
      const data = await usageRes.json()
      totalTokenUsage.value = data.total_usage || 0
      totalPromptTokens.value = data.prompt_tokens || 0
      totalCompletionTokens.value = data.completion_tokens || 0
      totalCachedTokens.value = data.cached_tokens || 0
      tokenDetails.value = data.details || []
      agentTokenDetails.value = data.by_agent || []
    }
  } finally {
    modelLoading.value = false
  }
}

function agentNameLabel(name) {
  const labels = {
    planner: 'Planner',
    writer: 'Writer',
    auditor: 'Auditor',
    reviser: 'Reviser',
    polisher: 'Polisher',
    settler: 'Settler',
    architect: 'Architect',
    truth_extractor: '记忆提取',
    role_namer: '角色命名',
    radar_classifier: '雷达分类',
    radar_analyzer: '雷达画像',
    radar_synthesizer: '雷达聚合',
    chat: '通用调用',
  }
  return labels[name] || name || '未知链路'
}

function openProviderModal(cfg = null) {
  editingProvider.value = cfg
  providerForm.value = cfg
    ? { provider: cfg.provider, label: cfg.label, base_url: cfg.base_url, api_key: '' }
    : { provider: '', label: '', base_url: '', api_key: '' }
  showProviderModal.value = true
}

async function saveProvider() {
  providerSaving.value = true
  try {
    const body = JSON.stringify(providerForm.value)
    const res = editingProvider.value
      ? await fetch(`/api/v1/llm-configs/${editingProvider.value.id}`, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body })
      : await fetch('/api/v1/llm-configs', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body })
    if (res.ok) {
      showProviderModal.value = false
      await loadModelSettings()
      await loadBooks()
    }
  } finally {
    providerSaving.value = false
  }
}

async function deleteProvider(id) {
  if (!confirm('确定删除该模型厂商及其所有模型？')) return
  const res = await fetch(`/api/v1/llm-configs/${id}`, { method: 'DELETE' })
  if (res.ok) {
    await loadModelSettings()
    await loadBooks()
  }
}

async function openTestConnection(cfg) {
  testingId.value = cfg.id
  testLoading.value = true
  testSuccess.value = false
  testError.value = ''
  testModels.value = []
  selectedTestModels.value = []
  try {
    const body = JSON.stringify({ config_id: cfg.id, base_url: cfg.base_url, api_key: '' })
    const res = await fetch('/api/v1/llm-configs/test-connection', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body })
    const data = await res.json()
    if (data.success) {
      testSuccess.value = true
      testModels.value = data.models || []
    } else {
      testError.value = data.error || '连接失败'
    }
  } catch {
    testError.value = '网络错误'
  } finally {
    testLoading.value = false
  }
}

async function importModels(configID) {
  const models = selectedTestModels.value.map(name => ({ model_name: name, is_enabled: true, is_default: false }))
  const res = await fetch(`/api/v1/llm-configs/${configID}/models`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ models }),
  })
  if (res.ok) {
    testingId.value = null
    await loadModelSettings()
    await loadBooks()
  }
}

async function toggleModel(m) {
  const cfg = llmConfigs.value.find(c => c.id === m.llm_config_id)
  if (!cfg) return
  const body = JSON.stringify({
    models: cfg.models.map(x => ({
      model_name: x.model_name,
      is_enabled: x.id === m.id ? !m.is_enabled : x.is_enabled,
      is_default: x.is_default,
    })),
  })
  const res = await fetch(`/api/v1/llm-configs/${m.llm_config_id}/models`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body })
  if (res.ok) {
    await loadModelSettings()
    await loadBooks()
  }
}

async function setDefaultModel(id) {
  const res = await fetch(`/api/v1/llm-models/${id}/default`, { method: 'POST' })
  if (res.ok) {
    await loadModelSettings()
    await loadBooks()
  }
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
  flex: 0 0 220px;
  position: sticky;
  top: 0;
  height: 100vh;
  overflow-y: auto;
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
.nav-subitems {
  display: grid;
  gap: 4px;
  padding: 4px 12px 10px 28px;
}
.nav-subitem {
  display: block;
  padding: 8px 12px;
  border-radius: 8px;
  color: #64748b;
  text-decoration: none;
  font-size: 13px;
  font-weight: 600;
}
.nav-subitem:hover,
.nav-subitem.active {
  color: #2563eb;
  background: #eff6ff;
}
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
.control-group input,
.control-group textarea {
  padding: 8px 12px;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  background: #f8fafc;
  color: #1e293b;
  font-size: 13px;
  outline: none;
  font-family: inherit;
}
.control-group select:focus,
.control-group input:focus,
.control-group textarea:focus { border-color: #2563eb; }
.control-group input,
.control-group textarea { width: 360px; }
.control-group textarea {
  min-height: 72px;
  resize: vertical;
  line-height: 1.6;
}
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
.wiki-toolbar {
  display: grid;
  grid-template-columns: minmax(220px, 1fr) 160px auto;
  gap: 10px;
  margin-bottom: 14px;
}
.wiki-search,
.wiki-type-select {
  width: 100%;
  min-height: 38px;
  padding: 8px 12px;
  border: 1px solid #cbd5e1;
  border-radius: 6px;
  background: #fff;
  color: #172033;
  font: inherit;
  font-size: 13px;
  outline: none;
}
.wiki-search:focus,
.wiki-type-select:focus { border-color: #2563eb; }
.wiki-layout {
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr);
  min-height: 560px;
  border: 1px solid #dbe3ee;
  border-radius: 8px;
  overflow: hidden;
  background: #fff;
}
.wiki-entity-nav {
  min-width: 0;
  max-height: 680px;
  overflow-y: auto;
  border-right: 1px solid #e2e8f0;
  background: #f8fafc;
}
.wiki-result-count {
  position: sticky;
  top: 0;
  z-index: 1;
  padding: 12px 14px;
  border-bottom: 1px solid #e2e8f0;
  background: #f8fafc;
  color: #64748b;
  font-size: 12px;
  font-weight: 600;
}
.wiki-entity-row {
  display: grid;
  grid-template-columns: 48px minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-height: 46px;
  padding: 8px 12px;
  border: 0;
  border-bottom: 1px solid #e8edf4;
  background: transparent;
  color: #334155;
  text-align: left;
  cursor: pointer;
}
.wiki-entity-row:hover { background: #eef4ff; }
.wiki-entity-row.active {
  background: #e8f0ff;
  box-shadow: inset 3px 0 #2563eb;
  color: #0f172a;
}
.wiki-entity-type,
.wiki-page-type {
  color: #2563eb;
  font-size: 11px;
  font-weight: 700;
}
.wiki-entity-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 600;
}
.wiki-inactive {
  color: #94a3b8;
  font-size: 10px;
}
.wiki-page {
  min-width: 0;
  padding: 20px 24px 28px;
}
.wiki-page-empty { min-height: 420px; justify-content: center; }
.wiki-page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding-bottom: 14px;
  border-bottom: 1px solid #e2e8f0;
}
.wiki-page-header h3 {
  margin: 4px 0 0;
  color: #0f172a;
  font-size: 22px;
  letter-spacing: 0;
}
.wiki-chapter-range {
  color: #64748b;
  font-size: 12px;
  white-space: nowrap;
}
.wiki-summary {
  margin: 16px 0 0;
  color: #334155;
  font-size: 14px;
  line-height: 1.75;
}
.wiki-aliases {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 14px;
}
.wiki-section-label {
  margin-right: 4px;
  color: #64748b;
  font-size: 12px;
}
.wiki-alias {
  padding: 2px 7px;
  border: 1px solid #cbd5e1;
  border-radius: 4px;
  color: #475569;
  font-size: 11px;
}
.wiki-section-block {
  margin-top: 24px;
}
.wiki-section-heading {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-bottom: 8px;
  border-bottom: 1px solid #e2e8f0;
}
.wiki-section-heading h4 {
  margin: 0;
  color: #172033;
  font-size: 14px;
}
.wiki-section-heading span {
  color: #64748b;
  font-size: 12px;
}
.wiki-empty-line {
  padding: 18px 0;
  color: #94a3b8;
  font-size: 13px;
}
.wiki-timeline {
  padding-top: 8px;
}
.wiki-timeline-item {
  display: grid;
  grid-template-columns: 12px minmax(0, 1fr);
  gap: 10px;
  padding: 10px 0;
}
.wiki-timeline-marker {
  width: 8px;
  height: 8px;
  margin-top: 6px;
  border: 2px solid #2563eb;
  border-radius: 50%;
  background: #fff;
}
.wiki-timeline-content { min-width: 0; }
.wiki-relation-main {
  display: flex;
  align-items: baseline;
  gap: 8px;
  color: #334155;
  font-size: 13px;
}
.wiki-relation-main strong { color: #0f172a; }
.wiki-relation-meta,
.wiki-event-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 4px;
  color: #64748b;
  font-size: 11px;
}
.wiki-current-tag { color: #15803d; font-weight: 600; }
.wiki-evidence {
  margin: 8px 0 0;
  padding: 8px 10px;
  border-left: 2px solid #94a3b8;
  background: #f8fafc;
  color: #475569;
  font-size: 12px;
  line-height: 1.6;
}
.wiki-evidence span {
  display: block;
  margin-top: 3px;
  color: #94a3b8;
  font-size: 10px;
}
.wiki-events {
  display: flex;
  flex-direction: column;
}
.wiki-event-row {
  display: grid;
  grid-template-columns: 76px minmax(0, 1fr);
  gap: 14px;
  padding: 12px 0;
  border-bottom: 1px solid #edf1f6;
}
.wiki-event-chapter {
  color: #2563eb;
  font-size: 12px;
  font-weight: 700;
}
.wiki-event-content { min-width: 0; }
.wiki-event-title {
  color: #172033;
  font-size: 13px;
  font-weight: 700;
}
.wiki-event-content p {
  margin: 5px 0 0;
  color: #475569;
  font-size: 13px;
  line-height: 1.65;
}
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

.settings-section {
  width: 100%;
  max-width: none;
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.settings-hero,
.settings-card {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 18px;
  box-shadow: 0 1px 3px rgba(15,23,42,0.04);
}
.settings-hero {
  display: flex;
  justify-content: space-between;
  gap: 24px;
  align-items: center;
  padding: 26px 28px;
}
.settings-hero h2 {
  color: #1e293b;
  font-size: 24px;
  margin-bottom: 8px;
}
.settings-hero p,
.settings-card-head p {
  color: #64748b;
  font-size: 13px;
  line-height: 1.7;
}
.token-summary {
  min-width: 180px;
  padding: 16px 18px;
  border-radius: 14px;
  background: #eff6ff;
  border: 1px solid #bfdbfe;
}
.token-summary span {
  display: block;
  color: #64748b;
  font-size: 12px;
  margin-bottom: 6px;
}
.token-summary strong {
  color: #1d4ed8;
  font-size: 24px;
}
.token-summary em {
  display: block;
  margin-top: 6px;
  color: #64748b;
  font-size: 12px;
  font-style: normal;
  line-height: 1.6;
}
.settings-grid {
  display: grid;
  grid-template-columns: minmax(320px, 0.9fr) minmax(520px, 1.4fr);
  gap: 20px;
}
.settings-card {
  padding: 22px;
}
.settings-card-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 18px;
}
.settings-card-head h3 {
  color: #1e293b;
  font-size: 17px;
  margin-bottom: 6px;
}
.settings-message {
  margin-top: 14px;
  padding: 10px 12px;
  border-radius: 10px;
  background: #f0fdf4;
  color: #15803d;
  font-size: 13px;
}
.empty-settings {
  padding: 28px 18px;
  color: #94a3b8;
  text-align: center;
  background: #f8fafc;
  border: 1px dashed #cbd5e1;
  border-radius: 12px;
  font-size: 13px;
}
.token-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 10px;
}
.token-pill {
  display: inline-flex;
  flex-direction: column;
  gap: 4px;
  padding: 10px 12px;
  border-radius: 12px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  color: #64748b;
  font-size: 12px;
}
.token-pill strong {
  color: #1e293b;
  font-size: 13px;
}
.token-pill em {
  color: #94a3b8;
  font-style: normal;
  line-height: 1.6;
}
.agent-token-row {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 10px;
  margin-top: 14px;
}
.agent-token-pill {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px;
  border-radius: 12px;
  background: #fff;
  border: 1px solid #e2e8f0;
  color: #64748b;
  font-size: 12px;
}
.agent-token-pill strong {
  color: #1e293b;
  font-size: 13px;
}
.agent-token-pill em {
  color: #94a3b8;
  font-style: normal;
  line-height: 1.6;
}
.model-settings-card {
  padding: 0;
  overflow: hidden;
}
.model-settings-head {
  padding: 22px;
  margin-bottom: 0;
  border-bottom: 1px solid #e2e8f0;
}
.provider-grid {
  display: grid;
  gap: 16px;
  padding: 18px 22px 22px;
}
.provider-card {
  border: 1px solid #e2e8f0;
  border-radius: 16px;
  background: #fff;
  overflow: hidden;
}
.provider-header {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
  padding: 18px;
  background: #f8fafc;
  border-bottom: 1px solid #e2e8f0;
}
.provider-name {
  color: #1e293b;
  font-size: 16px;
  font-weight: 700;
  margin-bottom: 4px;
}
.provider-url {
  color: #64748b;
  font-size: 12px;
  word-break: break-all;
}
.test-box {
  margin: 14px 18px 0;
  padding: 14px;
  border: 1px solid #fecaca;
  border-radius: 12px;
  background: #fef2f2;
  color: #991b1b;
  font-size: 13px;
}
.test-box.success {
  background: #f0fdf4;
  border-color: #bbf7d0;
  color: #166534;
}
.test-title {
  font-weight: 700;
  margin-bottom: 10px;
}
.model-check-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}
.model-check {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 10px;
  background: #fff;
  border: 1px solid #dcfce7;
  border-radius: 999px;
}
.model-list {
  display: grid;
  gap: 10px;
  padding: 18px;
}
.model-item {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: center;
  padding: 12px 14px;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  background: #fff;
}
.model-item.disabled {
  opacity: 0.62;
  background: #f8fafc;
}
.model-title {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #1e293b;
}
.model-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  color: #64748b;
  font-size: 12px;
  flex-wrap: wrap;
}
.default-badge,
.disabled-badge {
  display: inline-flex;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
}
.default-badge {
  background: #dbeafe;
  color: #1d4ed8;
}
.disabled-badge {
  background: #f1f5f9;
  color: #64748b;
}

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
.run-token-card {
  margin-top: 16px;
  padding: 14px;
  border: 1px solid #bfdbfe;
  border-radius: 10px;
  background: #eff6ff;
}
.run-token-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  color: #1e3a8a;
  font-size: 13px;
}
.run-token-head strong {
  font-size: 18px;
}
.run-token-meta {
  margin-top: 4px;
  color: #64748b;
  font-size: 12px;
}
.run-token-stages {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 8px;
  margin-top: 12px;
}
.run-token-stage {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  border-radius: 8px;
  background: #fff;
  color: #475569;
  font-size: 12px;
}
.run-token-stage strong {
  color: #2563eb;
}
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
.save-btn.sm { padding: 6px 12px; font-size: 12px; }

@media (max-width: 1100px) {
  .write-container {
    flex-direction: column;
  }
  .sidebar {
    width: 100%;
    flex: none;
    position: static;
    height: auto;
    overflow: visible;
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
  .nav-subitems {
    display: flex;
    flex-wrap: wrap;
    padding: 0;
    gap: 8px;
  }
  .nav-subitem {
    border: 1px solid #e2e8f0;
    border-radius: 999px;
    padding: 8px 14px;
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
  .control-group input,
  .control-group textarea {
    width: 100%;
  }
  .book-meta {
    gap: 16px;
  }
  .settings-hero,
  .provider-header,
  .model-item,
  .model-settings-head {
    flex-direction: column;
    align-items: stretch;
  }
  .settings-grid {
    grid-template-columns: 1fr;
  }
  .token-summary {
    min-width: 0;
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
  .wiki-layout {
    grid-template-columns: 220px minmax(0, 1fr);
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
  .wiki-toolbar,
  .wiki-layout {
    grid-template-columns: 1fr;
  }
  .wiki-entity-nav {
    max-height: 240px;
    border-right: 0;
    border-bottom: 1px solid #e2e8f0;
  }
  .wiki-page {
    padding: 16px;
  }
  .wiki-page-header,
  .wiki-relation-main {
    align-items: flex-start;
    flex-direction: column;
  }
  .wiki-event-row {
    grid-template-columns: 1fr;
    gap: 5px;
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
