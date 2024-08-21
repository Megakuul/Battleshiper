package pipeline

type EventOptions struct {
	EventBus   string
	Source     string
	Action     string
	TicketOpts *TicketOptions
}

func CreateEventOptions(eventbus, source, action string, ticketOpts *TicketOptions) *EventOptions {
	return &EventOptions{
		EventBus:   eventbus,
		Source:     source,
		Action:     action,
		TicketOpts: ticketOpts,
	}
}
