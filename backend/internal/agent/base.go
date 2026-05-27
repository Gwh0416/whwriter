package agent

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Agent interface {
	Name() string
	SystemPrompt() string
}

type ChatCompleter interface {
	Chat(systemPrompt string, messages []Message, temperature float64) (string, error)
}
