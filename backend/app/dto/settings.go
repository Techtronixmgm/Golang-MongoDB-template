package dto

type PublicSettingsResponse struct {
	RegistrationEnabled bool `json:"registration_enabled"`
	TwoFactorEnabled    bool `json:"two_factor_enabled"`
}

type UpdateSettingsRequest struct {
	RegistrationEnabled *bool `json:"registration_enabled"`
	TwoFactorEnabled    *bool `json:"two_factor_enabled"`
}
