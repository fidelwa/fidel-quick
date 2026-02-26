package earnburn

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientMenus(t *testing.T) {
	menus := ClientMenus()

	assert.Len(t, menus, 5)

	ids := make(map[string]bool)
	for _, m := range menus {
		ids[m.ID] = true
		assert.Equal(t, "client", m.Role)
		assert.NotEmpty(t, m.Title)
		assert.NotEmpty(t, m.Description)
	}

	assert.True(t, ids["check_points"])
	assert.True(t, ids["list_all_rewards"])
	assert.True(t, ids["redeem_rewards"])
	assert.True(t, ids["load_points_request"])
	assert.True(t, ids["submit_feedback"])
}

func TestCollaboratorMenus(t *testing.T) {
	menus := CollaboratorMenus()

	assert.Len(t, menus, 5)

	ids := make(map[string]bool)
	for _, m := range menus {
		ids[m.ID] = true
		assert.Equal(t, "collaborator", m.Role)
	}

	assert.True(t, ids["add_points"])
	assert.True(t, ids["list_points"])
	assert.True(t, ids["confirm_redemption"])
	assert.True(t, ids["update_points"])
	assert.True(t, ids["load_points_process"])
}

func TestFlowDefs(t *testing.T) {
	defs := FlowDefs()

	assert.Contains(t, defs, "add_points")
	assert.Contains(t, defs, "request_redemption")
	assert.Contains(t, defs, "confirm_redemption")
	assert.Contains(t, defs, "update_points")
	assert.Contains(t, defs, "load_points_process")
	assert.Contains(t, defs, "submit_feedback")
	assert.Contains(t, defs, "list_points")

	// Verify add_points flow has 2 steps
	addPoints := defs["add_points"]
	assert.Equal(t, "add_points", addPoints.CommandID)
	assert.Len(t, addPoints.Steps, 2)
	assert.Equal(t, "otp", addPoints.Steps[0].Key)
	assert.Equal(t, "photo", addPoints.Steps[1].Key)
	assert.True(t, addPoints.Steps[1].NeedsPhoto)

	// Verify update_points has 5 steps
	updatePoints := defs["update_points"]
	assert.Len(t, updatePoints.Steps, 5)
}

func TestValidateOTPFormat(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"ABC123", true},
		{"ABCDEF", true},
		{"12345", false},   // too short
		{"ABCDEFG", false}, // too long
		{"", false},
	}

	for _, tt := range tests {
		err := validateOTPFormat(tt.input)
		if tt.valid {
			assert.NoError(t, err, "input: %s", tt.input)
		} else {
			assert.Error(t, err, "input: %s", tt.input)
		}
	}
}

func TestValidateYesNo(t *testing.T) {
	valid := []string{"Si", "si", "SI", "No", "no", "NO"}
	for _, v := range valid {
		assert.NoError(t, validateYesNo(v))
	}

	invalid := []string{"Maybe", "yes", "Sí", "", "n"}
	for _, v := range invalid {
		assert.Error(t, validateYesNo(v))
	}
}

func TestValidatePositiveNumber(t *testing.T) {
	assert.NoError(t, validatePositiveNumber("1"))
	assert.NoError(t, validatePositiveNumber("100"))
	assert.NoError(t, validatePositiveNumber("999"))

	assert.Error(t, validatePositiveNumber("0"))
	assert.Error(t, validatePositiveNumber("abc"))
	assert.Error(t, validatePositiveNumber("-1"))
	assert.Error(t, validatePositiveNumber("1.5"))
}
