package loyalty

// UserContext identifies who is making the request.
type UserContext struct {
	CustomerID    string
	BusinessName  string
	Role          string // "client" or "collaborator"
	UserID        string // client_id or collaborator_id
	Phone         string
	ActiveModules []string // e.g. ["earn_burn", "cashback"]
	CanSwitchRole bool     // true if user can toggle collaborator/client
}

// Command represents a menu selection or flow execution request.
type Command struct {
	ID          string
	UserContext UserContext
	Data        map[string]string // collected data from flow steps
}

// CommandOption represents a selectable item in a WhatsApp interactive list.
type CommandOption struct {
	ID          string
	Title       string
	Description string
}

// CommandResult is what a module returns after handling a command.
type CommandResult struct {
	Message     string            // text message to send back to user
	Options     []CommandOption   // if set, shown as interactive list after Message
	ListHeader  string            // header for the interactive list
	Data        map[string]interface{}
}

// MenuDefinition describes a single menu option for a role.
type MenuDefinition struct {
	ID          string // command identifier, e.g. "check_points"
	Title       string // display text in WhatsApp list
	Description string // secondary text in WhatsApp list
	Role        string // "client" or "collaborator"
}

// FlowDefinition describes a multi-step flow for a command.
type FlowDefinition struct {
	CommandID string
	Steps     []StepDefinition
}

// StepDefinition describes a single step in a flow.
type StepDefinition struct {
	ID         string                // e.g. "ask_otp"
	Prompt     string                // message sent to user
	Validate   func(string) error    // nil means always valid
	NeedsPhoto bool                  // if true, expects image message
	Key        string                // key to store collected data under
}
