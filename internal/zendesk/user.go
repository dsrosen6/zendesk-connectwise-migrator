package zendesk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

type Users struct {
	Users []User `json:"users"`
}

type User struct {
	User struct {
		Id                   int           `json:"id"`
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
		UserFields           UserFields    `json:"user_fields"`
	} `json:"user"`
}

type UserFields struct {
	PSAContactId int `json:"psa_contact"`
}

func (c *Client) UpdateUser(ctx context.Context, user *User) error {
	slog.Debug("UpdateUser called")
	url := fmt.Sprintf("%s/users/%d", c.baseUrl, user.User.Id)

	jsonBytes, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("marshaling user to json: %w", err)
	}

	body := bytes.NewReader(jsonBytes)

	if err := c.apiRequest(ctx, "PUT", url, body, nil); err != nil {
		return fmt.Errorf("an error occured updating the user: %w", err)
	}

	return nil
}
func (c *Client) GetUser(ctx context.Context, userId int64) (User, error) {
	slog.Debug("zendesk.Client.GetUser called", "userId", userId)
	url := fmt.Sprintf("%s/users/%d", c.baseUrl, userId)
	u := &User{}

	if err := c.apiRequest(ctx, "GET", url, nil, &u); err != nil {
		return User{}, fmt.Errorf("an error occured getting the user: %w", err)
	}

	return *u, nil
}
