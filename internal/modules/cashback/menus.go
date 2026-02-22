package cashback

import (
	"fmt"

	"github.com/theluisbolivar/fidel-quick/internal/loyalty"
)

// ClientMenus returns the menu options for clients.
func ClientMenus() []loyalty.MenuDefinition {
	return []loyalty.MenuDefinition{
		{ID: "cb_check_balance", Title: "Consultar saldo", Description: "Ver tu saldo y movimientos", Role: "client"},
		{ID: "cb_list_rewards", Title: "Ver beneficios", Description: "Catalogo de beneficios disponibles", Role: "client"},
		{ID: "cb_redeem", Title: "Canjear beneficio", Description: "Canjear tu cashback por un beneficio", Role: "client"},
		{ID: "cb_load_request", Title: "Cargar cashback", Description: "Generar codigo para el colaborador", Role: "client"},
	}
}

// CollaboratorMenus returns the 5 menu options for collaborators.
func CollaboratorMenus() []loyalty.MenuDefinition {
	return []loyalty.MenuDefinition{
		{ID: "cb_add_cashback", Title: "Agregar cashback", Description: "Acreditar cashback a un cliente", Role: "collaborator"},
		{ID: "cb_list_balance", Title: "Consultar saldo de cliente", Description: "Ver saldo e historial", Role: "collaborator"},
		{ID: "cb_confirm_redemption", Title: "Confirmar canje", Description: "Validar codigo de canje", Role: "collaborator"},
		{ID: "cb_update_cashback", Title: "Corregir transaccion", Description: "Correccion dentro de 2h", Role: "collaborator"},
		{ID: "cb_load_process", Title: "Procesar carga cashback", Description: "Procesar codigo + ticket", Role: "collaborator"},
	}
}

// FlowDefs returns multi-step flow definitions for commands that need them.
func FlowDefs() map[string]loyalty.FlowDefinition {
	return map[string]loyalty.FlowDefinition{
		"cb_add_cashback": {
			CommandID: "cb_add_cashback",
			Steps: []loyalty.StepDefinition{
				{ID: "ask_otp", Prompt: "Escribe el codigo OTP del cliente:", Key: "otp", Validate: validateOTPFormat},
				{ID: "ask_photo", Prompt: "Envia la foto del ticket de compra:", Key: "photo", NeedsPhoto: true},
			},
		},
		"cb_request_redemption": {
			CommandID: "cb_request_redemption",
			Steps: []loyalty.StepDefinition{
				{ID: "confirm", Prompt: "Confirmas el canje? (Si/No):", Key: "confirm", Validate: validateYesNo},
			},
		},
		"cb_confirm_redemption": {
			CommandID: "cb_confirm_redemption",
			Steps: []loyalty.StepDefinition{
				{ID: "ask_code", Prompt: "Escribe el codigo de canje del cliente:", Key: "code", Validate: validateOTPFormat},
			},
		},
		"cb_update_cashback": {
			CommandID: "cb_update_cashback",
			Steps: []loyalty.StepDefinition{
				{ID: "ask_otp", Prompt: "Escribe el codigo OTP del cliente:", Key: "otp", Validate: validateOTPFormat},
				{ID: "select_tx", Prompt: "Selecciona la transaccion a corregir:", Key: "tx_id"},
				{ID: "new_amount", Prompt: "Escribe el nuevo monto de factura:", Key: "new_amount", Validate: validatePositiveDecimal},
				{ID: "evidence", Prompt: "Envia foto o descripcion del error:", Key: "evidence"},
				{ID: "reason", Prompt: "Escribe un breve comentario:", Key: "reason"},
			},
		},
		"cb_load_process": {
			CommandID: "cb_load_process",
			Steps: []loyalty.StepDefinition{
				{ID: "ask_code", Prompt: "Escribe el codigo del cliente:", Key: "code", Validate: validateOTPFormat},
				{ID: "ask_photo", Prompt: "Envia la foto del ticket de compra:", Key: "photo", NeedsPhoto: true},
			},
		},
		"cb_list_balance": {
			CommandID: "cb_list_balance",
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

func validatePositiveDecimal(input string) error {
	hasDigit := false
	hasDot := false
	for _, c := range input {
		if c >= '0' && c <= '9' {
			hasDigit = true
		} else if c == '.' && !hasDot {
			hasDot = true
		} else {
			return fmt.Errorf("escribe un monto valido (ej: 150.00)")
		}
	}
	if !hasDigit {
		return fmt.Errorf("escribe un monto valido (ej: 150.00)")
	}
	if input == "0" || input == "0.00" || input == "0.0" {
		return fmt.Errorf("el monto debe ser mayor a 0")
	}
	return nil
}
