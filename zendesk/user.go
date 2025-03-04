package zendesk

import (
	"context"
	"fmt"
	"time"
)

type User struct {
	User struct {
		Id                   int64         `json:"id"`
		Url                  string        `json:"url"`
		Name                 string        `json:"name"`
		Email                string        `json:"email"`
		CreatedAt            time.Time     `json:"created_at"`
		UpdatedAt            time.Time     `json:"updated_at"`
		TimeZone             string        `json:"time_zone"`
		IanaTimeZone         string        `json:"iana_time_zone"`
		Phone                interface{}   `json:"phone"`
		SharedPhoneNumber    interface{}   `json:"shared_phone_number"`
		Photo                interface{}   `json:"photo"`
		LocaleId             int           `json:"locale_id"`
		Locale               string        `json:"locale"`
		OrganizationId       int64         `json:"organization_id"`
		Role                 string        `json:"role"`
		Verified             bool          `json:"verified"`
		ExternalId           interface{}   `json:"external_id"`
		Tags                 []interface{} `json:"tags"`
		Alias                string        `json:"alias"`
		Active               bool          `json:"active"`
		Shared               bool          `json:"shared"`
		SharedAgent          bool          `json:"shared_agent"`
		LastLoginAt          interface{}   `json:"last_login_at"`
		TwoFactorAuthEnabled interface{}   `json:"two_factor_auth_enabled"`
		Signature            interface{}   `json:"signature"`
		Details              string        `json:"details"`
		Notes                string        `json:"notes"`
		RoleType             interface{}   `json:"role_type"`
		CustomRoleId         interface{}   `json:"custom_role_id"`
		Moderator            bool          `json:"moderator"`
		TicketRestriction    string        `json:"ticket_restriction"`
		OnlyPrivateComments  bool          `json:"only_private_comments"`
		RestrictedAgent      bool          `json:"restricted_agent"`
		Suspended            bool          `json:"suspended"`
		DefaultGroupId       interface{}   `json:"default_group_id"`
		ReportCsv            bool          `json:"report_csv"`
		UserFields           struct {
			AddigyAgentid         interface{} `json:"addigy_agentid"`
			AddigyDeviceModelName interface{} `json:"addigy_device_model_name"`
			AddigyDeviceName      interface{} `json:"addigy_device_name"`
			AddigySerialNumber    interface{} `json:"addigy_serial_number"`
		} `json:"user_fields"`
	} `json:"user"`
}

func (c *Client) GetUser(ctx context.Context, userId int64) (*User, error) {
	url := fmt.Sprintf("%s/users/%d", c.baseUrl, userId)
	u := &User{}

	if err := c.apiRequest(ctx, "GET", url, nil, &u); err != nil {
		return nil, fmt.Errorf("an error occured getting the user: %w", err)
	}

	return u, nil
}
