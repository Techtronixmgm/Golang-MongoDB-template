package dto

type PublicSettingsResponse struct {
	RegistrationEnabled bool   `json:"registration_enabled"`
	LoginHint           string `json:"login_hint"`
}

type MenuMaxDepthRequest struct {
	Top    *int `json:"top"`
	Left   *int `json:"left"`
	Bottom *int `json:"bottom"`
}

type MenuMaxDepthResponse struct {
	Top    int `json:"top"`
	Left   int `json:"left"`
	Bottom int `json:"bottom"`
}

type UpdateSettingsRequest struct {
	RegistrationEnabled   *bool                `json:"registration_enabled"`
	TwoFactorEnabled      *bool                `json:"two_factor_enabled"`
	LoginWithPrimaryEmail *bool                `json:"login_with_primary_email"`
	LoginWithUsername     *bool                `json:"login_with_username"`
	LoginWithPhone        *bool                `json:"login_with_phone"`
	LoginWithAltEmail     *bool                `json:"login_with_alt_email"`
	MenuMaxDepth          *MenuMaxDepthRequest `json:"menu_max_depth"`
}

type PrivateSettingsResponse struct {
	RegistrationEnabled   bool                 `json:"registration_enabled"`
	TwoFactorEnabled      bool                 `json:"two_factor_enabled"`
	LoginWithPrimaryEmail bool                 `json:"login_with_primary_email"`
	LoginWithUsername     bool                 `json:"login_with_username"`
	LoginWithPhone        bool                 `json:"login_with_phone"`
	LoginWithAltEmail     bool                 `json:"login_with_alt_email"`
	MenuMaxDepth          MenuMaxDepthResponse `json:"menu_max_depth"`
}
