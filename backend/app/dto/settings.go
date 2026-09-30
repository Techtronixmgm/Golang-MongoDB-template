package dto

type PublicSettingsResponse struct {
	RegistrationEnabled bool `json:"registration_enabled"`
}

type UpdateSettingsRequest struct {
	RegistrationEnabled   *bool `json:"registration_enabled"`
	TwoFactorEnabled      *bool `json:"two_factor_enabled"`
	LoginWithPrimaryEmail *bool `json:"login_with_primary_email" `
	LoginWithUsername     *bool `json:"login_with_username" `
	LoginWithPhone        *bool `json:"login_with_phone" `
	LoginWithAltEmail     *bool `json:"login_with_alt_email" `
}

type PrivateSettingsResponse struct {
	RegistrationEnabled   bool `json:"registration_enabled"`
	TwoFactorEnabled      bool `json:"two_factor_enabled"`
	LoginWithPrimaryEmail bool `json:"login_with_primary_email" `
	LoginWithUsername     bool `json:"login_with_username" `
	LoginWithPhone        bool `json:"login_with_phone" `
	LoginWithAltEmail     bool `json:"login_with_alt_email" `
}
