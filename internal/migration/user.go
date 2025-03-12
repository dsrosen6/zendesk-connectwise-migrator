package migration

import (
	"context"
	"fmt"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/apis/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/apis/zendesk"
	"log/slog"
	"strings"
)

func (c *Client) MigrateUser(ctx context.Context, zendeskUser zendesk.User, psaCompanyId int) (*psa.Contact, error) {
	slog.Debug("migration.Client.MigrateUser called",
		"zendeskUserId", zendeskUser.User.Id,
		"psaCompanyId", psaCompanyId,
		"psaContactId", zendeskUser.User.UserFields.PSAContactId)

	// Check if the Zendesk user already has a PSA contact id value - if so, they've already been migrated by this tool
	if zendeskUser.User.UserFields.PSAContactId != 0 {
		slog.Info("MigrateUser: already migrated",
			"zendeskUserId", zendeskUser.User.Id,
			"psaContactId", zendeskUser.User.UserFields.PSAContactId)
		return nil, nil
	}

	contact, err := c.CwClient.GetContactByEmail(ctx, zendeskUser.User.Email)
	if err != nil {
		slog.Error("MigrateUser: error", "action", "client.CwClient.GetContactByEmail", "error", err)
		return nil, fmt.Errorf("error getting contact by email: %w", err)
	}

	var alreadyExisted bool
	if contact != nil {
		alreadyExisted = true
	} else {
		firstName, lastName := separateName(zendeskUser.User.Name)
		contact = &psa.Contact{
			FirstName: firstName,
			LastName:  lastName,
			Company: psa.Company{
				Id: psaCompanyId,
			},
			CommunicationItems: []psa.CommunicationItem{
				{
					Type:  psa.CommunicationItemType{Name: "Email"},
					Value: zendeskUser.User.Email,
				},
			},
		}

		contact, err = c.CwClient.PostContact(ctx, contact)
		if err != nil {
			slog.Error("MigrateUser: error", "action", "PostContact", "error", err)
			return nil, fmt.Errorf("error creating new contact: %w", err)
		}

		slog.Debug("MigrateUser: contact created in PSA", "contactId", contact.Id)
	}

	zendeskUser.User.UserFields.PSAContactId = contact.Id
	if err := c.ZendeskClient.UpdateUser(ctx, &zendeskUser); err != nil {
		return nil, fmt.Errorf("error updating zendesk user with PSA contact id: %w", err)
	}

	slog.Info("MigrateUser: complete",
		"zendeskUserId", zendeskUser.User.Id,
		"psaContactId", contact.Id,
		"alreadyExisted", alreadyExisted)
	return contact, nil
}

func separateName(name string) (string, string) {
	nameParts := strings.Split(name, " ")
	if len(nameParts) == 1 {
		return nameParts[0], ""
	}

	return nameParts[0], strings.Join(nameParts[1:], " ")
}
