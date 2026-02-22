package earnburn

import (
	"fmt"

	"github.com/theluisbolivar/fidel-quick/internal/loyalty"
)

// ClientMenus returns the menu options for clients.
func ClientMenus() []loyalty.MenuDefinition {
	return []loyalty.MenuDefinition{
		{ID: "check_points", Title: "Consultar puntos", Description: "Ver tu balance y movimientos", Role: "client"},
		{ID: "list_all_rewards", Title: "Ver recompensas", Description: "Catalogo de premios disponibles", Role: "client"},
		{ID: "redeem_rewards", Title: "Canjear recompensa", Description: "Canjear tus puntos por un premio", Role: "client"},
		{ID: "load_points_request", Title: "Cargar puntos", Description: "Generar codigo para el colaborador", Role: "client"},
		{ID: "submit_feedback", Title: "Dejar feedback", Description: "Enviar comentario al negocio", Role: "client"},
	}
}

// CollaboratorMenus returns the 5 menu options for collaborators.
func CollaboratorMenus() []loyalty.MenuDefinition {
	return []loyalty.MenuDefinition{
		{ID: "add_points", Title: "Agregar puntos", Description: "Acreditar puntos a un cliente", Role: "collaborator"},
		{ID: "list_points", Title: "Consultar puntos de cliente", Description: "Ver balance e historial", Role: "collaborator"},
		{ID: "confirm_redemption", Title: "Confirmar canje", Description: "Validar codigo de canje", Role: "collaborator"},
		{ID: "update_points", Title: "Corregir transaccion", Description: "Correccion dentro de 2h", Role: "collaborator"},
		{ID: "load_points_process", Title: "Procesar carga de puntos", Description: "Procesar codigo + ticket", Role: "collaborator"},
	}
}

// FlowDefs returns multi-step flow definitions for commands that need them.
func FlowDefs() map[string]loyalty.FlowDefinition {
	return map[string]loyalty.FlowDefinition{
		"add_points": {
			CommandID: "add_points",
			Steps: []loyalty.StepDefinition{
				{ID: "ask_otp", Prompt: "Escribe el codigo OTP del cliente:", Key: "otp", Validate: validateOTPFormat},
				{ID: "ask_photo", Prompt: "Envia la foto del ticket de compra:", Key: "photo", NeedsPhoto: true},
			},
		},
		"request_redemption": {
			CommandID: "request_redemption",
			Steps: []loyalty.StepDefinition{
				{ID: "confirm", Prompt: "Confirmas el canje? (Si/No):", Key: "confirm", Validate: validateYesNo},
			},
		},
		"confirm_redemption": {
			CommandID: "confirm_redemption",
			Steps: []loyalty.StepDefinition{
				{ID: "ask_code", Prompt: "Escribe el codigo de canje del cliente:", Key: "code", Validate: validateOTPFormat},
			},
		},
		"update_points": {
			CommandID: "update_points",
			Steps: []loyalty.StepDefinition{
				{ID: "ask_otp", Prompt: "Escribe el codigo OTP del cliente:", Key: "otp", Validate: validateOTPFormat},
				{ID: "select_tx", Prompt: "Selecciona la transaccion a corregir:", Key: "tx_id"},
				{ID: "new_amount", Prompt: "Escribe el monto correcto en puntos:", Key: "new_amount", Validate: validatePositiveNumber},
				{ID: "evidence", Prompt: "Envia foto o descripcion del error:", Key: "evidence"},
				{ID: "reason", Prompt: "Escribe un breve comentario:", Key: "reason"},
			},
		},
		"load_points_process": {
			CommandID: "load_points_process",
			Steps: []loyalty.StepDefinition{
				{ID: "ask_code", Prompt: "Escribe el codigo del cliente:", Key: "code", Validate: validateOTPFormat},
				{ID: "ask_photo", Prompt: "Envia la foto del ticket de compra:", Key: "photo", NeedsPhoto: true},
			},
		},
		"submit_feedback": {
			CommandID: "submit_feedback",
			Steps: []loyalty.StepDefinition{
				{ID: "ask_message", Prompt: "Escribe tu comentario:", Key: "message"},
			},
		},
		"list_points": {
			CommandID: "list_points",
			Steps: []loyalty.StepDefinition{
				{ID: "ask_otp", Prompt: "Escribe el codigo OTP del cliente:", Key: "otp", Validate: validateOTPFormat},
			},
		},
	}
}

func validateOTPFormat(input string) error {
	if len(input) != 6 {
		return fmt.Errorf("el codigo debe tener 6 caracteres")
	}
	return nil
}

func validateYesNo(input string) error {
	switch input {
	case "Si", "si", "SI", "No", "no", "NO":
		return nil
	}
	return fmt.Errorf("responde Si o No")
}

func validatePositiveNumber(input string) error {
	for _, c := range input {
		if c < '0' || c > '9' {
			return fmt.Errorf("escribe un numero valido")
		}
	}
	if input == "0" {
		return fmt.Errorf("el monto debe ser mayor a 0")
	}
	return nil
}
