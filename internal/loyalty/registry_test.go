package loyalty

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Fake Module ---

type fakeModule struct {
	name   string
	menus  map[string][]MenuDefinition
	flows  map[string]FlowDefinition
	result *CommandResult
	err    error
}

func (m *fakeModule) Name() string                              { return m.name }
func (m *fakeModule) Menus() map[string][]MenuDefinition        { return m.menus }
func (m *fakeModule) FlowDefinitions() map[string]FlowDefinition { return m.flows }
func (m *fakeModule) RegisterRoutes(rg *gin.RouterGroup)        {}
func (m *fakeModule) HandleCommand(ctx context.Context, cmd Command) (*CommandResult, error) {
	return m.result, m.err
}

func newFakeModule(name string) *fakeModule {
	return &fakeModule{
		name: name,
		menus: map[string][]MenuDefinition{
			"client": {
				{ID: name + "_cmd1", Title: "Cmd 1", Role: "client"},
				{ID: name + "_cmd2", Title: "Cmd 2", Role: "client"},
			},
			"collaborator": {
				{ID: name + "_collab1", Title: "Collab 1", Role: "collaborator"},
			},
		},
		flows: map[string]FlowDefinition{
			name + "_cmd1": {CommandID: name + "_cmd1", Steps: []StepDefinition{{ID: "s1", Key: "key1"}}},
		},
		result: &CommandResult{Message: name + " result"},
	}
}

// --- Tests ---

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	assert.NotNil(t, r)
	assert.Empty(t, r.modules)
	assert.Empty(t, r.commands)
}

func TestRegister(t *testing.T) {
	r := NewRegistry()
	m := newFakeModule("earn_burn")
	r.Register(m)

	assert.Len(t, r.modules, 1)
	assert.Contains(t, r.modules, "earn_burn")

	// Commands should be indexed
	assert.Equal(t, "earn_burn", r.commands["earn_burn_cmd1"])
	assert.Equal(t, "earn_burn", r.commands["earn_burn_cmd2"])
	assert.Equal(t, "earn_burn", r.commands["earn_burn_collab1"])
}

func TestAllMenus_SingleModule(t *testing.T) {
	r := NewRegistry()
	r.Register(newFakeModule("earn_burn"))

	clientMenus := r.AllMenus("client")
	assert.Len(t, clientMenus, 2)

	collabMenus := r.AllMenus("collaborator")
	assert.Len(t, collabMenus, 1)

	unknownMenus := r.AllMenus("admin")
	assert.Empty(t, unknownMenus)
}

func TestAllMenus_MultipleModules(t *testing.T) {
	r := NewRegistry()
	r.Register(newFakeModule("earn_burn"))
	r.Register(newFakeModule("cashback"))

	clientMenus := r.AllMenus("client")
	assert.Len(t, clientMenus, 4) // 2 + 2

	collabMenus := r.AllMenus("collaborator")
	assert.Len(t, collabMenus, 2) // 1 + 1
}

func TestFilteredMenus_WithActiveModules(t *testing.T) {
	r := NewRegistry()
	r.Register(newFakeModule("earn_burn"))
	r.Register(newFakeModule("cashback"))

	// Only earn_burn active
	menus := r.FilteredMenus("client", []string{"earn_burn"})
	assert.Len(t, menus, 2)

	// Only cashback active
	menus = r.FilteredMenus("client", []string{"cashback"})
	assert.Len(t, menus, 2)

	// Both active
	menus = r.FilteredMenus("client", []string{"earn_burn", "cashback"})
	assert.Len(t, menus, 4)
}

func TestFilteredMenus_EmptyActiveModules_FallsBack(t *testing.T) {
	r := NewRegistry()
	r.Register(newFakeModule("earn_burn"))
	r.Register(newFakeModule("cashback"))

	menus := r.FilteredMenus("client", []string{})
	assert.Len(t, menus, 4) // falls back to AllMenus
}

func TestGetFlowDefinition_Found(t *testing.T) {
	r := NewRegistry()
	r.Register(newFakeModule("earn_burn"))

	flow, ok := r.GetFlowDefinition("earn_burn_cmd1")
	assert.True(t, ok)
	assert.Equal(t, "earn_burn_cmd1", flow.CommandID)
	assert.Len(t, flow.Steps, 1)
}

func TestGetFlowDefinition_NotFound(t *testing.T) {
	r := NewRegistry()
	r.Register(newFakeModule("earn_burn"))

	flow, ok := r.GetFlowDefinition("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, flow)
}

func TestGetFlowDefinition_CommandWithoutFlow(t *testing.T) {
	r := NewRegistry()
	r.Register(newFakeModule("earn_burn"))

	// earn_burn_cmd2 is a menu command but has no flow definition
	flow, ok := r.GetFlowDefinition("earn_burn_cmd2")
	assert.False(t, ok)
	assert.Nil(t, flow)
}

func TestDispatch_Success(t *testing.T) {
	r := NewRegistry()
	m := newFakeModule("earn_burn")
	m.result = &CommandResult{Message: "Points: 42"}
	r.Register(m)

	result, err := r.Dispatch(context.Background(), Command{
		ID: "earn_burn_cmd1",
		UserContext: UserContext{CustomerID: "cust-1"},
	})

	require.NoError(t, err)
	assert.Equal(t, "Points: 42", result.Message)
}

func TestDispatch_UnknownCommand(t *testing.T) {
	r := NewRegistry()
	r.Register(newFakeModule("earn_burn"))

	_, err := r.Dispatch(context.Background(), Command{ID: "unknown"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

func TestContainsStr(t *testing.T) {
	assert.True(t, containsStr([]string{"a", "b", "c"}, "b"))
	assert.False(t, containsStr([]string{"a", "b", "c"}, "d"))
	assert.False(t, containsStr([]string{}, "a"))
}
