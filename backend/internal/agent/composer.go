package agent

type ComposerAgent struct{}

func NewComposerAgent() *ComposerAgent {
	return &ComposerAgent{}
}

func (a *ComposerAgent) Name() string { return "composer" }

func (a *ComposerAgent) SystemPrompt() string {
	return ""
}

const composerDescription = `编排师（Composer）不是 LLM agent——它是上下文组装器（context assembler）。
它从 chapter_memo 和 truth files 中组装 ContextPackage + RuleStack + ChapterTrace，
为 Writer 提供治理后的输入契约（governed input contract）。

职责：
1. 从 chapter_memo 提取 goal、threadRefs、hook 账本
2. 从 truth files 检索相关上下文（current_state、pending_hooks、chapter_summaries、character_matrix 等）
3. 构建 RuleStack（运行时规则栈，处理 L4→L3 override）
4. 构建 ChapterTrace（章节追踪信息）
5. 输出 ContextPackage（治理后的上下文包）

Composer 是纯逻辑组件，不调用 LLM。`
