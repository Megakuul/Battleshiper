package pipeline

type EventOptions struct {
	Source     string
	Action     string
	TicketOpts *TicketOptions
}

func CreateEventOptions(source, action string, ticketOpts *TicketOptions) *EventOptions {
	return &EventOptions{
		Source:     source,
		Action:     action,
		TicketOpts: ticketOpts,
	}
}
