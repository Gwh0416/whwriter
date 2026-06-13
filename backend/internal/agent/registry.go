package agent

type Registry struct {
	agents map[string]Agent
}

func NewRegistry() *Registry {
	r := &Registry{agents: make(map[string]Agent)}

	r.Register(NewArchitectAgent())
	r.Register(NewFoundationReviewerAgent())
	r.Register(NewPlannerAgent())
	r.Register(NewComposerAgent())
	r.Register(NewWriterAgent())
	r.Register(NewSettlerAgent())
	r.Register(NewContinuityAuditor())
	r.Register(NewReviserAgent())
	r.Register(NewPolisherAgent())
	r.Register(NewRadarAgent())

	return r
}

func (r *Registry) Register(a Agent) {
	r.agents[a.Name()] = a
}

func (r *Registry) Get(name string) (Agent, bool) {
	a, ok := r.agents[name]
	return a, ok
}

func (r *Registry) All() []Agent {
	agents := make([]Agent, 0, len(r.agents))
	for _, a := range r.agents {
		agents = append(agents, a)
	}
	return agents
}

func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.agents))
	for name := range r.agents {
		names = append(names, name)
	}
	return names
}
